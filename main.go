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
		name           string
		url            string
		serverTimeUrl  string
		serverTimeDiff int64
	}{
		{"BN SPOT      API", "https://api.binance.com/api/v3/ping", "https://api.binance.com/api/v3/time", 0},
		{"BN FUTURE    API", "https://fapi.binance.com/fapi/v1/ping", "https://fapi.binance.com/fapi/v1/time", 0},
		{"BN DELIVERY  API", "https://dapi.binance.com/dapi/v1/ping", "https://dapi.binance.com/dapi/v1/time", 0},
		{"BN PORTFOLIO API", "https://papi.binance.com/papi/v1/ping", "", 0},
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
		rc := rc
		go func() {
			defer wg.Done()
			// 创建多个客户端实例
			client1, err := http_client.NewClientLibcurl()
			if err != nil {
				panic(err)
			}
			defer client1.Close()

			fmt.Println("开始获取服务器时间差...")
			//若serverTimeUrl不为空字符串 请求五十次serverTime 排除异常值，取均值
			if rc.serverTimeUrl != "" {
				serverTimeDiffSum := int64(0)
				serverTimeSuccessCount := int64(0)
				for i := 0; i < 50; i++ {
					serverTimeRes := client1.Get(rc.serverTimeUrl, 3000, 0)
					if serverTimeRes.Error != "" {
						log.Printf("[%s] 获取服务器时间差失败: %s", rc.name, serverTimeRes.Error)
						continue
					}
					if serverTimeRes.StatusCode == 200 {
						serverTimeBodyMap := map[string]interface{}{}
						err := json.Unmarshal([]byte(serverTimeRes.ResponseBody), &serverTimeBodyMap)
						if err != nil {
							log.Printf("[%s] 解析服务器时间差失败: [res:%s]%v", rc.name, serverTimeRes.ResponseBody, serverTimeRes.ResponseBody)
							continue
						}
						serverTimeTimestampInterface, ok := serverTimeBodyMap["serverTime"]
						if !ok {
							continue
						}
						//获取服务器毫秒时间戳
						serverTimeTimestamp := int64(serverTimeTimestampInterface.(float64))

						//转为纳秒时间戳
						serverTimeTimestampNs := serverTimeTimestamp * 1000000

						//获取请求的开始纳秒时间戳
						requestStartTimestampNs := serverTimeRes.RequestTimeNs

						//计算请求结束时的纳秒时间戳
						requestEndTimestampNs := serverTimeRes.ResponseTimeNs

						//计算请求一个来回的中间点纳秒时间戳
						requestMidTimestampNs := (requestEndTimestampNs + requestStartTimestampNs) / 2

						//计算服务器时间差
						serverTimeDiffNs := serverTimeTimestampNs - requestMidTimestampNs

						fmt.Printf("[%s] 服务器时间戳: %d ns ≈ %.3f us ≈ %.6f ms 本地请求中点时间戳: %d ns ≈ %.3f us ≈ %.6f ms\n",
							rc.name,
							serverTimeTimestampNs,
							float64(serverTimeTimestampNs)/1000,
							float64(serverTimeTimestampNs)/1000000,
							requestMidTimestampNs,
							float64(requestMidTimestampNs)/1000,
							float64(requestMidTimestampNs)/1000000)

						//累加服务器时间差纳秒
						serverTimeDiffSum += serverTimeDiffNs
						serverTimeSuccessCount++
					}
				}

				//计算服务器时间差纳秒均值
				serverTimeDiffAvgNs := serverTimeDiffSum / serverTimeSuccessCount
				rc.serverTimeDiff = serverTimeDiffAvgNs
				fmt.Printf("[%s] 服务器时间差: %d ns ≈ %.3f us ≈ %.6f ms\n",
					rc.name, serverTimeDiffAvgNs, float64(serverTimeDiffAvgNs)/1000, float64(serverTimeDiffAvgNs)/1000000)
			}

			avgLatency := int64(0)
			for i := 0; i < 1000; i++ {
				res := client1.Get(rc.url, 3000, 0)
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
				result := resultMap[rc.name]
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

	// ============================
	// WebSocket 延迟测试部分
	// ============================
	wsrunCases := []struct {
		name string
		url  string
	}{
		{"BN SPOT      WS STREAM", "wss://stream.binance.com:9443/stream?streams=btcusdt@depth@100ms"},
		{"BN FUTURE    WS STREAM", "wss://fstream.binance.com/stream?streams=btcusdt@depth@0ms"},
		{"BN DELIVERY  WS STREAM", "wss://dstream.binance.com/stream?streams=btcusd_perp@depth@0ms"},
	}

	// 初始化WebSocket libcurl
	wsCtx, wsCancel := context.WithCancel(context.Background())
	startSpinner(wsCtx, "正在初始化WebSocket libcurl...")
	err = http_client.InitWebSocketLibcurl()
	wsCancel()
	if err != nil {
		log.Fatal("WebSocket libcurl初始化失败:", err)
	}
	defer http_client.CleanupWebSocketLibcurl()
	fmt.Println("✓ WebSocket libcurl初始化成功!")

	wsResultMap := make(map[string]*TestResult)

	// 初始化wsResultMap
	for _, rc := range wsrunCases {
		wsResultMap[rc.name] = &TestResult{}
	}

	fmt.Println("\n开始WebSocket延迟测试...")
	fmt.Printf("测试目标: %s\n", strings.Join(func() []string {
		var names []string
		for _, rc := range wsrunCases {
			names = append(names, rc.name)
		}
		return names
	}(), ", "))
	fmt.Println("=", strings.Repeat("=", 50))

	// 启动WebSocket实时状态显示goroutine
	wsStatusCtx, wsStatusCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-wsStatusCtx.Done():
				return
			case <-ticker.C:
				fmt.Print("\r")
				for i, rc := range wsrunCases {
					result := wsResultMap[rc.name]
					result.mu.RLock()
					successCount := atomic.LoadInt64(&result.successCount)
					sumLatency := atomic.LoadInt64(&result.sumLatency)
					result.mu.RUnlock()

					if successCount > 0 {
						avgLatency := sumLatency / successCount
						fmt.Printf("%s: %d/%d (%s) ", rc.name, successCount, 1000, formatLatency(avgLatency))
					} else {
						fmt.Printf("%s: 0/1000 (连接中...) ", rc.name)
					}
					if i < len(wsrunCases)-1 {
						fmt.Print("| ")
					}
				}
			}
		}
	}()

	for _, rc := range wsrunCases {
		wg.Add(1)
		rc := rc
		go func() {
			defer wg.Done()
			// 创建WebSocket客户端实例
			client, err := http_client.NewWebSocketClientLibcurl()
			if err != nil {
				log.Printf("[%s] 创建客户端失败: %v", rc.name, err)
				return
			}
			defer client.Close()

			// 建立连接
			res := client.Connect(rc.url, 5000)
			if res.Error != "" {
				log.Printf("[%s] 连接失败: %s", rc.name, res.Error)
				return
			}
			// 连接成功后不再单独打印，由状态显示器统一显示

			avgLatency := int64(0)
			//接收1000次消息
			for {
				// 检查是否已完成1000次
				result := wsResultMap[rc.name]
				if atomic.LoadInt64(&result.successCount) >= 1000 {
					break
				}

				recv, ok, err := client.Recv()
				if err != nil {
					log.Printf("[%s] 接收消息失败: %v", rc.name, err)
					continue
				}
				if !ok {
					// 暂时无消息，继续等待
					continue
				}

				now := time.Now().UnixNano()
				unmarshalMap := map[string]interface{}{}
				err = json.Unmarshal([]byte(recv), &unmarshalMap)
				if err != nil {
					continue // 跳过无效消息
				}

				// 解析消息时间戳
				dataMapInterface, ok := unmarshalMap["data"]
				if !ok {
					continue
				}

				dataMap, ok := dataMapInterface.(map[string]interface{})
				if !ok {
					continue
				}

				msgTimestampInterface, ok := dataMap["E"]
				if !ok {
					continue
				}
				msgTimestamp := int64(msgTimestampInterface.(float64))
				//毫秒转纳秒
				msgTimestampNano := msgTimestamp * 1000000

				targetLatency := now - msgTimestampNano

				//去除明显偏移的极值
				if avgLatency > 0 && targetLatency > avgLatency*2 {
					continue
				}

				// 更新统计数据
				atomic.AddInt64(&result.sumLatency, targetLatency)
				atomic.AddInt64(&result.successCount, 1)
				avgLatency = atomic.LoadInt64(&result.sumLatency) / atomic.LoadInt64(&result.successCount)
			}
		}()
	}

	wg.Wait()
	wsStatusCancel() // 停止WebSocket实时状态显示

	fmt.Printf("\r%s\n", strings.Repeat(" ", 100)) // 清除状态行
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("WebSocket测试完成! 最终结果:")
	fmt.Println(strings.Repeat("=", 60))

	for _, rc := range wsrunCases {
		v := wsResultMap[rc.name]
		successCount := atomic.LoadInt64(&v.successCount)
		sumLatency := atomic.LoadInt64(&v.sumLatency)

		if successCount > 0 {
			avgLatencyNs := sumLatency / successCount
			fmt.Printf("📊 %s:\n", rc.name)
			fmt.Printf("   ✅ 成功接收: %d/1000\n", successCount)
			fmt.Printf("   ⚡ 平均延迟: %s\n", formatLatency(avgLatencyNs))
			fmt.Printf("   📈 成功率: %.1f%%\n", float64(successCount)/10.0)
			fmt.Println()
		} else {
			fmt.Printf("❌ %s: 所有WebSocket消息接收都失败了\n", rc.name)
		}
	}

	fmt.Println("=== 🎉 WebSocket延迟测试程序执行完成 ===")

}
