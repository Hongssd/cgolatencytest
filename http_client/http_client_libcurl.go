package http_client

/*
#cgo LDFLAGS: -lcurl
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

// ResultLibcurl 结构体，对应C的HttpResultLibcurl
type ResultLibcurl struct {
	LatencyNs     int64
	StatusCode    int
	Error         string
	DNSTimeUs     int64
	ConnectTimeUs int64
	TLSTimeUs     int64
	ResponseBody  string
	ResponseSize  int
}

// HTTPRequestLibcurl 使用libcurl执行HTTP请求
func HTTPRequestLibcurl(url string, timeoutMs int, forceHttpVersion int, method int, postData string, headers []string) ResultLibcurl {
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

	res := C.http_request_libcurl(cURL, C.int(timeoutMs), C.int(forceHttpVersion),
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
		LatencyNs:     int64(res.latency_ns),
		StatusCode:    int(res.status_code),
		Error:         goErr,
		DNSTimeUs:     int64(res.dns_time_ns),
		ConnectTimeUs: int64(res.connect_time_ns),
		TLSTimeUs:     int64(res.tls_time_ns),
		ResponseBody:  responseBody,
		ResponseSize:  int(res.response_size),
	}
}

// HTTPHeadRequestLibcurl 执行HEAD请求（向后兼容）
func HTTPHeadRequestLibcurl(url string, timeoutMs int, forceHttpVersion int) ResultLibcurl {
	return HTTPRequestLibcurl(url, timeoutMs, forceHttpVersion, HTTP_METHOD_HEAD, "", nil)
}

// HTTPGetRequestLibcurl 执行GET请求
func HTTPGetRequestLibcurl(url string, timeoutMs int, forceHttpVersion int) ResultLibcurl {
	return HTTPRequestLibcurl(url, timeoutMs, forceHttpVersion, HTTP_METHOD_GET, "", nil)
}

// HTTPPostRequestLibcurl 执行POST请求
func HTTPPostRequestLibcurl(url string, timeoutMs int, forceHttpVersion int, postData string, headers []string) ResultLibcurl {
	return HTTPRequestLibcurl(url, timeoutMs, forceHttpVersion, HTTP_METHOD_POST, postData, headers)
}

// InitLibcurl 初始化libcurl客户端
func InitLibcurl() error {
	r := C.http_client_init_libcurl()
	if r != 0 {
		return &CError{Code: int(r)}
	}
	return nil
}

// CleanupLibcurl 清理libcurl资源
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
