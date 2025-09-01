#ifndef HTTP_CLIENT_LIBCURL_H
#define HTTP_CLIENT_LIBCURL_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// HTTP请求方法枚举
typedef enum {
    HTTP_METHOD_HEAD = 0,
    HTTP_METHOD_GET = 1,
    HTTP_METHOD_POST = 2,
    HTTP_METHOD_PUT = 3,
    HTTP_METHOD_DELETE = 4,
    HTTP_METHOD_PATCH = 5
} HttpMethod;

// HTTP客户端句柄结构
typedef struct HttpClientLibcurl HttpClientLibcurl;

// HTTP请求结果结构
typedef struct {
    int64_t latency_ns;
    int64_t request_time_ns;  // 发起请求时刻的纳秒时间戳
    int status_code;
    char* error_message;
    int64_t dns_time_ns;
    int64_t connect_time_ns;
    int64_t tls_time_ns;
    char* response_body;
    size_t response_size;
} HttpResultLibcurl;

// 核心接口函数
HttpClientLibcurl* http_client_new_libcurl();
int http_client_init_libcurl();
HttpResultLibcurl http_request_libcurl(HttpClientLibcurl* client, const char* url, int timeout_ms, 
                                      int force_http_version, HttpMethod method,
                                      const char* post_data, const char** headers);
void http_free_error_libcurl(char* ptr);
void http_free_response_libcurl(char* ptr);
void http_client_destroy_libcurl(HttpClientLibcurl* client);
void http_client_cleanup_libcurl();

#ifdef __cplusplus
}
#endif

#endif
