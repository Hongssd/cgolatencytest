#include "http_client_libcurl.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <curl/curl.h>

// 全局CURL句柄
static CURL* curl_handle = NULL;

// 响应体数据结构
typedef struct {
    char* data;
    size_t size;
} ResponseData;

// 获取当前时间（纳秒）
static int64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC_RAW, &ts);
    return (int64_t)ts.tv_sec * 1000000000LL + (int64_t)ts.tv_nsec;
}

// 错误信息创建函数
static char* make_error(const char* msg) {
    if (!msg) return NULL;
    size_t len = strlen(msg);
    char* err = malloc(len + 1);
    if (err) strcpy(err, msg);
    return err;
}

// 智能选择HTTP版本的辅助函数
static int should_use_http2(const char* url) {
    if (!url) return 0;
    
    // 检查是否是WebSocket Secure (WSS) - 使用HTTP/2
    if (strncmp(url, "wss://", 6) == 0) {
        return 1;
    }
    
    // 检查是否是HTTPS - 使用HTTP/2
    if (strncmp(url, "https://", 8) == 0) {
        return 1;
    }
    
    // HTTP和WS使用HTTP/1.1
    return 0;
}

// 响应体写入回调函数
static size_t write_callback(void* contents, size_t size, size_t nmemb, void* userp) {
    ResponseData* resp = (ResponseData*)userp;
    size_t realsize = size * nmemb;
    
    char* ptr = realloc(resp->data, resp->size + realsize + 1);
    if (!ptr) return 0;
    
    resp->data = ptr;
    memcpy(&(resp->data[resp->size]), contents, realsize);
    resp->size += realsize;
    resp->data[resp->size] = 0;
    
    return realsize;
}

// 初始化HTTP客户端
int http_client_init_libcurl() {
    if (curl_handle) return 0;
    
    if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK) {
        return -1;
    }
    
    curl_handle = curl_easy_init();
    if (!curl_handle) {
        curl_global_cleanup();
        return -2;
    }
    
    return 0;
}

// 执行HTTP请求
HttpResultLibcurl http_request_libcurl(const char* url, int timeout_ms, 
                                      int force_http_version, HttpMethod method,
                                      const char* post_data, const char** headers) {
    HttpResultLibcurl result = {0};
    result.latency_ns = -1;
    result.status_code = 0;
    result.error_message = NULL;
    result.dns_time_ns = -1;
    result.connect_time_ns = -1;
    result.tls_time_ns = -1;
    result.response_body = NULL;
    result.response_size = 0;
    
    if (!curl_handle || !url) {
        result.error_message = make_error(!curl_handle ? "HTTP client not initialized" : "Invalid URL");
        return result;
    }
    
    // 记录开始时间
    int64_t start_time = get_time_ns();
    
    // 重置并配置CURL
    curl_easy_reset(curl_handle);
    curl_easy_setopt(curl_handle, CURLOPT_URL, url);
    curl_easy_setopt(curl_handle, CURLOPT_TIMEOUT_MS, (long)timeout_ms);
    curl_easy_setopt(curl_handle, CURLOPT_CONNECTTIMEOUT_MS, (long)(timeout_ms / 2));
    curl_easy_setopt(curl_handle, CURLOPT_USERAGENT, "HTTPLatencyTest/1.0");
    
    // 根据HTTP方法配置请求
    switch (method) {
        case HTTP_METHOD_HEAD:
            curl_easy_setopt(curl_handle, CURLOPT_HEADER, 1L);
            curl_easy_setopt(curl_handle, CURLOPT_NOBODY, 1L);
            break;
            
        case HTTP_METHOD_GET:
            curl_easy_setopt(curl_handle, CURLOPT_HTTPGET, 1L);
            break;
            
        case HTTP_METHOD_POST:
            curl_easy_setopt(curl_handle, CURLOPT_POST, 1L);
            if (post_data) {
                curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDS, post_data);
            }
            break;
            
        case HTTP_METHOD_PUT:
            curl_easy_setopt(curl_handle, CURLOPT_CUSTOMREQUEST, "PUT");
            if (post_data) {
                curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDS, post_data);
            }
            break;
            
        case HTTP_METHOD_DELETE:
            curl_easy_setopt(curl_handle, CURLOPT_CUSTOMREQUEST, "DELETE");
            break;
            
        case HTTP_METHOD_PATCH:
            curl_easy_setopt(curl_handle, CURLOPT_CUSTOMREQUEST, "PATCH");
            if (post_data) {
                curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDS, post_data);
            }
            break;
    }
    
    // 设置自定义请求头
    if (headers) {
        struct curl_slist* header_list = NULL;
        for (int i = 0; headers[i] != NULL; i++) {
            header_list = curl_slist_append(header_list, headers[i]);
        }
        if (header_list) {
            curl_easy_setopt(curl_handle, CURLOPT_HTTPHEADER, header_list);
        }
    }
    

    ResponseData resp = {0};
    curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION, write_callback);
    curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA, &resp);
    
    // 执行请求
    CURLcode res = curl_easy_perform(curl_handle);
    
    if (res == CURLE_OK) {
        // 获取状态码和时间信息
        long response_code;
        curl_easy_getinfo(curl_handle, CURLINFO_RESPONSE_CODE, &response_code);
        result.status_code = (int)response_code;
        result.latency_ns = get_time_ns() - start_time;
        
        // 设置响应体
        if (resp.data && resp.size > 0) {
            result.response_body = resp.data;
            result.response_size = resp.size;
        } else {
            free(resp.data); // 释放空数据
        }
        
        // 获取详细时间信息
        curl_off_t dns_time, connect_time, app_connect_time;
        curl_easy_getinfo(curl_handle, CURLINFO_NAMELOOKUP_TIME_T, &dns_time);
        curl_easy_getinfo(curl_handle, CURLINFO_CONNECT_TIME_T, &connect_time);
        curl_easy_getinfo(curl_handle, CURLINFO_APPCONNECT_TIME_T, &app_connect_time);
        
        result.dns_time_ns = (int64_t)(dns_time);        // 微秒
        result.connect_time_ns = (int64_t)(connect_time); // 微秒
        result.tls_time_ns = (int64_t)(app_connect_time); // 微秒
    } else {
        result.error_message = make_error(curl_easy_strerror(res));
        free(resp.data); // 释放可能的部分数据
    }

    
    // 智能HTTP版本选择
    if (force_http_version == 0) {
        if (should_use_http2(url)) {
            curl_easy_setopt(curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_2_0);
        } else {
            curl_easy_setopt(curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_1_1);
        }
    } else if (force_http_version == 1) {
        curl_easy_setopt(curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_1_1);
    } else if (force_http_version == 2) {
        curl_easy_setopt(curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_2_0);
    }
    
    return result;
}


// 释放错误信息内存
void http_free_error_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

// 释放响应体内存
void http_free_response_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

// 清理HTTP客户端资源
void http_client_cleanup_libcurl() {
    if (curl_handle) {
        curl_easy_cleanup(curl_handle);
        curl_handle = NULL;
    }
    curl_global_cleanup();
}
