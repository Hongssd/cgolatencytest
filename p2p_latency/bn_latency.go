package p2p_latency

import (
	"cgolatencytest/http_client"
	"cgolatencytest/mylog"

	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// 进度指示器字符
var log = mylog.Log
var json = jsoniter.ConfigCompatibleWithStandardLibrary

type BnLatencyResult struct {
	HttpBinanceSpotLatencyNs      int64 //BN SPOT HTTP 纳秒延迟
	HttpBinanceFutureLatencyNs    int64 //BN FUTURE HTTP 纳秒延迟
	HttpBinanceDeliveryLatencyNs  int64 //BN DELIVERY HTTP 纳秒延迟
	HttpBinancePortfolioLatencyNs int64 //BN PORTFOLIO HTTP 纳秒延迟
	WsBinanceSpotLatencyNs        int64 //BN SPOT WS 纳秒延迟
	WsBinanceFutureLatencyNs      int64 //BN FUTURE WS 纳秒延迟
	WsBinanceDeliveryLatencyNs    int64 //BN DELIVERY WS 纳秒延迟
}

func TestBinanceHttpAndWsLatency() (*BnLatencyResult, error) {
	log.Debug("开始测试Binance HTTP和WebSocket延迟...")

	err := http_client.InitLibcurl()
	if err != nil {
		log.Error("libcurl初始化失败:", err)
		return nil, err
	}
	defer http_client.CleanupLibcurl()
	runCases := []struct {
		name           string
		url            string
		serverTimeUrl  string
		serverTimeDiff int64
	}{
		{"BN SPOT      API", "https://api4.binance.com/api/v3/ping", "https://api4.binance.com/api/v3/time", 0},
		{"BN FUTURE    API", "https://fapi-mm.binance.com/fapi/v1/ping", "https://fapi-mm.binance.com/fapi/v1/time", 0},
		{"BN DELIVERY  API", "https://dapi-mm.binance.com/dapi/v1/ping", "https://dapi-mm.binance.com/dapi/v1/time", 0},
		{"BN PORTFOLIO API", "https://papi.binance.com/papi/v1/ping", "", 0},
	}

	type TestResult struct {
		successCount int64
		sumLatency   int64
		avgLatency   int64
	}

	resultMap := make(map[string]*TestResult)

	// 初始化resultMap
	for _, rc := range runCases {
		resultMap[rc.name] = &TestResult{}
	}

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

			//若serverTimeUrl不为空字符串 请求一百次次serverTime 取均值
			if rc.serverTimeUrl != "" {
				serverTimeDiffSum := int64(0)
				serverTimeSuccessCount := int64(0)

				for i := 0; i < 10; i++ {
					serverTimeRes := client1.Get(rc.serverTimeUrl, 3000, 0)
					if serverTimeRes.Error != "" {
						log.Errorf("[%s] 获取服务器时间差失败: %s", rc.name, serverTimeRes.Error)
						continue
					}

					if serverTimeRes.StatusCode == 200 {
						serverTimeBodyMap := map[string]interface{}{}
						err := json.Unmarshal([]byte(serverTimeRes.ResponseBody), &serverTimeBodyMap)
						if err != nil {
							log.Errorf("[%s] 解析服务器时间差失败: [res:%s]%v", rc.name, serverTimeRes.ResponseBody, serverTimeRes.ResponseBody)
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
						serverTimeDiffNs := requestMidTimestampNs - serverTimeTimestampNs

						//累加服务器时间差纳秒
						serverTimeDiffSum += serverTimeDiffNs
						serverTimeSuccessCount++
					}
				}

				//计算服务器时间差纳秒均值
				serverTimeDiffAvgNs := int64(0)
				if serverTimeSuccessCount > 0 {
					serverTimeDiffAvgNs = serverTimeDiffSum / serverTimeSuccessCount
				}
				//rc是局部变量，需要修改原值
				for i, rc2 := range runCases {
					if rc2.name == rc.name {
						runCases[i].serverTimeDiff = serverTimeDiffAvgNs
						break
					}
				}
				log.Infof("[%s] 服务器%d次请求平均时间差: %d ns ≈ %.3f us ≈ %.6f ms\n",
					rc.name, serverTimeSuccessCount, serverTimeDiffAvgNs, float64(serverTimeDiffAvgNs)/1000, float64(serverTimeDiffAvgNs)/1000000)
			}

			time.Sleep(5 * time.Second)
			avgLatency := int64(0)
			for i := 0; i < 100; i++ {
				res := client1.Get(rc.url, 3000, 0)
				if res.Error != "" {
					continue
				}

				if res.StatusCode < 100 || res.StatusCode > 599 {
					continue
				}

				// // 去除明显偏移的极值
				// if avgLatency > 0 && res.LatencyNs > avgLatency*2 {
				// 	continue
				// }

				// 更新统计数据
				result := resultMap[rc.name]
				atomic.AddInt64(&result.sumLatency, res.LatencyNs)
				atomic.AddInt64(&result.successCount, 1)
				avgLatency = atomic.LoadInt64(&result.sumLatency) / atomic.LoadInt64(&result.successCount)
				atomic.StoreInt64(&result.avgLatency, avgLatency)
			}

		}()
	}
	wg.Wait()

	// ============================
	// WebSocket 延迟测试部分
	// ============================
	wsrunCases := []struct {
		name           string
		url            string
		serverTimeDiff int64
	}{
		{"BN SPOT      WS STREAM", "wss://stream.binance.com:9443/stream?streams=btcusdt@depth@100ms/ethusdt@depth@100ms/solusdt@depth@100ms", 0},
		{"BN FUTURE    WS STREAM", "wss://fstream-mm.binance.com/stream?streams=btcusdt@depth@0ms/ethusdt@depth@0ms", 0},
		{"BN DELIVERY  WS STREAM", "wss://dstream-mm.binance.com/stream?streams=btcusd_perp@depth@0ms", 0},
		// {"BN FUTURE    WS STREAM", "wss://fstream-mm.binance.com/stream", 0},
		// {"BN DELIVERY  WS STREAM", "wss://dstream-mm.binance.com/stream", 0},
	}

	for rci, rc := range wsrunCases {
		for _, rc2 := range runCases {
			if rc.name[:4] == rc2.name[:4] {
				wsrunCases[rci].serverTimeDiff = rc2.serverTimeDiff
			}
		}
	}
	time.Sleep(2 * time.Second)

	// 初始化WebSocket libcurl
	err = http_client.InitWebSocketLibcurl()
	if err != nil {
		log.Fatal("WebSocket libcurl初始化失败:", err)
	}
	defer http_client.CleanupWebSocketLibcurl()
	wsResultMap := make(map[string]*TestResult)
	// 初始化wsResultMap
	for _, rc := range wsrunCases {
		wsResultMap[rc.name] = &TestResult{}
	}

	for _, rc := range wsrunCases {
		wg.Add(1)
		rc := rc
		go func() {
			defer wg.Done()
			// 创建WebSocket客户端实例
			client, err := http_client.NewWebSocketClientLibcurl()
			if err != nil {
				log.Errorf("[%s] 创建客户端失败: %v", rc.name, err)
				return
			}
			defer client.Close()

			// 建立连接
			res := client.Connect(rc.url, 5000)
			if res.Error != "" {
				log.Errorf("[%s] 连接失败: %s", rc.name, res.Error)
				return
			}
			// 连接成功后不再单独打印，由状态显示器统一显示

			avgLatency := int64(0)
			//接收1000次消息
			for {
				// 检查是否已完成1000次
				result := wsResultMap[rc.name]
				if atomic.LoadInt64(&result.successCount) >= 500 {
					break
				}

				recv, ok, err := client.Recv()
				if err != nil {
					log.Errorf("[%s] 接收消息失败: %v", rc.name, err)
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

				//引入服务器时间差修正
				msgTimestampNano += rc.serverTimeDiff
				targetLatency := now - msgTimestampNano

				// //去除明显偏移的极值
				// if avgLatency > 0 && targetLatency > avgLatency*2 {
				// 	continue
				// }

				// 更新统计数据
				atomic.AddInt64(&result.sumLatency, targetLatency)
				atomic.AddInt64(&result.successCount, 1)
				avgLatency = atomic.LoadInt64(&result.sumLatency) / atomic.LoadInt64(&result.successCount)
				atomic.StoreInt64(&result.avgLatency, avgLatency)
			}
		}()
	}

	wg.Wait()

	log.Debug("Binance HTTP和WebSocket延迟测试完成...")
	log.Debug(resultMap)
	log.Debug(wsResultMap)

	result := &BnLatencyResult{
		HttpBinanceSpotLatencyNs:      resultMap[runCases[0].name].avgLatency,
		HttpBinanceFutureLatencyNs:    resultMap[runCases[1].name].avgLatency,
		HttpBinanceDeliveryLatencyNs:  resultMap[runCases[2].name].avgLatency,
		HttpBinancePortfolioLatencyNs: resultMap[runCases[3].name].avgLatency,
		WsBinanceSpotLatencyNs:        wsResultMap[wsrunCases[0].name].avgLatency,
		WsBinanceFutureLatencyNs:      wsResultMap[wsrunCases[1].name].avgLatency,
		WsBinanceDeliveryLatencyNs:    wsResultMap[wsrunCases[2].name].avgLatency,
	}

	log.Debug("==========测试结果========")
	log.Debugf("HTTP      Binance SPOT:      %.6f ms", float64(result.HttpBinanceSpotLatencyNs)/1000000)
	log.Debugf("HTTP      Binance FUTURE:    %.6f ms", float64(result.HttpBinanceFutureLatencyNs)/1000000)
	log.Debugf("HTTP      Binance DELIVERY:  %.6f ms", float64(result.HttpBinanceDeliveryLatencyNs)/1000000)
	log.Debugf("HTTP      Binance PORTFOLIO: %.6f ms", float64(result.HttpBinancePortfolioLatencyNs)/1000000)
	log.Debugf("WebSocket Binance SPOT:      %.6f ms", float64(result.WsBinanceSpotLatencyNs)/1000000)
	log.Debugf("WebSocket Binance FUTURE:    %.6f ms", float64(result.WsBinanceFutureLatencyNs)/1000000)
	log.Debugf("WebSocket Binance DELIVERY:  %.6f ms", float64(result.WsBinanceDeliveryLatencyNs)/1000000)
	log.Debug("=========================")

	return result, nil
}
