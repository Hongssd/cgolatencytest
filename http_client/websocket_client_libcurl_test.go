package http_client

import (
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
			if res.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", res.StatusCode)
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
	if err != nil {
		t.Errorf("Expected error after closing client, got none: %v", err)
	}
	t.Logf("code: %d", res)
}
