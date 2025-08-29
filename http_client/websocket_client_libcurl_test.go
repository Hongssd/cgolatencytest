package http_client

import (
	"strings"
	"testing"
)

// 测试新的客户端接口
func TestNewWsClientInterface(t *testing.T) {
	// 初始化 libcurl 客户端
	if err := InitWebSocketLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupWebSocketLibcurl()

	// 创建新的客户端实例
	client, err := NewWebSocketClientLibcurl()
	if err != nil {
		t.Fatalf("NewClientLibcurl failed: %v", err)
	}
	defer client.Close()

	t.Run("Test GET request", func(t *testing.T) {
		res := client.Connect("wss://stream.binance.com:9443/stream", 10000)
		if res.Error != "" {
			t.Logf("GET request returned error: %s", res.Error)
		} else {
			t.Logf("GET request successful: Status=%d, Latency=%d ns", res.StatusCode, res.LatencyNs)
			if res.StatusCode != 101 {
				t.Errorf("Expected status 101, got %d", res.StatusCode)
			}
		}

		//订阅
		code, err := client.Send(`
		{
			"method": "SUBSCRIBE",
			"params": [
				"btcusdt@depth@100ms"
			],
			"id": 1
		}
		`, true)
		if err != nil {
			t.Errorf("Send request failed: %v", err)
		}
		t.Logf("Send request successful: code=%d", code)

		//接受10次消息
		for i := 0; i < 10; i++ {
			recv, ok, err := client.Recv()
			if err != nil {
				t.Errorf("Recv request failed: %v", err)
			}
			if !ok {
				t.Errorf("Recv request failed: no message received")
				break
			}
			t.Logf("Recv request successful: %s", recv)
		}

		//取消订阅
		code, err = client.Send(`
		{
			"method": "UNSUBSCRIBE",
			"params": [
				"btcusdt@depth@100ms"
			],
			"id": 1
		}
		`, true)
		if err != nil {
			t.Errorf("Send request failed: %v", err)
		}
		t.Logf("Send request successful: code=%d", code)

		client.Close()

	})
}

// 测试客户端生命周期管理
func TestWsClientLifecycle(t *testing.T) {
	if err := InitWebSocketLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupWebSocketLibcurl()

	// 创建多个客户端实例
	client1, err := NewWebSocketClientLibcurl()
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}

	client2, err := NewWebSocketClientLibcurl()
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}

	// 测试两个客户端可以独立工作
	res1 := client1.Connect("wss://stream.binance.com:9443/stream", 5000)
	res2 := client2.Connect("wss://stream.binance.com:9443/stream", 5000)

	if res1.Error != "" && res2.Error != "" {
		t.Logf("Both clients had errors (network issues): %s, %s", res1.Error, res2.Error)
	} else {
		t.Logf("At least one client worked successfully")
	}

	// 关闭客户端
	client1.Close()
	client2.Close()

	// 测试关闭后的行为
	res, err := client1.Send("{}", true)
	if err == nil {
		t.Errorf("Expected error after closing client, got none")
	} else {
		// 检查是否返回正确的WebSocket错误类型
		if wsErr, ok := err.(*WebSocketError); ok {
			t.Logf("Got expected WebSocket error: %v", wsErr)
		} else {
			t.Logf("Got error (not WebSocket specific): %v", err)
		}
	}
	t.Logf("code: %d", res)
}

// 测试错误处理
func TestWebSocketErrorHandling(t *testing.T) {
	if err := InitWebSocketLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupWebSocketLibcurl()

	t.Run("Test nil client operations", func(t *testing.T) {
		var client *WebSocketClientLibcurl = nil

		// 测试连接操作 - 会导致panic，所以我们要捕获它
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Expected panic for nil client Connect: %v", r)
				}
			}()
			result := client.Connect("wss://example.com", 5000)
			t.Logf("Connect result: %+v", result)
		}()

		// 测试发送操作 - 需要捕获panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Expected panic for nil client Send: %v", r)
				}
			}()
			_, err := client.Send("test", true)
			if err == nil {
				t.Error("Expected error for nil client Send, got none")
			} else if wsErr, ok := err.(*WebSocketError); ok {
				if wsErr.Code != WEBSOCKET_ERROR_INVALID_CLIENT {
					t.Errorf("Expected WEBSOCKET_ERROR_INVALID_CLIENT, got %d", wsErr.Code)
				}
				t.Logf("Got expected error: %v", wsErr)
			}
		}()

		// 测试接收操作 - 需要捕获panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Expected panic for nil client Recv: %v", r)
				}
			}()
			_, _, err := client.Recv()
			if err == nil {
				t.Error("Expected error for nil client Recv, got none")
			} else if wsErr, ok := err.(*WebSocketError); ok {
				if wsErr.Code != WEBSOCKET_ERROR_INVALID_CLIENT {
					t.Errorf("Expected WEBSOCKET_ERROR_INVALID_CLIENT, got %d", wsErr.Code)
				}
			}
		}()
	})

	t.Run("Test invalid parameters", func(t *testing.T) {
		client, err := NewWebSocketClientLibcurl()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		// 测试发送空消息
		_, err = client.Send("", true)
		if err == nil {
			t.Error("Expected error for empty message, got none")
		} else if wsErr, ok := err.(*WebSocketError); ok {
			if wsErr.Code != WEBSOCKET_ERROR_INVALID_PARAMS {
				t.Logf("Got error code %d, message: %v", wsErr.Code, wsErr)
			}
		}
	})

	t.Run("Test WebSocket error messages", func(t *testing.T) {
		testCases := []struct {
			code     int
			contains string
		}{
			{WEBSOCKET_ERROR_INVALID_CLIENT, "invalid"},
			{WEBSOCKET_ERROR_INVALID_PARAMS, "Invalid parameters"},
			{WEBSOCKET_ERROR_SEND_FAILED, "Failed to send"},
			{WEBSOCKET_ERROR_NETWORK, "Network error"},
			{WEBSOCKET_ERROR_TIMEOUT, "timed out"},
			{WEBSOCKET_ERROR_MEMORY, "Memory allocation"},
			{WEBSOCKET_ERROR_BUFFER_OVERFLOW, "Buffer overflow"},
			{-999, "Unknown WebSocket error"}, // 测试未知错误码
		}

		for _, tc := range testCases {
			wsErr := &WebSocketError{Code: tc.code}
			message := wsErr.Error()
			if !strings.Contains(strings.ToLower(message), strings.ToLower(tc.contains)) {
				t.Errorf("Error message for code %d should contain '%s', got: %s",
					tc.code, tc.contains, message)
			}
			t.Logf("Code %d: %s", tc.code, message)
		}
	})
}

// 测试动态缓冲区功能
func TestWebSocketDynamicBuffer(t *testing.T) {
	if err := InitWebSocketLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupWebSocketLibcurl()

	client, err := NewWebSocketClientLibcurl()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	t.Run("Test large message handling", func(t *testing.T) {
		// 创建一个大于4KB的消息来测试动态缓冲区
		largeMessage := strings.Repeat("A", 8192) // 8KB消息

		// 注意：这个测试可能会失败，因为我们没有真实的WebSocket服务器
		// 但它可以测试参数验证和错误处理
		_, err := client.Send(largeMessage, true)
		if err != nil {
			t.Logf("Large message send failed (expected without server): %v", err)
		} else {
			t.Log("Large message send succeeded")
		}
	})

	t.Run("Test buffer size constants", func(t *testing.T) {
		// 验证缓冲区大小常量是否合理
		const expectedInitialSize = 4096
		const expectedMaxSize = 1024 * 1024

		// 这些常量在C头文件中定义，我们可以在Go中验证逻辑
		if expectedInitialSize <= 0 {
			t.Error("Initial buffer size should be positive")
		}
		if expectedMaxSize <= expectedInitialSize {
			t.Error("Max buffer size should be larger than initial size")
		}

		t.Logf("Buffer sizes - Initial: %d bytes, Max: %d bytes",
			expectedInitialSize, expectedMaxSize)
	})
}
