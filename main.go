package main

import (
	"cgolatencytest/http_client"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 进度指示器字符
var spinnerChars = []string{"|", "/", "-", "\\"}

// 开始进度指示器
func startSpinner(ctx context.Context, prefix string) {
	go func() {
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("\r%s 完成!\n", prefix)
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s", prefix, spinnerChars[i%len(spinnerChars)])
				i++
			}
		}
	}()
}

// 格式化延迟显示
func formatLatency(latencyNs int64) string {
	latencyMs := float64(latencyNs) / 1000000
	return fmt.Sprintf("%.2f ms", latencyMs)
}

func main() {
	fmt.Println("=== HTTP 并发延迟测试程序 ===")

	// 初始化libcurl
	ctx, cancel := context.WithCancel(context.Background())
	startSpinner(ctx, "正在初始化libcurl...")
	err := http_client.InitLibcurl()
	cancel()
	if err != nil {
		log.Fatal("libcurl初始化失败:", err)
	}
	defer http_client.CleanupLibcurl()
	fmt.Println("✓ libcurl初始化成功!")
	runCases := []struct {
		name string
		url  string
	}{
		{"BN SPOT     API", "https://api.binance.com/api/v3/ping"},
		{"BN FUTURE   API", "https://fapi.binance.com/fapi/v1/ping"},
		{"BN DELIVERY API", "https://dapi.binance.com/dapi/v1/ping"},
		// {"BN SPOT API", "https://icanhazip.com"},
		// {"BN FUTURE API", "https://icanhazip.com"},
		// {"BN DELIVERY API", "https://icanhazip.com"},
	}

	type TestResult struct {
		successCount int64
		sumLatency   int64
		mu           sync.RWMutex
	}

	resultMap := make(map[string]*TestResult)

	// 初始化resultMap
	for _, rc := range runCases {
		resultMap[rc.name] = &TestResult{}
	}

	fmt.Println("\n开始HTTP延迟测试...")
	fmt.Printf("测试目标: %s\n", strings.Join(func() []string {
		var names []string
		for _, rc := range runCases {
			names = append(names, rc.name)
		}
		return names
	}(), ", "))
	fmt.Println("=", strings.Repeat("=", 50))

	// 启动实时状态显示goroutine
	statusCtx, statusCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-statusCtx.Done():
				return
			case <-ticker.C:
				fmt.Print("\r")
				for i, rc := range runCases {
					result := resultMap[rc.name]
					result.mu.RLock()
					successCount := atomic.LoadInt64(&result.successCount)
					sumLatency := atomic.LoadInt64(&result.sumLatency)
					result.mu.RUnlock()

					if successCount > 0 {
						avgLatency := sumLatency / successCount
						fmt.Printf("%s: %d/%d (%s) ", rc.name, successCount, 1000, formatLatency(avgLatency))
					} else {
						fmt.Printf("%s: 0/1000 (等待中...) ", rc.name)
					}
					if i < len(runCases)-1 {
						fmt.Print("| ")
					}
				}
			}
		}
	}()

	var wg sync.WaitGroup
	for _, rc := range runCases {
		wg.Add(1)
		testCase := rc
		go func() {
			defer wg.Done()
			// 创建多个客户端实例
			client1, err := http_client.NewClientLibcurl()
			if err != nil {
				panic(err)
			}
			defer client1.Close()

			avgLatency := int64(0)
			for i := 0; i < 1000; i++ {
				res := client1.Get(testCase.url, 3000, 0)
				if res.Error != "" {
					continue
				}

				if res.StatusCode < 100 || res.StatusCode > 599 {
					continue
				}

				// 去除明显偏移的极值
				if avgLatency > 0 && res.LatencyNs > avgLatency*2 {
					continue
				}

				// 更新统计数据
				result := resultMap[testCase.name]
				atomic.AddInt64(&result.sumLatency, res.LatencyNs)
				atomic.AddInt64(&result.successCount, 1)
				avgLatency = atomic.LoadInt64(&result.sumLatency) / atomic.LoadInt64(&result.successCount)
			}

		}()
	}
	wg.Wait()
	statusCancel() // 停止实时状态显示

	fmt.Printf("\r%s\n", strings.Repeat(" ", 100)) // 清除状态行
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试完成! 最终结果:")
	fmt.Println(strings.Repeat("=", 60))

	for _, rc := range runCases {
		v := resultMap[rc.name]
		successCount := atomic.LoadInt64(&v.successCount)
		sumLatency := atomic.LoadInt64(&v.sumLatency)

		if successCount > 0 {
			avgLatencyNs := sumLatency / successCount
			fmt.Printf("📊 %s:\n", rc.name)
			fmt.Printf("   ✅ 成功请求: %d/1000\n", successCount)
			fmt.Printf("   ⚡ 平均延迟: %s\n", formatLatency(avgLatencyNs))
			fmt.Printf("   📈 成功率: %.1f%%\n", float64(successCount)/10.0)
			fmt.Println()
		} else {
			fmt.Printf("❌ %s: 所有请求都失败了\n", rc.name)
		}
	}

	fmt.Println("=== 🎉 HTTP延迟测试程序执行完成 ===")

	wsrunCases := []struct {
		name string
		url  string
	}{
		{"BN SPOT     WS STREAM", "wss://stream.binance.com:9443/stream?streams=btcusdt@depth@100ms"},
		{"BN FUTURE   WS STREAM", "wss://fstream.binance.com/stream?streams=btcusdt@depth@0ms"},
		{"BN DELIVERY WS STREAM", "wss://dstream.binance.com/stream?streams=btcusd_perp@depth@0ms"},
	}
	_ = wsrunCases

	fmt.Println("\n开始WebSocket延迟测试...")
	wsclient, err := http_client.NewWebSocketClientLibcurl()
	if err != nil {
		panic(err)
	}
	defer wsclient.Close()
	for _, rc := range wsrunCases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 创建多个WS客户端实例
			client, err := http_client.NewWebSocketClientLibcurl()
			if err != nil {
				panic(err)
			}
			defer client.Close()

			res := client.Connect(rc.url, 5000)
			if res.Error != "" {
				panic(res.Error)
			}
			fmt.Printf("[%s]connect to %s success: %d\n", rc.name, rc.url, res.StatusCode)

			avgLatency := int64(0)
			//接收100次消息
			for i := 0; i < 100; i++ {
				recv, ok, err := client.Recv()
				if err != nil {
					panic(err)
				}
				if !ok {
					continue
				}
				now := time.Now().UnixMilli()
				fmt.Printf("[%s]recv msg size: %s\n", rc.name, recv)
				unmarshalMap := map[string]interface{}{}
				err = json.Unmarshal([]byte(recv), &unmarshalMap)
				if err != nil {
					fmt.Printf("[%s]unmarshal error: %v\n", rc.name, err)
					continue
				}
				dataMapInterface, ok := unmarshalMap["data"]
				if !ok {
					continue
				}

				dataMap, ok := dataMapInterface.(map[string]interface{})
				if !ok {
					continue
				}

				msgTimestamp, ok := dataMap["E"]
				if !ok {
					continue
				}

				targetLatency := now - msgTimestamp.(int64)
				avgLatency = (avgLatency + targetLatency) / 2
				fmt.Printf("[%s]targetLatency: %d avgLatency: %d\n", rc.name, targetLatency, avgLatency)
			}
		}()
	}

	wg.Wait()

	fmt.Println("=== 🎉 WebSocket延迟测试程序执行完成 ===")

}
