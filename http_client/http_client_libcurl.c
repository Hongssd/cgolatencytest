#include "http_client_libcurl.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <curl/curl.h>

struct HttpClientLibcurl {
    CURL* curl_handle;
    int is_initialized;
};

static int global_curl_initialized = 0;

typedef struct {
    char* data;
    size_t size;
} ResponseData;

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

static int should_use_http2(const char* url) {
    if (!url) return 0;
    if (strncmp(url, "wss://", 6) == 0 || strncmp(url, "https://", 8) == 0) {
        return 1;
    }
    return 0;
}

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

int http_client_init_libcurl() {
    if (global_curl_initialized) return 0;
    if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK) return -1;
    global_curl_initialized = 1;
    return 0;
}

HttpClientLibcurl* http_client_new_libcurl() {
    if (!global_curl_initialized && http_client_init_libcurl() != 0) return NULL;
    
    HttpClientLibcurl* client = malloc(sizeof(HttpClientLibcurl));
    if (!client) return NULL;
    
    client->curl_handle = curl_easy_init();
    if (!client->curl_handle) {
        free(client);
        return NULL;
    }
    
    client->is_initialized = 1;
    return client;
}

HttpResultLibcurl http_request_libcurl(HttpClientLibcurl* client, const char* url, int timeout_ms, 
                                      int force_http_version, HttpMethod method,
                                      const char* post_data, const char** headers) {
    HttpResultLibcurl result = {0};
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
    curl_easy_setopt(client->curl_handle, CURLOPT_TIMEOUT_MS, (long)timeout_ms);
    curl_easy_setopt(client->curl_handle, CURLOPT_CONNECTTIMEOUT_MS, (long)(timeout_ms / 2));
    curl_easy_setopt(client->curl_handle, CURLOPT_USERAGENT, "HTTPLatencyTest/1.0");
    
    if (force_http_version == 0) {
        curl_easy_setopt(client->curl_handle, CURLOPT_HTTP_VERSION, 
                        should_use_http2(url) ? CURL_HTTP_VERSION_2_0 : CURL_HTTP_VERSION_1_1);
    } else if (force_http_version == 1) {
        curl_easy_setopt(client->curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_1_1);
    } else if (force_http_version == 2) {
        curl_easy_setopt(client->curl_handle, CURLOPT_HTTP_VERSION, CURL_HTTP_VERSION_2_0);
    }
    
    switch (method) {
        case HTTP_METHOD_HEAD:
            curl_easy_setopt(client->curl_handle, CURLOPT_HEADER, 1L);
            curl_easy_setopt(client->curl_handle, CURLOPT_NOBODY, 1L);
            break;
        case HTTP_METHOD_GET:
            curl_easy_setopt(client->curl_handle, CURLOPT_HTTPGET, 1L);
            break;
        case HTTP_METHOD_POST:
            curl_easy_setopt(client->curl_handle, CURLOPT_POST, 1L);
            if (post_data) curl_easy_setopt(client->curl_handle, CURLOPT_POSTFIELDS, post_data);
            break;
        case HTTP_METHOD_PUT:
            curl_easy_setopt(client->curl_handle, CURLOPT_CUSTOMREQUEST, "PUT");
            if (post_data) curl_easy_setopt(client->curl_handle, CURLOPT_POSTFIELDS, post_data);
            break;
        case HTTP_METHOD_DELETE:
            curl_easy_setopt(client->curl_handle, CURLOPT_CUSTOMREQUEST, "DELETE");
            break;
        case HTTP_METHOD_PATCH:
            curl_easy_setopt(client->curl_handle, CURLOPT_CUSTOMREQUEST, "PATCH");
            if (post_data) curl_easy_setopt(client->curl_handle, CURLOPT_POSTFIELDS, post_data);
            break;
    }
    
    if (headers) {
        struct curl_slist* header_list = NULL;
        for (int i = 0; headers[i] != NULL; i++) {
            header_list = curl_slist_append(header_list, headers[i]);
        }
        if (header_list) {
            curl_easy_setopt(client->curl_handle, CURLOPT_HTTPHEADER, header_list);
        }
    }
    
    ResponseData resp = {0};
    curl_easy_setopt(client->curl_handle, CURLOPT_WRITEFUNCTION, write_callback);
    curl_easy_setopt(client->curl_handle, CURLOPT_WRITEDATA, &resp);
    
    CURLcode res = curl_easy_perform(client->curl_handle);
    
    if (res == CURLE_OK) {
        long response_code;
        curl_easy_getinfo(client->curl_handle, CURLINFO_RESPONSE_CODE, &response_code);
        result.status_code = (int)response_code;
        result.latency_ns = get_time_ns() - start_time;
        
        if (resp.data && resp.size > 0) {
            result.response_body = resp.data;
            result.response_size = resp.size;
        } else {
            free(resp.data);
        }
        
        curl_off_t dns_time, connect_time, app_connect_time;
        curl_easy_getinfo(client->curl_handle, CURLINFO_NAMELOOKUP_TIME_T, &dns_time);
        curl_easy_getinfo(client->curl_handle, CURLINFO_CONNECT_TIME_T, &connect_time);
        curl_easy_getinfo(client->curl_handle, CURLINFO_APPCONNECT_TIME_T, &app_connect_time);
        
        result.dns_time_ns = (int64_t)(dns_time * 1000);
        result.connect_time_ns = (int64_t)(connect_time * 1000);
        result.tls_time_ns = (int64_t)(app_connect_time * 1000);
    } else {
        result.error_message = make_error(curl_easy_strerror(res));
        free(resp.data);
    }
    
    return result;
}

void http_free_error_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

void http_free_response_libcurl(char* ptr) {
    if (ptr) free(ptr);
}

void http_client_destroy_libcurl(HttpClientLibcurl* client) {
    if (client) {
        if (client->curl_handle) {
            curl_easy_cleanup(client->curl_handle);
            client->curl_handle = NULL;
        }
        client->is_initialized = 0;
        free(client);
    }
}

void http_client_cleanup_libcurl() {
    if (global_curl_initialized) {
        curl_global_cleanup();
        global_curl_initialized = 0;
    }
}
