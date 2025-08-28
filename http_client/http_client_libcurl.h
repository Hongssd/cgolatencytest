#ifndef HTTP_CLIENT_LIBCURL_H
#define HTTP_CLIENT_LIBCURL_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// HTTP请求方法枚举
typedef enum {
    HTTP_METHOD_HEAD = 0,    // HEAD请求
    HTTP_METHOD_GET = 1,     // GET请求
    HTTP_METHOD_POST = 2,    // POST请求
    HTTP_METHOD_PUT = 3,     // PUT请求
    HTTP_METHOD_DELETE = 4,  // DELETE请求
    HTTP_METHOD_PATCH = 5    // PATCH请求
} HttpMethod;

// HTTP请求结果结构
typedef struct {
    int64_t latency_ns;     // 延迟时间（微秒），失败时为-1
    int status_code;         // HTTP状态码，失败时为0
    char* error_message;     // 错误信息字符串
    int64_t dns_time_ns;     // DNS解析时间（微秒）
    int64_t connect_time_ns; // TCP连接时间（微秒）
    int64_t tls_time_ns;     // TLS握手时间（微秒）
    char* response_body;     // 响应体内容（仅GET/POST等有body的方法）
    size_t response_size;    // 响应体大小
} HttpResultLibcurl;

// 初始化HTTP客户端
// 返回0表示成功，负数表示失败
int http_client_init_libcurl();

// 执行HTTP请求并测量延迟
// url: 完整的URL
// timeout_ms: 超时时间（毫秒）
// force_http_version: 强制HTTP版本 (0=自动, 1=HTTP/1.1, 2=HTTP/2)
// method: HTTP请求方法
// post_data: POST数据（仅POST/PUT/PATCH方法需要）
// headers: 自定义请求头（可选，NULL表示使用默认）
// 返回HttpResultLibcurl结构
HttpResultLibcurl http_request_libcurl(const char* url, int timeout_ms, 
                                      int force_http_version, HttpMethod method,
                                      const char* post_data, const char** headers);

// 释放错误信息和响应体内存
void http_free_error_libcurl(char* ptr);
void http_free_response_libcurl(char* ptr);

// 清理HTTP客户端资源
void http_client_cleanup_libcurl();

#ifdef __cplusplus
}
#endif

#endif // HTTP_CLIENT_LIBCURL_H
