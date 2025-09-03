package p2p_base

import (
	"cgolatencytest/mylog"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

var log = mylog.Log
var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	ProtocolID = "/latency-p2p/1.0.0"
)

type P2PBaseNode struct {
	Host              host.Host
	PeerName          string //名称
	ConnectionManager *P2pConnectionManager
	msgChan           chan Message
}

type seedReader struct {
	seed  []byte
	index int
}

func newSeedReader(seed string) *seedReader {
	// 使用SHA256来处理种子，确保有足够的熵
	hash := sha256.Sum256([]byte(seed))
	return &seedReader{
		seed:  hash[:],
		index: 0,
	}
}

func (r *seedReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.seed) {
		// 如果已经用完种子数据，重新生成新的哈希
		newHash := sha256.Sum256(r.seed)
		r.seed = newHash[:]
		r.index = 0
	}

	n = copy(p, r.seed[r.index:])
	r.index += n
	return n, nil
}

func GetIDFromSeed(seedStr string) (string, error) {
	reader := newSeedReader(seedStr)
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, reader)
	if err != nil {
		log.Errorf("使用种子生成私钥失败: %v", err)
		return "", err
	}

	// 打印节点ID，方便用户确认
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Errorf("生成节点ID失败: %v", err)
		return "", err
	}
	// log.Infof("使用种子'%s'生成的节点ID: %s\n", seedStr, pid.String())
	return pid.String(), nil
}

func NewP2PBaseNode(peerName string, port int, seedStr string, targetPeers map[string]string) (*P2PBaseNode, error) {

	var priv crypto.PrivKey
	var err error

	if seedStr != "" {
		// 使用种子字符串生成私钥
		reader := newSeedReader(seedStr)
		priv, _, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, reader)
		if err != nil {
			log.Errorf("使用种子生成私钥失败: %v", err)
			return nil, err
		}

		// 打印节点ID，方便用户确认
		pid, err := peer.IDFromPrivateKey(priv)
		if err != nil {
			log.Errorf("生成节点ID失败: %v", err)
			return nil, err
		}
		log.Infof("使用种子'%s'生成的节点ID: %s\n", seedStr, pid.String())

		// 导出私钥
		privKeyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			log.Errorf("导出私钥失败: %v", err)
			return nil, err
		}

		log.Infof("对应的私钥（十六进制）: %s\n", hex.EncodeToString(privKeyBytes))

	} else {
		// 随机生成新的私钥
		priv, _, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
		if err != nil {
			log.Fatal(err)
		}

		// 导出私钥并打印，方便用户保存
		privKeyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			log.Fatalf("导出私钥失败: %v", err)
		}
		fmt.Printf("随机生成的私钥（十六进制）: %s\n", hex.EncodeToString(privKeyBytes))
	}

	// 使用 QUIC 传输协议创建节点
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port)),
		libp2p.DefaultSecurity,
	)
	if err != nil {
		log.Errorf("创建节点失败: %v", err)
		return nil, err
	}

	baseNode := &P2PBaseNode{
		Host:     h,
		PeerName: peerName,
		msgChan:  make(chan Message, 100),
	}

	connectionManager := NewP2pConnectionManager(baseNode, targetPeers)

	baseNode.ConnectionManager = connectionManager

	// 设置流处理器
	h.SetStreamHandler(ProtocolID, baseNode.GetHandleRemoteStream())

	// 打印节点信息
	for _, addr := range h.Addrs() {
		fmt.Printf("监听地址: %s/p2p/%s\n", addr, h.ID().String())
	}

	return baseNode, nil
}

func (baseNode *P2PBaseNode) handleConnectionMsg(connection *P2pConnection) {
	for {
		data, err := connection.Recv()
		if err != nil {
			log.Errorf("接收消息失败: %v", err)
			break
		}
		var msg Message
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Errorf("解析消息失败: %v", err)
			continue
		}
		// log.Infof("[%s] 收到消息: %s", baseNode.PeerName, msg.MsgId)
		if msg.MsgType == MsgTypeAck {
			//转发消息确认
			// log.Infof("[%s] 转发消息确认: %s", baseNode.PeerName, msg.MsgId)
			*connection.MainAckChan <- msg
			continue
		}
		//转发消息
		baseNode.msgChan <- msg

		// 消息确认发送到远端
		if msg.NeedsAck {
			ackMsg := NewAckMessage(baseNode.Host.ID(), connection.PeerId, baseNode.PeerName, msg.FromPeerName, msg.MsgId)
			ackMsgBytes, err := json.Marshal(ackMsg)
			if err != nil {
				log.Errorf("序列化消息失败: %v", err)
				continue
			}
			err = connection.Send(ackMsgBytes)
			if err != nil {
				log.Errorf("发送消息失败: %v", err)
				continue
			}
			// log.Infof("[%s] 发送消息确认: %v", baseNode.PeerName, ackMsg)
		}
	}
}

// 获取流处理器
func (baseNode *P2PBaseNode) GetHandleRemoteStream() network.StreamHandler {
	return func(stream network.Stream) {
		//捕获远端创建的流构建连接
		connection := baseNode.ConnectionManager.HandleStream(stream)
		//持续接收消息并处理
		go baseNode.handleConnectionMsg(connection)
	}
}

func (baseNode *P2PBaseNode) SendMsg(targetName string, msgData string, needsAck bool) error {
	connection, err := baseNode.ConnectionManager.GetConnection(targetName)
	if err != nil {
		return err
	}
	msg := NewDataMessage(baseNode.Host.ID(), connection.PeerId, baseNode.PeerName, targetName, msgData, needsAck)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var waitAckChan chan bool
	//消息发送前建立等待确认通道
	if needsAck {
		waitAckChan = make(chan bool)
		connection.WaitAckMsgMap.Store(msg.MsgId, &waitAckChan)
		defer func() {
			close(waitAckChan)
			connection.WaitAckMsgMap.Delete(msg.MsgId)
		}()
	}

	if err := connection.Send(msgBytes); err != nil {
		return err
	}

	//消息确认
	if needsAck {
		// log.Infof("[%s] 等待消息确认: %s", baseNode.PeerName, msg.MsgId)
		select {
		case <-waitAckChan:
			// log.Infof("[%s] 收到消息确认: %s", baseNode.PeerName, msg.MsgId)
			return nil
		case <-time.After(time.Second * 5):
			return fmt.Errorf("消息确认超时")
		}
	}
	return nil
}

func (baseNode *P2PBaseNode) BroadcastMsg(msgData string, needsAck bool) error {
	var wg sync.WaitGroup
	for targetName := range baseNode.ConnectionManager.TargetPeerAddrMap {
		wg.Add(1)
		go func(targetName string) {
			defer wg.Done()
			err := baseNode.SendMsg(targetName, msgData, needsAck)
			if err != nil {
				log.Errorf("发送消息失败: %v", err)
			}
		}(targetName)
	}
	wg.Wait()
	return nil
}

func (baseNode *P2PBaseNode) MsgChan() <-chan Message {
	return baseNode.msgChan
}
