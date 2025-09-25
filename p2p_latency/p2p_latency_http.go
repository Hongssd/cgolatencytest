package p2p_latency

import (
	"fmt"
	"net/http"
)

// API响应结构
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 币安延迟响应结构
type BnLatencyResponse struct {
	NodeName string          `json:"node_name"`
	Latency  BnLatencyResult `json:"latency"`
}

// OKX延迟响应结构
type OkxLatencyResponse struct {
	NodeName string           `json:"node_name"`
	Latency  OkxLatencyResult `json:"latency"`
}

// 节点延迟响应结构
type NodeLatencyResponse struct {
	NodeName string `json:"node_name"`
	Latency  int64  `json:"latency_us"`
}

// 币安延迟API处理器
func (n *P2PLatencyNode) handleBnLatency(w http.ResponseWriter, r *http.Request) {
	log.Infof("收到币安延迟查询请求")

	if n == nil {
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

	bnLatencyAll := n.GetBnLatencyAll()
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

// OKX延迟API处理器
func (n *P2PLatencyNode) handleOkxLatency(w http.ResponseWriter, r *http.Request) {
	log.Infof("收到OKX延迟查询请求")

	if n == nil {
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

	okxLatencyAll := n.GetOkxLatencyAll()
	var responses []OkxLatencyResponse

	for nodeName, latency := range okxLatencyAll {
		responses = append(responses, OkxLatencyResponse{
			NodeName: nodeName,
			Latency:  latency,
		})
		log.Infof("节点[%s]OKX延迟信息: %+v", nodeName, latency)
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
func (n *P2PLatencyNode) handleNodeLatency(w http.ResponseWriter, r *http.Request) {
	log.Infof("收到节点延迟查询请求")

	if n == nil {
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

	nodeLatencyAll := n.GetAllAvgLatency()
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

// 启动HTTP服务器
func (n *P2PLatencyNode) StartHTTPServer(http_port int) {
	// 注册API路由
	http.HandleFunc("/api/bn-latency", n.handleBnLatency)
	http.HandleFunc("/api/okx-latency", n.handleOkxLatency)
	http.HandleFunc("/api/node-latency", n.handleNodeLatency)

	// 启动服务器
	if http_port == 0 {
		http_port = 8080
	}
	serverAddr := fmt.Sprintf(":%d", http_port)
	log.Infof("HTTP服务器启动，监听端口: %d", http_port)
	log.Infof("API端点:")
	log.Infof("  GET /api/bn-latency - 查询币安延迟")
	log.Infof("  GET /api/okx-latency - 查询OKX延迟")
	log.Infof("  GET /api/node-latency - 查询节点延迟")

	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Errorf("HTTP服务器启动失败: %v", err)
	}
	log.Infof("HTTP服务器启动成功，监听端口: %d", http_port)
}
