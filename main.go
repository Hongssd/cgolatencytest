package main

import (
	"net"
	"strconv"
	"time"

	"github.com/Hongssd/cgolatencytest/config"
	"github.com/Hongssd/cgolatencytest/mylog"
	"github.com/Hongssd/cgolatencytest/p2p_latency"
)

var log = mylog.Log

func main() {
	// 启动HTTP服务器

	myIP, err := p2p_latency.GetMyIP()
	if err != nil {
		log.Errorf("获取本机IP失败: %v", err)
		return
	}
	allNodeList := config.GetConfigSlice("p2p_nodes")

	p2pPort := 0
	otherNodeList := make([]string, 0)
	for _, node := range allNodeList {
		host, port, err := net.SplitHostPort(node)
		if err != nil {
			log.Errorf("获取节点地址失败: %v", err)
			return
		}
		if host == myIP {
			p2pPort, _ = strconv.Atoi(port)
		} else {
			otherNodeList = append(otherNodeList, node)
		}
	}

	p2pNode, err := p2p_latency.NewP2PLatencyNode(myIP, p2pPort, otherNodeList)
	if err != nil {
		log.Errorf("创建P2P节点失败: %v", err)
		return
	}

	http_port := config.GetConfigInt("http_port")

	go p2pNode.StartHTTPServer(http_port)

	log.Info("开始监控币安延迟信息...")
	// 原有的延迟监控逻辑
	for {
		time.Sleep(time.Second * 30)

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
