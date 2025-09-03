package p2p_latency

import (
	"time"

	"github.com/google/uuid"
)

// 刷新币安延迟信息
func (n *P2PLatencyNode) refreshBnLatency() error {
	result, err := TestBinanceHttpAndWsLatency()
	if err != nil {
		return err
	}

	n.BnLatency = result

	return nil
}

// 广播币安延迟消息给所有远程P2P节点
func (n *P2PLatencyNode) broadcastBnLatencyMsg() error {
	//广播延迟信息
	bnLatencyData, err := json.Marshal(n.BnLatency)
	if err != nil {
		return err
	}

	//构建返回消息
	p2pMsgRes := P2PMessage{
		IsReq: false,
		Res: P2PRes{
			ReqId:        uuid.New().String(),
			ReqType:      P2PReqTypeBnLatency,
			ResData:      string(bnLatencyData),
			ErrCode:      0,
			ErrMsg:       "",
			InTimestamp:  time.Now().UnixNano(),
			OutTimestamp: time.Now().UnixNano(),
		},
	}
	log.Infof("本地节点发送执行结果消息：%+v", p2pMsgRes)

	p2pMsgResBytes, err := json.Marshal(p2pMsgRes)
	if err != nil {
		log.Errorf("Marshal error: %v", err)
		return err
	}
	//广播币安延迟消息给所有远程P2P节点
	err = n.Node.BroadcastMsg(string(p2pMsgResBytes), true)
	if err != nil {
		log.Errorf("SendResMsg error: %v", err)
		return err
	}
	return nil
}


func (n *P2PLatencyNode) handleBnLatencyMsgReq(p2pMsg P2PMessage, fromPeerName string, inTimestamp int64) error {

	bnLatencyData, err := json.Marshal(n.BnLatency)
	if err != nil {
		return err
	}

	//构建返回消息
	p2pMsgRes := P2PMessage{
		IsReq: false,
		Req:   p2pMsg.Req,
		Res: P2PRes{
			ReqId:        p2pMsg.Req.ReqId,
			ReqType:      p2pMsg.Req.ReqType,
			ResData:      string(bnLatencyData),
			ErrCode:      0,
			ErrMsg:       "",
			InTimestamp:  inTimestamp,
			OutTimestamp: time.Now().UnixNano(),
		},
	}

	log.Infof("本地节点发送执行结果消息：%+v", p2pMsgRes)

	p2pMsgResBytes, err := json.Marshal(p2pMsgRes)
	if err != nil {
		log.Errorf("Marshal error: %v", err)
		return err
	}
	//发送返回消息给远程P2P节点
	err = n.Node.SendMsg(fromPeerName, string(p2pMsgResBytes), true)
	if err != nil {
		log.Errorf("SendResMsg error: %v", err)
		return err
	}
	return nil
}

func (n *P2PLatencyNode) handleBnLatencyMsgRes(p2pMsg P2PMessage, fromPeerName string) error {
	targetBnLatency := BnLatencyResult{}
	err := json.Unmarshal([]byte(p2pMsg.Res.ResData), &targetBnLatency)
	if err != nil {
		return err
	}
	n.NodeBnLatencyMap.Store(fromPeerName, targetBnLatency)
	return nil
}

// 通过节点名获取币安延迟信息
func (n *P2PLatencyNode) GetBnLatencyFromNodeName(nodeName string) BnLatencyResult {
	if nodeName == n.Node.PeerName {
		return *n.BnLatency
	}
	bnLatency, ok := n.NodeBnLatencyMap.Load(nodeName)
	if !ok {
		return BnLatencyResult{}
	}
	return bnLatency
}

// 获取所有延迟信息
func (n *P2PLatencyNode) GetBnLatencyAll() map[string]BnLatencyResult {
	bnLatencyMap := make(map[string]BnLatencyResult)
	n.NodeBnLatencyMap.Range(func(key string, value BnLatencyResult) bool {
		bnLatencyMap[key] = value
		return true
	})

	bnLatencyMap[n.Node.PeerName] = *n.BnLatency
	return bnLatencyMap
}
