package http_client

import (
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type VegetaClient struct {
	attack *vegeta.Attacker
}

func NewVegetaClient() *VegetaClient {
	attack := vegeta.NewAttacker()
	return &VegetaClient{attack: attack}
}

func (c *VegetaClient) Request(url string, method string, timeoutMs int) vegeta.Metrics {
	rate := vegeta.Rate{Freq: 1, Per: time.Second}
	duration := time.Duration(timeoutMs) * time.Millisecond
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: method,
		URL:    url,
	})

	var metrics vegeta.Metrics
	for res := range c.attack.Attack(targeter, rate, duration, "VegetaClient") {
		metrics.Add(res)
	}
	metrics.Close()

	return metrics
}

func (c *VegetaClient) Close() {
	c.attack.Stop()
}
