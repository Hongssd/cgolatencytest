package p2p_base

import (
	"cgolatencytest/myutils"
	"context"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// 管理p2p连接的链接池
type P2pConnectionManager struct {
	BaseNode          *P2PBaseNode
	TargetPeerAddrMap map[string]string         // name -> full addr
	ThisStreamMap     map[string]*P2pConnection // full addr -> stream
	RemoteStreamMap   map[string]*P2pConnection // remote peerId -> stream
}

type P2pConnection struct {
	PeerId        peer.ID
	PeerAddr      string
	Stream        network.Stream
	WaitAckMsgMap *myutils.MySyncMap[string, *chan bool] // 等待确认的消息通道 msgId -> 确认通道
	MainAckChan   *chan Message                          // 异步消息确认
}

// 创建连接管理器
func NewP2pConnectionManager(baseNode *P2PBaseNode, targetPeerAddrs map[string]string) *P2pConnectionManager {
	manager := &P2pConnectionManager{
		BaseNode:          baseNode,
		TargetPeerAddrMap: make(map[string]string),
		ThisStreamMap:     make(map[string]*P2pConnection),
		RemoteStreamMap:   make(map[string]*P2pConnection),
	}

	for name, addr := range targetPeerAddrs {
		//防止自链接
		if name == baseNode.PeerName {
			continue
		}
		manager.TargetPeerAddrMap[name] = addr
	}

	return manager
}

func (manager *P2pConnectionManager) NewP2PConnection(peerId peer.ID, targetPeerAddr string, stream network.Stream) *P2pConnection {

	connection := &P2pConnection{
		PeerId:   peerId,
		PeerAddr: targetPeerAddr,
		Stream:   stream,
	}

	mainAckChan := make(chan Message, 100)
	waitAckMsgMap := myutils.GetPointer(myutils.NewMySyncMap[string, *chan bool]())

	connection.MainAckChan = &mainAckChan
	connection.WaitAckMsgMap = waitAckMsgMap
	//异步消息确认
	go func() {
		for {
			msg := <-*connection.MainAckChan
			// log.Infof("[%s] 主通道收到消息确认: %s", manager.BaseNode.PeerName, msg.AckMsgId)
			if ch, ok := connection.WaitAckMsgMap.Load(msg.AckMsgId); ok {
				*ch <- true
				// log.Infof("[%s] 主通道确认消息转发: %s", manager.BaseNode.PeerName, msg.AckMsgId)
			}
		}
	}()
	return connection
}

// 连接到目标节点 并建立新链接
func (manager *P2pConnectionManager) ConnectToPeer(targetName string) error {
	targetPeerAddr, ok := manager.TargetPeerAddrMap[targetName]
	if !ok {
		err := fmt.Errorf("%s 地址不存在", targetName)
		log.Errorf("链接到目标节点错误: %v", err)
		return err
	}

	p2pAddr, err := multiaddr.NewMultiaddr(targetPeerAddr)
	if err != nil {
		log.Errorf("地址转换错误: %v", err)
		return err
	}
	peer, err := peer.AddrInfoFromP2pAddr(p2pAddr)
	if err != nil {
		log.Errorf("地址转换错误: %v", err)
		return err
	}

	if err := manager.BaseNode.Host.Connect(context.Background(), *peer); err != nil {
		log.Errorf("链接节点错误: %v", err)
		return err
	}
	log.Infof("[%s]已连接到节点: [%s]%s\n", manager.BaseNode.PeerName, targetName, targetPeerAddr)

	stream, err := manager.BaseNode.Host.NewStream(context.Background(), peer.ID, ProtocolID)
	if err != nil {
		log.Errorf("新建流错误: %v", err)
		return err
	}

	connection := manager.NewP2PConnection(peer.ID, targetPeerAddr, stream)

	log.Infof("[%s]已建立新流: [%s]%s", manager.BaseNode.PeerName, targetName, connection.PeerAddr)

	manager.ThisStreamMap[targetName] = connection

	go manager.BaseNode.handleConnectionMsg(connection)

	return nil
}

// 获取目标节点的链接
func (manager *P2pConnectionManager) GetConnection(targetName string) (*P2pConnection, error) {
	var err error
	connection, ok := manager.ThisStreamMap[targetName]
	if !ok || connection.Stream.Conn().IsClosed() {
		// 链接不存在或已关闭，建立新链接
		err = manager.ConnectToPeer(targetName)
		if err != nil {
			return nil, err
		}
		connection = manager.ThisStreamMap[targetName]
	}
	return connection, nil
}

// 捕获远程节点的链接
func (manager *P2pConnectionManager) HandleStream(stream network.Stream) *P2pConnection {
	remotePeerId := stream.Conn().RemotePeer()
	connection := manager.NewP2PConnection(remotePeerId, stream.Conn().RemoteMultiaddr().String(), stream)
	manager.RemoteStreamMap[remotePeerId.String()] = connection
	log.Infof("[%s]已捕获远程节点链接: [%s]%s", manager.BaseNode.PeerName, remotePeerId.String(), stream.Conn().RemoteMultiaddr().String())
	return connection
}

// 获取远程节点的链接
func (manager *P2pConnectionManager) GetRemoteConnection(remotePeerId peer.ID) (*P2pConnection, error) {
	connection, ok := manager.RemoteStreamMap[remotePeerId.String()]
	if !ok || connection.Stream.Conn().IsClosed() {
		return nil, fmt.Errorf("远程节点链接不存在或已关闭")
	}
	return connection, nil
}

// 写入数据
func (conn *P2pConnection) Send(data []byte) error {
	// 准备一个包含数据长度的头部(4字节)
	dataLen := len(data)
	header := make([]byte, 4)
	header[0] = byte(dataLen >> 24)
	header[1] = byte(dataLen >> 16)
	header[2] = byte(dataLen >> 8)
	header[3] = byte(dataLen)

	// 先写入长度头部
	_, err := conn.Stream.Write(header)
	if err != nil {
		return err
	}

	// 再写入实际数据
	_, err = conn.Stream.Write(data)
	return err
}

// 读取数据
func (conn *P2pConnection) Recv() ([]byte, error) {
	// 读取4字节的长度头部
	header := make([]byte, 4)
	_, err := io.ReadFull(conn.Stream, header)
	if err != nil {
		if err != io.EOF {
			log.Errorf("读取消息长度错误: %v", err)
		}
		return []byte{}, err
	}

	// 解析消息长度
	dataLen := int(header[0])<<24 | int(header[1])<<16 | int(header[2])<<8 | int(header[3])

	// 合理性检查
	if dataLen <= 0 || dataLen > 10*1024*1024 { // 最大限制为10MB
		return []byte{}, fmt.Errorf("无效的消息长度: %d", dataLen)
	}

	// 根据长度分配精确大小的缓冲区
	buf := make([]byte, dataLen)
	_, err = io.ReadFull(conn.Stream, buf)
	if err != nil {
		if err != io.EOF {
			log.Errorf("读取消息内容错误: %v", err)
		}
		return []byte{}, err
	}

	return buf, nil
}

// 关闭链接
func (conn *P2pConnection) Close() error {
	return conn.Stream.Close()
}
