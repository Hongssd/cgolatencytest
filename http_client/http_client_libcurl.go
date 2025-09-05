package http_client

/*
#cgo CFLAGS: -I${SRCDIR}/lib/include -O3 -march=native -mtune=native -Wall -Wextra
#cgo LDFLAGS: ${SRCDIR}/lib/libcurl.a -lssl -lcrypto -lz -lpthread -lnghttp2 -lpsl -lidn2
#include "http_client_libcurl.h"
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

// HTTP方法常量
const (
	HTTP_METHOD_HEAD   = 0
	HTTP_METHOD_GET    = 1
	HTTP_METHOD_POST   = 2
	HTTP_METHOD_PUT    = 3
	HTTP_METHOD_DELETE = 4
	HTTP_METHOD_PATCH  = 5
)

// ResultLibcurl 结构体
type ResultLibcurl struct {
	LatencyNs      int64
	RequestTimeNs  int64
	ResponseTimeNs int64
	StatusCode     int
	Error          string
	DNSTimeNs      int64
	ConnectTimeNs  int64
	TLSTimeNs      int64
	ResponseBody   string
	ResponseSize   int
}

// ClientLibcurl HTTP客户端实例
type ClientLibcurl struct {
	client unsafe.Pointer
}

// NewClientLibcurl 创建新的HTTP客户端实例
func NewClientLibcurl() (*ClientLibcurl, error) {
	client := C.http_client_new_libcurl()
	if client == nil {
		return nil, &CError{Code: -1}
	}
	return &ClientLibcurl{client: unsafe.Pointer(client)}, nil
}

// Close 关闭HTTP客户端实例
func (c *ClientLibcurl) Close() {
	if c.client != nil {
		C.http_client_destroy_libcurl((*C.HttpClientLibcurl)(c.client))
		c.client = nil
	}
}

// Request 执行HTTP请求
func (c *ClientLibcurl) Request(url string, timeoutMs int, forceHttpVersion int, method int, postData string, headers []string) ResultLibcurl {
	if c.client == nil {
		return ResultLibcurl{Error: "Client not initialized"}
	}

	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	var cPostData *C.char
	if postData != "" {
		cPostData = C.CString(postData)
		defer C.free(unsafe.Pointer(cPostData))
	}

	var cHeaders **C.char
	if len(headers) > 0 {
		cHeadersArray := make([]*C.char, len(headers)+1)
		for i, header := range headers {
			cHeadersArray[i] = C.CString(header)
			defer C.free(unsafe.Pointer(cHeadersArray[i]))
		}
		cHeadersArray[len(headers)] = nil
		cHeaders = &cHeadersArray[0]
	}

	res := C.http_request_libcurl((*C.HttpClientLibcurl)(c.client), cURL, C.int(timeoutMs), C.int(forceHttpVersion),
		C.HttpMethod(method), cPostData, cHeaders)

	var goErr string
	if res.error_message != nil {
		goErr = C.GoString(res.error_message)
		C.http_free_error_libcurl(res.error_message)
	}

	var responseBody string
	if res.response_body != nil {
		responseBody = C.GoString(res.response_body)
		C.http_free_response_libcurl(res.response_body)
	}

	return ResultLibcurl{
		LatencyNs:      int64(res.latency_ns),
		RequestTimeNs:  int64(res.request_time_ns),
		ResponseTimeNs: int64(res.response_time_ns),
		StatusCode:     int(res.status_code),
		Error:          goErr,
		DNSTimeNs:      int64(res.dns_time_ns),
		ConnectTimeNs:  int64(res.connect_time_ns),
		TLSTimeNs:      int64(res.tls_time_ns),
		ResponseBody:   responseBody,
		ResponseSize:   int(res.response_size),
	}
}

// HTTP方法便捷函数
func (c *ClientLibcurl) Head(url string, timeoutMs int, forceHttpVersion int) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_HEAD, "", nil)
}

func (c *ClientLibcurl) Get(url string, timeoutMs int, forceHttpVersion int) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_GET, "", nil)
}

func (c *ClientLibcurl) Post(url string, timeoutMs int, forceHttpVersion int, postData string, headers []string) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_POST, postData, headers)
}

func (c *ClientLibcurl) Put(url string, timeoutMs int, forceHttpVersion int, postData string, headers []string) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_PUT, postData, headers)
}

func (c *ClientLibcurl) Delete(url string, timeoutMs int, forceHttpVersion int) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_DELETE, "", nil)
}

func (c *ClientLibcurl) Patch(url string, timeoutMs int, forceHttpVersion int, postData string, headers []string) ResultLibcurl {
	return c.Request(url, timeoutMs, forceHttpVersion, HTTP_METHOD_PATCH, postData, headers)
}

// 全局环境管理
func InitLibcurl() error {
	r := C.http_client_init_libcurl()
	if r != 0 {
		return &CError{Code: int(r)}
	}
	return nil
}

func CleanupLibcurl() {
	C.http_client_cleanup_libcurl()
}

// CError 错误类型
type CError struct {
	Code int
}

func (e *CError) Error() string {
	return "libcurl operation failed"
}
