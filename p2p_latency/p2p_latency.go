package p2p_latency

import (
	"cgolatencytest/config"
	"cgolatencytest/myutils"
	"cgolatencytest/p2p_base"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type P2PLatencyNode struct {
	Node      *p2p_base.P2PBaseNode
	BnLatency *BnLatencyResult

	//目标节点平均网络延迟
	NodeAvgLatencyMap *myutils.MySyncMap[string, int64] // nodeName -> avgLatency

	//目标节点币安延迟信息
	NodeBnLatencyMap *myutils.MySyncMap[string, BnLatencyResult] // nodeName -> bnLatency
}

var thisP2PLatencyNode *P2PLatencyNode

func GetP2PLatencyNode() *P2PLatencyNode {
	return thisP2PLatencyNode
}

func init() {
	//初始化本节点，单例模式
	myIP, err := getMyIP()
	if err != nil {
		log.Errorf("获取 IP 失败：%v\n", err)
		return
	}

	//节点名及获取种子字符串
	nodeName := getNodeName(myIP, 10000)
	//种子字符串默认为节点名
	seedStr := nodeName

	//获取其他所有节点
	nodeNameList := config.GetConfigSlice("p2p_nodes")

	//注册其他所有节点
	targetPeers := make(map[string]string)
	for _, nodeName := range nodeNameList {
		seed := nodeName
		host, port, err := net.SplitHostPort(nodeName)
		if err != nil {
			log.Errorf("获取节点地址失败：%s - %v\n", nodeName, err)
			continue
		}

		peerId, err := p2p_base.GetIDFromSeed(seed)
		if err != nil {
			log.Errorf("获取节点PeerId失败: %s - %v", seed, err)
			continue
		}

		targetPeers[nodeName] = fmt.Sprintf("/ip4/%s/udp/%s/quic-v1/p2p/%s", host, port, peerId)
	}

	thisNode, err := p2p_base.NewP2PBaseNode(nodeName, 10000, seedStr, targetPeers)
	if err != nil {
		log.Errorf("创建p2p节点失败：%v\n", err)
		return
	}
	thisP2PLatencyNode = newP2PLatencyNode(thisNode)

	go func() {
		//持续读取消息通道
		for msg := range thisNode.MsgChan() {
			log.Infof("[%s]收到消息: %s, 消息延迟: %dus", thisNode.PeerName, msg.MsgData, (time.Now().UnixNano()-msg.TimestampNano)/1000)
			//判断是否符合请求格式
			inTimestamp := time.Now().UnixNano()
			_ = inTimestamp
			var p2pMsg P2PMessage
			err := json.Unmarshal([]byte(msg.MsgData), &p2pMsg)
			if err != nil {
				log.Errorf("Unmarshal error: %v", err)
				continue
			}

			if p2pMsg.IsReq {
				//处理请求
				switch p2pMsg.Req.ReqType {
				case P2PReqTypeLatency:
					//远程节点传入延迟请求捕获并存入
					err = thisP2PLatencyNode.handleAvgLatencyMsg(p2pMsg, msg.FromPeerName)
				case P2PReqTypeBnLatency:
					//远程节点发起币安延迟请求直接返回币安延迟信息
					err = thisP2PLatencyNode.handleBnLatencyMsgReq(p2pMsg, msg.FromPeerName, inTimestamp)
				default:
					log.Errorf("P2P节点[%s]不支持的请求类型: %s", msg.FromPeerName, p2pMsg.Req.ReqType)
				}
			} else {
				//捕获应答
				switch p2pMsg.Res.ReqType {
				case P2PReqTypeBnLatency:
					//远程节点响应返回币安延迟信息，存入缓存
					err = thisP2PLatencyNode.handleBnLatencyMsgRes(p2pMsg, msg.FromPeerName)
				default:
					log.Errorf("P2P节点[%s]不支持的应答类型: %s", msg.FromPeerName, p2pMsg.Res.ReqType)
				}
			}
			if err != nil {
				log.Error(err)
			}
		}
	}()

	go func() {
		//每分钟广播一次延迟消息
		for {
			time.Sleep(time.Minute * 1)
			err := thisP2PLatencyNode.broadcastAvgLatencyMsg()
			if err != nil {
				log.Error(err)
			}
		}
	}()

	go func() {
		//每分钟计算一次币安延迟信息
		for {
			time.Sleep(time.Minute * 1)
			err := thisP2PLatencyNode.refreshBnLatency()
			if err != nil {
				log.Error(err)
				continue
			}
			err = thisP2PLatencyNode.broadcastBnLatencyMsg()
			if err != nil {
				log.Error(err)
				continue
			}
		}
	}()

	//刷新一次币安延迟
	thisP2PLatencyNode.refreshBnLatency()

}

func newP2PLatencyNode(node *p2p_base.P2PBaseNode) *P2PLatencyNode {
	return &P2PLatencyNode{
		Node:              node,
		BnLatency:         &BnLatencyResult{},
		NodeAvgLatencyMap: myutils.GetPointer(myutils.NewMySyncMap[string, int64]()),
		NodeBnLatencyMap:  myutils.GetPointer(myutils.NewMySyncMap[string, BnLatencyResult]()),
	}
}

func getNodeName(ip string, port int) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func getMyIP() (string, error) {
	resp, err := http.Get("http://icanhazip.com")
	if err != nil {
		log.Errorf("获取 IP 失败：%v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("读取响应失败：%v\n", err)
		return "", err
	}

	// 移除可能存在的换行符或空格
	ip := strings.TrimSpace(string(body))

	log.Debugf("此公网 IP 地址是: %s\n", ip)

	return ip, nil
}
