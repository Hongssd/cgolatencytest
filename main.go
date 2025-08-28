package main

import (
	"fmt"
	"gocpptest/http_client"
	"log"
	"sort"
	"sync"
	"time"
)

// TestResult 测试结果结构体
type TestResult struct {
	LatencyNs     int64
	StatusCode    int
	Error         string
	DNSTimeNs     int64
	ConnectTimeNs int64
	TLSTimeNs     int64
	ResponseBody  string
	ResponseSize  int
}

// 并发测试函数
func runConcurrentTest(url string, timeoutMs int, forceHttpVersion int, numRequests int) {
	fmt.Printf("开始并发测试，目标URL: %s\n", url)
	fmt.Printf("并发请求数量: %d\n", numRequests)
	fmt.Printf("超时设置: %d ms\n", timeoutMs)
	fmt.Println("==================================")

	// 创建结果通道
	resultChan := make(chan TestResult, numRequests)

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup

	// 记录开始时间
	startTime := time.Now()

	// 启动并发请求，使用信号量控制并发数
	semaphore := make(chan struct{}, 20) // 限制最大并发数为20

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行HTTP请求，最多重试2次
			var result http_client.ResultLibcurl

			for retry := 0; retry < 3; retry++ {
				result = http_client.HTTPGetRequestLibcurl(url, timeoutMs, forceHttpVersion)
				if result.Error == "" {

					fmt.Println("res body:", result.ResponseBody)
					fmt.Printf("Latency: %d ns, Status Code: %d", result.LatencyNs, result.StatusCode)
					fmt.Printf("DNS: %d ns, Connect: %d ns, TLS: %d ns",
						result.DNSTimeUs, result.ConnectTimeUs, result.TLSTimeUs)

					break
				}
				if retry < 2 {
					time.Sleep(10 * time.Millisecond) // 短暂延迟后重试
				}
			}

			// 转换为TestResult并发送到通道
			testResult := TestResult{
				LatencyNs:     result.LatencyNs,
				StatusCode:    result.StatusCode,
				Error:         result.Error,
				DNSTimeNs:     result.DNSTimeUs,
				ConnectTimeNs: result.ConnectTimeUs,
				TLSTimeNs:     result.TLSTimeUs,
				ResponseBody:  result.ResponseBody,
				ResponseSize:  result.ResponseSize,
			}

			resultChan <- testResult

			if requestID%10 == 0 {
				fmt.Printf("已完成 %d 个请求...\n", requestID+1)
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(resultChan)

	// 计算总耗时
	totalTime := time.Since(startTime)

	// 收集结果并计算统计信息
	var (
		totalLatency     int64
		totalDNS         int64
		totalConnect     int64
		totalTLS         int64
		successCount     int
		errorCount       int
		statusCodeCounts = make(map[int]int)
		errorTypes       = make(map[string]int)
		latencies        []int64
	)

	// 从通道读取所有结果
	for result := range resultChan {
		if result.Error != "" {
			errorCount++
			errorTypes[result.Error]++
		} else {
			successCount++
			totalLatency += result.LatencyNs
			totalDNS += result.DNSTimeNs
			totalConnect += result.ConnectTimeNs
			totalTLS += result.TLSTimeNs
			statusCodeCounts[result.StatusCode]++
			latencies = append(latencies, result.LatencyNs)
		}
	}

	// 计算平均值
	var avgLatency, avgDNS, avgConnect, avgTLS int64
	if successCount > 0 {
		avgLatency = totalLatency / int64(successCount)
		avgDNS = totalDNS / int64(successCount)
		avgConnect = totalConnect / int64(successCount)
		avgTLS = totalTLS / int64(successCount)
	}

	// 计算延迟统计信息
	var minLatency, maxLatency, p50Latency, p95Latency, p99Latency int64
	if len(latencies) > 0 {
		// 排序延迟数据
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		minLatency = latencies[0]
		maxLatency = latencies[len(latencies)-1]
		p50Latency = latencies[len(latencies)*50/100]
		if len(latencies) >= 20 {
			p95Latency = latencies[len(latencies)*95/100]
		}
		if len(latencies) >= 100 {
			p99Latency = latencies[len(latencies)*99/100]
		}
	}

	// 输出测试结果
	fmt.Println("\n==================================")
	fmt.Println("并发测试完成！")
	fmt.Printf("总耗时: %v\n", totalTime)
	fmt.Printf("成功请求: %d\n", successCount)
	fmt.Printf("失败请求: %d\n", errorCount)
	fmt.Printf("成功率: %.2f%%\n", float64(successCount)/float64(numRequests)*100)
	fmt.Printf("QPS: %.2f\n", float64(successCount)/totalTime.Seconds())

	if successCount > 0 {
		fmt.Println("\n延迟统计 (毫秒):")
		fmt.Printf("  平均延迟: %.2f\n", float64(avgLatency)/1e6)
		fmt.Printf("  最小延迟: %.2f\n", float64(minLatency)/1e6)
		fmt.Printf("  最大延迟: %.2f\n", float64(maxLatency)/1e6)
		fmt.Printf("  P50延迟: %.2f\n", float64(p50Latency)/1e6)
		fmt.Printf("  P95延迟: %.2f\n", float64(p95Latency)/1e6)
		fmt.Printf("  P99延迟: %.2f\n", float64(p99Latency)/1e6)

		fmt.Println("\n详细延迟分析:")
		fmt.Printf("  DNS解析: %.2f ms\n", float64(avgDNS)/1e6)
		fmt.Printf("  连接建立: %.2f ms\n", float64(avgConnect)/1e6)
		fmt.Printf("  TLS握手: %.2f ms\n", float64(avgTLS)/1e6)

		fmt.Println("\n状态码分布:")
		for statusCode, count := range statusCodeCounts {
			fmt.Printf("  %d: %d 次\n", statusCode, count)
		}
	}

	if errorCount > 0 {
		fmt.Println("\n错误类型分布:")
		for errorType, count := range errorTypes {
			fmt.Printf("  %s: %d 次\n", errorType, count)
		}
	}

	fmt.Println("==================================")
}

func main() {
	fmt.Println("=== HTTP 并发延迟测试程序 ===")

	// 初始化libcurl
	fmt.Println("正在初始化libcurl...")
	err := http_client.InitLibcurl()
	if err != nil {
		log.Fatal("libcurl初始化失败:", err)
	}
	defer http_client.CleanupLibcurl()

	// 测试配置 - 使用更保守的参数
	testURL := "https://fapi.binance.com/fapi/v1/time"
	timeoutMs := 10000 // 增加到10秒
	forceHttpVersion := 1
	numRequests := 10 // 减少到10个请求

	//测试一次
	runConcurrentTest(testURL, timeoutMs, forceHttpVersion, 2)

	// 运行并发测试
	// runConcurrentTest(testURL, timeoutMs, forceHttpVersion, numRequests)
	_ = numRequests
	fmt.Println("=== 测试程序执行完成 ===")
}
