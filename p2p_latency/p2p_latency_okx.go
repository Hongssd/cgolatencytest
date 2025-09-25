package p2p_latency

import (
	"time"

	"github.com/google/uuid"
)

// 刷新OKX延迟信息
func (n *P2PLatencyNode) refreshOkxLatency() error {
	result, err := TestOkxHttpAndWsLatency()
	if err != nil {
		return err
	}
	n.OkxLatency = result
	return nil
}

// 广播OKX延迟消息给所有远程P2P节点
func (n *P2PLatencyNode) broadcastOkxLatencyMsg() error {
	//广播延迟信息
	okxLatencyData, err := json.Marshal(n.OkxLatency)
	if err != nil {
		return err
	}

	//构建返回消息
	p2pMsgRes := P2PMessage{
		IsReq: false,
		Res: P2PRes{
			ReqId:        uuid.New().String(),
			ReqType:      P2PReqTypeOkxLatency,
			ResData:      string(okxLatencyData),
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
	//广播OKX延迟消息给所有远程P2P节点
	err = n.Node.BroadcastMsg(string(p2pMsgResBytes), true)
	if err != nil {
		log.Errorf("SendResMsg error: %v", err)
		return err
	}
	return nil
}

// 处理OKX延迟请求
func (n *P2PLatencyNode) handleOkxLatencyMsgReq(p2pMsg P2PMessage, fromPeerName string, inTimestamp int64) error {

	okxLatencyData, err := json.Marshal(n.OkxLatency)
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
			ResData:      string(okxLatencyData),
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

func (n *P2PLatencyNode) handleOkxLatencyMsgRes(p2pMsg P2PMessage, fromPeerName string) error {
	targetOkxLatency := OkxLatencyResult{}
	err := json.Unmarshal([]byte(p2pMsg.Res.ResData), &targetOkxLatency)
	if err != nil {
		return err
	}
	n.NodeOkxLatencyMap.Store(fromPeerName, targetOkxLatency)
	return nil
}

// 通过节点名获取OKX延迟信息
func (n *P2PLatencyNode) GetOkxLatencyFromNodeName(nodeName string) OkxLatencyResult {
	if nodeName == n.Node.PeerName {
		return *n.OkxLatency
	}
	okxLatency, ok := n.NodeOkxLatencyMap.Load(nodeName)
	if !ok {
		return OkxLatencyResult{}
	}
	return okxLatency
}

// 获取所有OKX延迟信息
func (n *P2PLatencyNode) GetOkxLatencyAll() map[string]OkxLatencyResult {
	okxLatencyMap := make(map[string]OkxLatencyResult)
	n.NodeOkxLatencyMap.Range(func(key string, value OkxLatencyResult) bool {
		okxLatencyMap[key] = value
		return true
	})

	if n.OkxLatency != nil {
		okxLatencyMap[n.Node.PeerName] = *n.OkxLatency
	}

	return okxLatencyMap
}
