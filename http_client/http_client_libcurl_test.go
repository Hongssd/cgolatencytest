package http_client

import (
	"testing"
)

// 测试新的客户端接口
func TestNewClientInterface(t *testing.T) {
	// 初始化 libcurl 客户端
	if err := InitLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupLibcurl()

	// 创建新的客户端实例
	client, err := NewClientLibcurl()
	if err != nil {
		t.Fatalf("NewClientLibcurl failed: %v", err)
	}
	defer client.Close()

	t.Run("Test GET request", func(t *testing.T) {
		res := client.Get("https://api.binance.com/api/v3/ping", 10000, 0)
		if res.Error != "" {
			t.Logf("GET request returned error: %s", res.Error)
		} else {
			t.Logf("GET request successful: Status=%d, Latency=%d ns", res.StatusCode, res.LatencyNs)
			if res.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", res.StatusCode)
			}
		}
	})
}

// 测试客户端生命周期管理
func TestClientLifecycle(t *testing.T) {
	if err := InitLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupLibcurl()

	// 创建多个客户端实例
	client1, err := NewClientLibcurl()
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}

	client2, err := NewClientLibcurl()
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}

	// 测试两个客户端可以独立工作
	res1 := client1.Get("https://icanhazip.com", 5000, 0)
	res2 := client2.Get("https://icanhazip.com", 5000, 0)

	if res1.Error != "" && res2.Error != "" {
		t.Logf("Both clients had errors (network issues): %s, %s", res1.Error, res2.Error)
	} else {
		t.Logf("At least one client worked successfully")
	}

	// 关闭客户端
	client1.Close()
	client2.Close()

	// 测试关闭后的行为
	res := client1.Get("https://icanhazip.com", 5000, 0)
	if res.Error == "" {
		t.Errorf("Expected error after closing client, got none")
	}
}
