package http_client

import (
	"fmt"
	"testing"
)

func TestHTTPHeadRequestLibcurl(t *testing.T) {
	// 初始化 libcurl 客户端
	if err := InitLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupLibcurl()

	testCases := []struct {
		name string
		url  string
	}{
		{"HTTP icanhazip.com", "http://icanhazip.com"},
		{"HTTPS icanhazip.com", "https://icanhazip.com"},
		{"HTTP www.google.com", "http://www.google.com/"},
		{"HTTPS www.google.com", "https://www.google.com/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := HTTPGetRequestLibcurl(tc.url, 5000, 0)
			if res.Error != "" {
				t.Logf("Request returned error: %s", res.Error)
			} else {
				t.Logf("Latency: %d us, Status Code: %d", res.LatencyNs, res.StatusCode)
				t.Logf("DNS: %d us, Connect: %d us, TLS: %d us",
					res.DNSTimeUs, res.ConnectTimeUs, res.TLSTimeUs)

				if res.StatusCode < 100 || res.StatusCode > 599 {
					t.Errorf("Invalid HTTP status code: %d", res.StatusCode)
				}
			}
		})
	}
}

func TestInvalidHostLibcurl(t *testing.T) {
	if err := InitLibcurl(); err != nil {
		t.Fatalf("InitLibcurl failed: %v", err)
	}
	defer CleanupLibcurl()

	res := HTTPHeadRequestLibcurl("http://invalid.host", 3000, 0)
	if res.Error == "" {
		t.Errorf("Expected error for invalid host, got nil")
	} else {
		fmt.Println("Got expected error:", res.Error)
	}
}
