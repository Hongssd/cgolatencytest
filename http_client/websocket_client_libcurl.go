package http_client

/*
#cgo LDFLAGS: -lcurl
#include "websocket_client_libcurl.h"
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

// WebSocketResultLibcurl 结构体（与C保持一致）
type WebSocketResultLibcurl struct {
	LatencyNs  int64
	StatusCode int
	Error      string
}

// WebSocketClientLibcurl Go封装的客户端
type WebSocketClientLibcurl struct {
	client unsafe.Pointer
}

// InitWebSocketLibcurl 初始化全局环境
func InitWebSocketLibcurl() error {
	r := C.websocket_client_init_libcurl()
	if r != 0 {
		return &CError{Code: int(r)}
	}
	return nil
}

// CleanupWebSocketLibcurl 清理全局环境
func CleanupWebSocketLibcurl() {
	C.websocket_client_cleanup_libcurl()
}

// NewWebSocketClientLibcurl 创建新的WebSocket客户端实例
func NewWebSocketClientLibcurl() (*WebSocketClientLibcurl, error) {
	client := C.websocket_client_new_libcurl()
	if client == nil {
		return nil, &CError{Code: -1}
	}
	return &WebSocketClientLibcurl{client: unsafe.Pointer(client)}, nil
}

// Close 关闭并释放WebSocket客户端
func (c *WebSocketClientLibcurl) Close() {
	if c.client != nil {
		C.websocket_client_destroy_libcurl((*C.WebSocketClientLibcurl)(c.client))
		c.client = nil
	}
}

// Connect 建立WebSocket连接
func (c *WebSocketClientLibcurl) Connect(url string, timeoutMs int) WebSocketResultLibcurl {
	if c.client == nil {
		return WebSocketResultLibcurl{Error: "Client not initialized"}
	}

	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	res := C.websocket_connect_libcurl((*C.WebSocketClientLibcurl)(c.client), cURL, C.int(timeoutMs))

	var goErr string
	if res.error_message != nil {
		goErr = C.GoString(res.error_message)
		C.websocket_free_error_libcurl(res.error_message)
	}

	return WebSocketResultLibcurl{
		LatencyNs:  int64(res.latency_ns),
		StatusCode: int(res.status_code),
		Error:      goErr,
	}
}

// Send 发送WebSocket消息
// isText = true 发送文本，false 发送二进制
func (c *WebSocketClientLibcurl) Send(msg string, isText bool) (int, error) {
	if c.client == nil {
		return -1, &CError{Code: -1}
	}

	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))

	sent := C.websocket_send_libcurl((*C.WebSocketClientLibcurl)(c.client),
		cMsg, C.size_t(len(msg)), C.int(boolToInt(isText)))

	if sent < 0 {
		return int(sent), &CError{Code: int(sent)}
	}
	return int(sent), nil
}

// Recv 接收WebSocket消息
// 返回消息字符串、是否文本、错误
func (c *WebSocketClientLibcurl) Recv() (string, bool, error) {
	if c.client == nil {
		return "", false, &CError{Code: -1}
	}

	var outLen C.size_t
	var outIsText C.int

	msg := C.websocket_recv_libcurl((*C.WebSocketClientLibcurl)(c.client), &outLen, &outIsText)
	if msg == nil {
		return "", false, nil
	}
	defer C.websocket_free_message_libcurl(msg)

	goMsg := C.GoStringN(msg, C.int(outLen))
	return goMsg, outIsText != 0, nil
}

// 工具函数：bool 转 int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
