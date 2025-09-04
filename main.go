package main

import (
	"cgolatencytest/mylog"
	"cgolatencytest/p2p_latency"
	"time"
)

var log = mylog.Log

func main() {

	for {
		time.Sleep(time.Second * 30)
		p2pNode := p2p_latency.GetP2PLatencyNode()

		bnLatencyAll := p2pNode.GetBnLatencyAll()

		for k, v := range bnLatencyAll {
			log.Infof("节点[%s]币安延迟信息: %+v", k, v)
		}

		nodeLatencyAll := p2pNode.GetAllAvgLatency()
		for k, v := range nodeLatencyAll {
			log.Infof("节点[%s]平均延迟信息: %+v", k, v)
		}

	}

}
