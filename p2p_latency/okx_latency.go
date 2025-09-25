package p2p_latency

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hongssd/cgolatencytest/http_client"
)

type OkxLatencyResult struct {
	HttpOkxLatencyNs int64 //OKX HTTP 纳秒延迟
	WsOkxLatencyNs   int64 //OKX WS 纳秒延迟
}

func TestOkxHttpAndWsLatency() (*OkxLatencyResult, error) {
	log.Debug("开始测试Okx HTTP和WebSocket延迟...")

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
		{"OKX API", "https://www.okx.com/api/v5/public/time", "https://www.okx.com/api/v5/public/time", 0},
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

				for i := 0; i < 5; i++ {
					serverTimeRes := client1.Get(rc.serverTimeUrl, 3000, 0)
					if serverTimeRes.Error != "" {
						log.Errorf("[%s] 获取服务器时间差失败: %s", rc.name, serverTimeRes.Error)
						continue
					}

					if serverTimeRes.StatusCode == 200 {
						serverTimeBodyMap := map[string]interface{}{}
						// log.Info("serverTimeRes.ResponseBody: ", serverTimeRes.ResponseBody)
						err := json.Unmarshal([]byte(serverTimeRes.ResponseBody), &serverTimeBodyMap)
						if err != nil {
							log.Errorf("[%s] 解析服务器时间差失败: [res:%s]%v", rc.name, serverTimeRes.ResponseBody, serverTimeRes.ResponseBody)
							continue
						}

						// log.Info("serverTimeBodyMap: ", serverTimeBodyMap)
						dataInterface, ok := serverTimeBodyMap["data"]
						if !ok {
							continue
						}
						// log.Info("dataInterface: ", dataInterface)
						dataInterfaceList, ok := dataInterface.([]interface{})
						if !ok {
							continue
						}
						// log.Info("dataInterfaceList: ", dataInterfaceList)
						dataInterfaceMap, ok := dataInterfaceList[0].(map[string]interface{})
						if !ok {
							continue
						}

						// log.Info("dataInterfaceMap: ", dataInterfaceMap)
						serverTimeTimestampInterface, ok := dataInterfaceMap["ts"]
						if !ok {
							continue
						}
						// log.Info("serverTimeTimestampInterface: ", serverTimeTimestampInterface)
						//获取服务器毫秒时间戳
						serverTimeTimestamp, err := strconv.ParseInt(serverTimeTimestampInterface.(string), 10, 64)
						if err != nil {
							continue
						}
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
			for i := 0; i < 5; i++ {
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
	log.Info("开始等待HTTP测试完成")
	start := time.Now()
	wg.Wait()

	log.Infof("HTTP测试完成，耗时:%v", time.Since(start))

	// ============================
	// WebSocket 延迟测试部分
	// ============================
	wsrunCases := []struct {
		name           string
		url            string
		subscribeMsg   string
		serverTimeDiff int64
	}{
		{"OKX WS STREAM", "wss://ws.okx.com/ws/v5/public",
			`
		{
			"id": "1512",
			"op": "subscribe",
			"args": [{
				"channel": "bbo-tbt",
				"instId": "BTC-USDT-SWAP"
			},{
				"channel": "bbo-tbt",
				"instId": "ETH-USDT-SWAP"
			},{
				"channel": "bbo-tbt",
				"instId": "BTC-USDT"
			},{
				"channel": "bbo-tbt",
				"instId": "ETH-USDT"
			}]
		} `,
			0},
		// {"BN FUTURE    WS STREAM", "wss://fstream.binance.com/stream?streams=btcusdt@depth@0ms/ethusdt@depth@0ms", 0},
		// {"BN DELIVERY  WS STREAM", "wss://dstream.binance.com/stream?streams=btcusd_perp@depth@0ms", 0},
		// // {"BN FUTURE    WS STREAM", "wss://fstream-mm.binance.com/stream", 0},
		// {"BN DELIVERY  WS STREAM", "wss://dstream-mm.binance.com/stream", 0},
	}

	wsrunCases[0].serverTimeDiff = runCases[0].serverTimeDiff

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

			//链接成功后发送一条订阅消息
			code, err := client.Send(rc.subscribeMsg, true)
			if err != nil {
				log.Errorf("[%s] 发送订阅消息失败: %v", rc.name, err)
				return
			}
			log.Infof("[%s] 发送订阅消息成功: %d", rc.name, code)

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
				// log.Info("ws recv: ", recv)
				now := time.Now().UnixNano()

				type WsRecv struct {
					Arg struct {
						Channel string `json:"channel"`
						InstId  string `json:"instId"`
					} `json:"arg"`
					Data []struct {
						Asks  [][]string `json:"asks"`
						Bids  [][]string `json:"bids"`
						Ts    string     `json:"ts"`
						SeqId int64      `json:"seqId"`
					} `json:"data"`
				}

				wsRecv := WsRecv{}
				err = json.Unmarshal([]byte(recv), &wsRecv)
				if err != nil {
					continue // 跳过无效消息
				}
				if len(wsRecv.Data) == 0 {
					continue
				}

				msgTimestamp, err := strconv.ParseInt(wsRecv.Data[0].Ts, 10, 64)
				if err != nil {
					continue
				}
				// log.Info("msgTimestamp: ", msgTimestamp)
				//毫秒转纳秒
				msgTimestampNano := msgTimestamp * 1000000

				//引入服务器时间差修正
				targetLatency := now - msgTimestampNano + rc.serverTimeDiff

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

	log.Info("开始等待WS测试完成")
	start = time.Now()
	wg.Wait()

	log.Infof("WS测试完成，耗时:%v", time.Since(start))

	log.Debug("OKX HTTP和WebSocket延迟测试完成...")
	log.Debug(resultMap)
	log.Debug(wsResultMap)

	result := &OkxLatencyResult{
		HttpOkxLatencyNs: resultMap[runCases[0].name].avgLatency,
		WsOkxLatencyNs:   wsResultMap[wsrunCases[0].name].avgLatency,
	}

	log.Debug("==========测试结果========")
	log.Debugf("HTTP      OKX:      %.6f ms", float64(result.HttpOkxLatencyNs)/1000000)
	log.Debugf("WebSocket OKX:      %.6f ms", float64(result.WsOkxLatencyNs)/1000000)
	log.Debug("=========================")

	return result, nil
}
