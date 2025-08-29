#include "websocket_client_libcurl.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <curl/curl.h>

struct WebSocketClientLibcurl {
    CURL *curl_handle;
    int is_initialized;
};

static int global_ws_initialized = 0;

static int64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC_RAW, &ts);
    return (int64_t)ts.tv_sec * 1000000000LL + (int64_t)ts.tv_nsec;
}

static char* make_error(const char* msg) {
    if (!msg) return NULL;
    size_t len = strlen(msg);
    if (len == 0) return NULL;
    
    char* err = malloc(len + 1);
    if (!err) return NULL; // 内存分配失败
    
    strcpy(err, msg);
    return err;
}

int websocket_client_init_libcurl() {
    if (global_ws_initialized) return 0;
    if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK) return -1;
    global_ws_initialized = 1;
    return 0;
}

WebSocketClientLibcurl* websocket_client_new_libcurl() {
    if (!global_ws_initialized && websocket_client_init_libcurl() != 0) return NULL;

    WebSocketClientLibcurl* client = malloc(sizeof(WebSocketClientLibcurl));
    if (!client) return NULL;

    client->curl_handle = curl_easy_init();
    if (!client->curl_handle) {
        free(client);
        return NULL;
    }

    client->is_initialized = 1;
    return client;
}

// 建立 WebSocket 连接
WebSocketResultLibcurl websocket_connect_libcurl(WebSocketClientLibcurl* client, const char* url, int timeout_ms) {
    WebSocketResultLibcurl result = {0};
    result.latency_ns = -1;

    if (!client || !client->is_initialized || !client->curl_handle || !url) {
        result.error_message = make_error(!client ? "Invalid client" :
                                          !client->is_initialized ? "Client not initialized" :
                                          !client->curl_handle ? "CURL handle not available" : "Invalid URL");
        return result;
    }

    int64_t start_time = get_time_ns();

    curl_easy_reset(client->curl_handle);
    curl_easy_setopt(client->curl_handle, CURLOPT_URL, url);
    curl_easy_setopt(client->curl_handle, CURLOPT_CONNECT_ONLY, 2L); // 启用 WebSocket 模式
    curl_easy_setopt(client->curl_handle, CURLOPT_TIMEOUT_MS, (long)timeout_ms);
    
    // 设置WebSocket特定的选项
    curl_easy_setopt(client->curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_1_1);
    curl_easy_setopt(client->curl_handle, CURLOPT_FOLLOWLOCATION, 1L);

    CURLcode res = curl_easy_perform(client->curl_handle);
    if (res == CURLE_OK) {
        result.latency_ns = get_time_ns() - start_time;
        result.status_code = 101; // WebSocket 握手成功 (HTTP 101 Switching Protocols)
    } else {
        result.error_message = make_error(curl_easy_strerror(res));
    }

    return result;
}

// 发送 WebSocket 消息（修正版）
int websocket_send_libcurl(WebSocketClientLibcurl* client, const char* msg, size_t len, int is_text) {
    if (!client || !client->is_initialized || !client->curl_handle) 
        return WEBSOCKET_ERROR_INVALID_CLIENT;
    if (!msg || len == 0) 
        return WEBSOCKET_ERROR_INVALID_PARAMS;

    size_t sent = 0;
    unsigned int flags = is_text ? CURLWS_TEXT : CURLWS_BINARY;

    CURLcode rc = curl_ws_send(client->curl_handle,
                               msg,
                               len,
                               &sent,
                               0,     // fragsize: 非分片发送用 0
                               flags);

    switch (rc) {
        case CURLE_OK:
            return (int)sent;
        case CURLE_AGAIN:
            return 0; // 需要重试
        case CURLE_OPERATION_TIMEDOUT:
            return WEBSOCKET_ERROR_TIMEOUT;
        case CURLE_OUT_OF_MEMORY:
            return WEBSOCKET_ERROR_MEMORY;
        default:
            return WEBSOCKET_ERROR_SEND_FAILED;
    }
}

// 接收 WebSocket 消息（改进版本）
char* websocket_recv_libcurl(WebSocketClientLibcurl* client, size_t* out_len, int* out_is_text) {
    if (!client || !client->is_initialized || !client->curl_handle) return NULL;

    size_t buffer_size = WEBSOCKET_INITIAL_BUFFER_SIZE;
    char* buffer = malloc(buffer_size);
    if (!buffer) return NULL;

    size_t total_received = 0;
    const struct curl_ws_frame *frame = NULL;
    int retry_count = 0;
    const int max_retries = 10; // 最多重试10次
    
    // 循环接收直到获得完整消息或出错
    while (retry_count < max_retries) {
        size_t nread = 0;
        size_t available_space = buffer_size - total_received;
        
        // 如果剩余空间不足，扩展缓冲区
        if (available_space < 1024 && buffer_size < WEBSOCKET_MAX_BUFFER_SIZE) {
            size_t new_size = buffer_size * 2;
            if (new_size > WEBSOCKET_MAX_BUFFER_SIZE) {
                new_size = WEBSOCKET_MAX_BUFFER_SIZE;
            }
            
            char* new_buffer = realloc(buffer, new_size);
            if (!new_buffer) {
                free(buffer);
                return NULL; // 内存扩展失败
            }
            buffer = new_buffer;
            buffer_size = new_size;
            available_space = buffer_size - total_received;
        }
        
        // 检查是否达到最大缓冲区限制
        if (available_space < 1024) {
            free(buffer);
            return NULL; // 缓冲区溢出
        }

        CURLcode rc = curl_ws_recv(client->curl_handle, 
                                   buffer + total_received, 
                                   available_space - 1, // 保留一个字节用于null终止符
                                   &nread, &frame);
        
        // 处理不同的返回码
        if (rc == CURLE_AGAIN) {
            // 暂无数据，增加重试计数并继续
            retry_count++;
            // 如果已经有数据，检查是否是完整帧
            if (total_received > 0 && frame && !(frame->flags & CURLWS_CONT)) {
                break; // 已有完整帧，可以返回
            }
            // 短暂等待后重试（使用简单的延时模拟）
            struct timespec ts = {0, 10000000}; // 10ms
            nanosleep(&ts, NULL);
            continue;
        }
        
        if (rc != CURLE_OK) {
            // 其他错误，但如果已经有数据，尝试返回
            if (total_received > 0) {
                break;
            }
            free(buffer);
            return NULL; // 接收错误且无数据
        }
        
        // 重置重试计数（收到数据时）
        if (nread > 0) {
            retry_count = 0;
            total_received += nread;
            
            // 检查是否是完整的帧
            if (frame) {
                // 如果不是续传帧，表示这是完整的消息或消息的最后一部分
                if (!(frame->flags & CURLWS_CONT)) {
                    break;
                }
            } else {
                // 没有帧信息，假设是完整的
                break;
            }
        } else {
            // nread == 0，可能是连接关闭或暂无数据
            retry_count++;
        }
    }
    
    if (total_received == 0) {
        free(buffer);
        return NULL;
    }
    
    // 确保以null结尾
    buffer[total_received] = '\0';
    
    // 优化内存使用：缩小到实际大小
    if (total_received + 1 < buffer_size) {
        char* optimized_buffer = realloc(buffer, total_received + 1);
        if (optimized_buffer) {
            buffer = optimized_buffer;
        }
    }
    
    if (out_len) *out_len = total_received;
    if (out_is_text) *out_is_text = (frame && (frame->flags & CURLWS_TEXT)) != 0;
    
    return buffer;
}

void websocket_free_error_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

void websocket_free_message_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

void websocket_client_destroy_libcurl(WebSocketClientLibcurl* client) {
    if (client) {
        if (client->curl_handle) {
            curl_easy_cleanup(client->curl_handle);
            client->curl_handle = NULL;
        }
        client->is_initialized = 0;
        free(client);
    }
}

void websocket_client_cleanup_libcurl() {
    if (global_ws_initialized) {
        curl_global_cleanup();
        global_ws_initialized = 0;
    }
}
