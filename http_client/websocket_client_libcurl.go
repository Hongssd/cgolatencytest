package http_client

/*
#cgo CFLAGS: -I${SRCDIR}/lib/include -O3 -march=native -mtune=native -Wall -Wextra
#cgo LDFLAGS: ${SRCDIR}/lib/libcurl.a -lssl -lcrypto -lz -lpthread -lnghttp2 -lpsl -lidn2
#include "websocket_client_libcurl.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// WebSocket 错误码常量（与C代码保持一致）
const (
	WEBSOCKET_OK                    = 0
	WEBSOCKET_ERROR_INVALID_CLIENT  = -1
	WEBSOCKET_ERROR_INVALID_PARAMS  = -2
	WEBSOCKET_ERROR_SEND_FAILED     = -3
	WEBSOCKET_ERROR_NETWORK         = -4
	WEBSOCKET_ERROR_TIMEOUT         = -5
	WEBSOCKET_ERROR_MEMORY          = -6
	WEBSOCKET_ERROR_BUFFER_OVERFLOW = -7
)

// WebSocketResultLibcurl 结构体（与C保持一致）
type WebSocketResultLibcurl struct {
	LatencyNs  int64
	StatusCode int
	Error      string
}

// WebSocketError 封装WebSocket特定错误
type WebSocketError struct {
	Code    int
	Message string
}

func (e *WebSocketError) Error() string {
	switch e.Code {
	case WEBSOCKET_ERROR_INVALID_CLIENT:
		return "WebSocket client is invalid or not initialized"
	case WEBSOCKET_ERROR_INVALID_PARAMS:
		return "Invalid parameters provided"
	case WEBSOCKET_ERROR_SEND_FAILED:
		return "Failed to send WebSocket message"
	case WEBSOCKET_ERROR_NETWORK:
		return "Network error occurred"
	case WEBSOCKET_ERROR_TIMEOUT:
		return "Operation timed out"
	case WEBSOCKET_ERROR_MEMORY:
		return "Memory allocation failed"
	case WEBSOCKET_ERROR_BUFFER_OVERFLOW:
		return "Buffer overflow detected"
	default:
		return fmt.Sprintf("Unknown WebSocket error (code: %d)", e.Code)
	}
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
		return -1, &WebSocketError{Code: WEBSOCKET_ERROR_INVALID_CLIENT}
	}

	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))

	sent := C.websocket_send_libcurl((*C.WebSocketClientLibcurl)(c.client),
		cMsg, C.size_t(len(msg)), C.int(boolToInt(isText)))

	if sent < 0 {
		return int(sent), &WebSocketError{Code: int(sent)}
	}
	return int(sent), nil
}

// Recv 接收WebSocket消息
// 返回消息字符串、是否文本、错误
func (c *WebSocketClientLibcurl) Recv() (string, bool, error) {
	if c.client == nil {
		return "", false, &WebSocketError{Code: WEBSOCKET_ERROR_INVALID_CLIENT}
	}

	var outLen C.size_t
	var outIsText C.int

	msg := C.websocket_recv_libcurl((*C.WebSocketClientLibcurl)(c.client), &outLen, &outIsText)
	if msg == nil {
		return "", false, nil // 暂无数据，不是错误
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
