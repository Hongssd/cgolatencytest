package main

import (
	"cgolatencytest/http_client"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

func main() {
	fmt.Println("=== HTTP 并发延迟测试程序 ===")

	// 初始化libcurl
	fmt.Println("正在初始化libcurl...")
	err := http_client.InitLibcurl()
	if err != nil {
		log.Fatal("libcurl初始化失败:", err)
	}
	defer http_client.CleanupLibcurl()
	runCases := []struct {
		name string
		url  string
	}{
		{"BN SPOT API", "https://api.binance.com/sapi/v1/ping"},
		{"BN FUTURE API", "https://fapi.binance.com/fapi/v1/ping"},
		{"BN DELIVERY API", "https://dapi.binance.com/dapi/v1/ping"},
		// {"BN SPOT API", "https://icanhazip.com"},
		// {"BN FUTURE API", "https://icanhazip.com"},
		// {"BN DELIVERY API", "https://icanhazip.com"},
	}

	resultMap := map[string]struct {
		successCount int64
		sumLatency   int64
	}{}
	var wg sync.WaitGroup
	for _, rc := range runCases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 创建多个客户端实例
			client1, err := http_client.NewClientLibcurl()
			if err != nil {
				panic(err)
			}
			defer client1.Close()
			sumLatency := int64(0)
			successCount := int64(0)
			for i := 0; i < 1000; i++ {

				res := client1.Get(rc.url, 3000, 0)
				if res.Error != "" {
					continue
				}

				if res.StatusCode < 100 || res.StatusCode > 599 {
					continue
				}
				// fmt.Printf("Target URL: %s\n", rc.url)
				fmt.Println(res)
				// fmt.Printf("Latency: %d ns, Status Code: %d\n", res.LatencyNs, res.StatusCode)
				// fmt.Printf("DNS: %d ns, Connect: %d ns, TLS: %d ns\n",
				// res.DNSTimeNs, res.ConnectTimeNs, res.TLSTimeNs)
				atomic.AddInt64(&sumLatency, res.LatencyNs)
				atomic.AddInt64(&successCount, 1)
			}
			resultMap[rc.name] = struct {
				successCount int64
				sumLatency   int64
			}{
				successCount: successCount,
				sumLatency:   sumLatency,
			}
		}()
	}
	wg.Wait()
	for k, v := range resultMap {
		fmt.Printf("Target: %s 成功请求次数：%d 平均延迟：%d ns ≈ %.8f ms\n", k, v.successCount, v.sumLatency/v.successCount, float64(v.sumLatency)/float64(v.successCount)/1000000)
	}

	fmt.Println("=== 测试程序执行完成 ===")
}
