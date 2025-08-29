#ifndef WEBSOCKET_CLIENT_LIBCURL_H
#define WEBSOCKET_CLIENT_LIBCURL_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// WebSocket 客户端句柄
typedef struct WebSocketClientLibcurl WebSocketClientLibcurl;

// WebSocket 请求结果结构
typedef struct {
    int64_t latency_ns;     // 握手耗时 (纳秒)
    int status_code;        // 连接返回的状态码 (101 表示成功)
    char* error_message;    // 错误消息 (失败时有效)
} WebSocketResultLibcurl;

// 初始化/销毁
int websocket_client_init_libcurl();
WebSocketClientLibcurl* websocket_client_new_libcurl();
void websocket_client_destroy_libcurl(WebSocketClientLibcurl* client);
void websocket_client_cleanup_libcurl();

// 建立连接
WebSocketResultLibcurl websocket_connect_libcurl(WebSocketClientLibcurl* client, const char* url, int timeout_ms);

// 发送消息
// is_text=1 表示文本消息, 0 表示二进制消息
int websocket_send_libcurl(WebSocketClientLibcurl* client, const char* msg, size_t len, int is_text);

// 接收消息
// 返回堆分配的字符串, 需要用 websocket_free_message_libcurl 释放
// out_len 返回消息长度, out_is_text=1 表示文本消息
char* websocket_recv_libcurl(WebSocketClientLibcurl* client, size_t* out_len, int* out_is_text);

// 释放辅助函数
void websocket_free_error_libcurl(char* ptr);
void websocket_free_message_libcurl(char* ptr);

#ifdef __cplusplus
}
#endif

#endif
