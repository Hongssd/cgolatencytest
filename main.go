package main

import (
	"cgolatencytest/config"
	"cgolatencytest/mylog"
	"cgolatencytest/p2p_latency"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var log = mylog.Log

var http_port = config.GetConfigInt("http_port")

// API响应结构
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 币安延迟响应结构
type BnLatencyResponse struct {
	NodeName string                      `json:"node_name"`
	Latency  p2p_latency.BnLatencyResult `json:"latency"`
}

// 节点延迟响应结构
type NodeLatencyResponse struct {
	NodeName string `json:"node_name"`
	Latency  int64  `json:"latency_us"`
}

// 币安延迟API处理器
func handleBnLatency(w http.ResponseWriter, r *http.Request) {
	log.Infof("收到币安延迟查询请求")

	p2pNode := p2p_latency.GetP2PLatencyNode()
	if p2pNode == nil {
		response := ApiResponse{
			Code:    500,
			Message: "P2P节点未初始化",
			Data:    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	bnLatencyAll := p2pNode.GetBnLatencyAll()
	var responses []BnLatencyResponse

	for nodeName, latency := range bnLatencyAll {
		responses = append(responses, BnLatencyResponse{
			NodeName: nodeName,
			Latency:  latency,
		})
		log.Infof("节点[%s]币安延迟信息: %+v", nodeName, latency)
	}

	response := ApiResponse{
		Code:    200,
		Message: "查询成功",
		Data:    responses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 节点延迟API处理器
func handleNodeLatency(w http.ResponseWriter, r *http.Request) {
	log.Infof("收到节点延迟查询请求")

	p2pNode := p2p_latency.GetP2PLatencyNode()
	if p2pNode == nil {
		response := ApiResponse{
			Code:    500,
			Message: "P2P节点未初始化",
			Data:    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	nodeLatencyAll := p2pNode.GetAllAvgLatency()
	var responses []NodeLatencyResponse

	for nodeName, latency := range nodeLatencyAll {
		responses = append(responses, NodeLatencyResponse{
			NodeName: nodeName,
			Latency:  latency,
		})
		log.Infof("节点[%s]平均延迟信息: %+v", nodeName, latency)
	}

	response := ApiResponse{
		Code:    200,
		Message: "查询成功",
		Data:    responses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// 启动HTTP服务器
	go startHTTPServer()

	// 原有的延迟监控逻辑
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

// 启动HTTP服务器
func startHTTPServer() {
	// 注册API路由
	http.HandleFunc("/api/bn-latency", handleBnLatency)
	http.HandleFunc("/api/node-latency", handleNodeLatency)

	// 启动服务器
	serverAddr := fmt.Sprintf(":%d", http_port)
	log.Infof("HTTP服务器启动，监听端口: %d", http_port)
	log.Infof("API端点:")
	log.Infof("  GET /api/bn-latency - 查询币安延迟")
	log.Infof("  GET /api/node-latency - 查询节点延迟")

	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Errorf("HTTP服务器启动失败: %v", err)
	}
	log.Infof("HTTP服务器启动成功，监听端口: %d", http_port)
}
