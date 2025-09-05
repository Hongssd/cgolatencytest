package p2p_latency

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// 广播节点平均延迟
func (n *P2PLatencyNode) broadcastAvgLatencyMsg() error {
	var timestampStr = strconv.FormatInt(time.Now().UnixNano(), 10)

	p2pReq := P2PReq{
		ReqId:   uuid.New().String(),
		ReqType: P2PReqTypeLatency,
		ReqData: timestampStr,
	}

	p2pReqMsg := P2PMessage{
		IsReq: true,
		Req:   p2pReq,
	}

	p2pReqMsgBytes, err := json.Marshal(p2pReqMsg)
	if err != nil {
		log.Errorf("序列化请求失败: %v", err)
		return err
	}
	//广播
	err = n.Node.BroadcastMsg(string(p2pReqMsgBytes), true)
	if err != nil {
		return fmt.Errorf("广播网络平均延迟消息失败: %v", err)
	}
	return nil
}

// 处理节点平均延迟
func (n *P2PLatencyNode) handleAvgLatencyMsg(p2pMsg P2PMessage, fromPeerName string) error {
	var timestampStr = p2pMsg.Req.ReqData
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		log.Errorf("ParseInt error: %v", err)
		return err
	}

	//计算延迟 使用当前毫秒时间戳减请求时间戳
	latency := time.Now().UnixNano() - timestamp

	newAvgLatency := latency
	//更新目标节点延迟
	nowAvgLatency, ok := n.NodeAvgLatencyMap.Load(fromPeerName)
	if ok {
		newAvgLatency = (nowAvgLatency + latency) / 2
	}

	n.NodeAvgLatencyMap.Store(fromPeerName, newAvgLatency)
	log.Infof("更新目标节点[%s]网络平均延迟: %.6fms -> %.6fms", fromPeerName,
		float64(nowAvgLatency)/1000000, float64(newAvgLatency)/1000000)
	return nil
}

// 获取目标节点平均延迟
func (n *P2PLatencyNode) GetAvgLatencyFromNodeName(nodeName string) int64 {
	avgLatency, ok := n.NodeAvgLatencyMap.Load(nodeName)
	if !ok {
		return 0
	}
	return avgLatency
}

// 获取所有节点平均延迟
func (n *P2PLatencyNode) GetAllAvgLatency() map[string]int64 {
	avgLatencyMap := make(map[string]int64)
	n.NodeAvgLatencyMap.Range(func(key string, value int64) bool {
		avgLatencyMap[key] = value
		return true
	})
	return avgLatencyMap
}
