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
    char* err = malloc(len + 1);
    if (err) strcpy(err, msg);
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
    if (!client || !client->is_initialized || !client->curl_handle) return -1;
    if (!msg || len == 0) return -2;

    size_t sent = 0;
    unsigned int flags = is_text ? CURLWS_TEXT : CURLWS_BINARY;

    CURLcode rc = curl_ws_send(client->curl_handle,
                               msg,
                               len,
                               &sent,
                               0,     // fragsize: 非分片发送用 0
                               flags);

    if (rc == CURLE_OK) {
        return (int)sent;
    } else if (rc == CURLE_AGAIN) {
        return 0; // 需要重试
    }
    return -3; // 出错
}

// 接收 WebSocket 消息（修正版）
char* websocket_recv_libcurl(WebSocketClientLibcurl* client, size_t* out_len, int* out_is_text) {
    if (!client || !client->is_initialized || !client->curl_handle) return NULL;

    char buffer[4096];
    size_t nread = 0;
    const struct curl_ws_frame *frame = NULL;

    CURLcode rc = curl_ws_recv(client->curl_handle, buffer, sizeof(buffer), &nread, &frame);
    if (rc == CURLE_AGAIN) {
        return NULL; // 暂无数据
    }
    if (rc != CURLE_OK || nread == 0) {
        return NULL;
    }

    char* msg = malloc(nread + 1);
    if (!msg) return NULL;
    memcpy(msg, buffer, nread);
    msg[nread] = '\0';

    if (out_len) *out_len = nread;
    if (out_is_text) *out_is_text = (frame && (frame->flags & CURLWS_TEXT)) != 0;

    return msg;
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
