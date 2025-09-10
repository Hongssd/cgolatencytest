package config

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/gobuffalo/packr/v2"
	"github.com/spf13/viper"
)

var configFile string
var network string
var version string
var dynamic string
var arbitrage string

var p2pNodeName string

func init() {
	flag.StringVar(&configFile, "config", "", "configFile")
	flag.StringVar(&network, "network", "", "network")
	flag.StringVar(&version, "version", "", "version")
	flag.StringVar(&dynamic, "dynamic", "", "dynamic")
	flag.StringVar(&arbitrage, "arbitrage", "", "arbitrage")

	flag.StringVar(&p2pNodeName, "p2p-node-name", "", "p2pNodeName")

	flag.Parse()
	SetDeafultConfig()
	ReadConfig()
}

func GetNetwork() string {
	return network
}

func GetVersion() string {
	return version
}

func GetDynamic() string {
	return dynamic
}

func GetArbitrage() string {
	return arbitrage
}

func GetP2pNodeName() string {
	return p2pNodeName
}

func GetConfig(name string) string {
	result := viper.GetString(name)
	// mylog.Debugf("Successfully read configuration %s is %s\n", name, result)
	return result
}

func GetConfigInt(name string) int {
	return viper.GetInt(name)
}

func GetConfigBool(name string) bool {
	return viper.GetBool(name)
}

func GetConfigSlice(name string) []string {
	result := viper.GetStringSlice(name)
	return result
}

func GetConfigStringMap(name string) map[string]interface{} {
	result := viper.GetStringMap(name)
	return result
}

func GetConfigStringMapString(name string) map[string]string {
	result := viper.GetStringMapString(name)
	return result
}

func SetDeafultConfig() {
	fmt.Println("Initialize configuration default settings")
}

func ReadConfig() bool {
	box := packr.New("config", "./")
	configType := "yml"
	defaultConfig, err := box.Find("default.yml")
	if err != nil {
		panic(fmt.Sprintf("Fatal error config: %s", err.Error()))
	}
	v := viper.New()
	v.SetConfigType(configType)
	err = v.ReadConfig(bytes.NewReader(defaultConfig))
	if err != nil {
		panic(fmt.Sprintf("Fatal error config: %s", err.Error()))
	}

	configs := v.AllSettings()
	// 将default中的配置全部以默认配置写入
	for k, v := range configs {
		viper.SetDefault(k, v)
	}
	env := configFile
	// 根据配置的env读取相应的配置信息
	if env != "" {
		envConfig, err := box.Find(env)
		if err != nil {
			panic(fmt.Sprintf("Fatal error config: %s", err.Error()))
		}
		viper.SetConfigType(configType)
		err = viper.ReadConfig(bytes.NewReader(envConfig))
		fmt.Println("配置文件加载:", env)
		if err != nil {
			panic(fmt.Sprintf("Fatal error config: %s", err.Error()))
		}
	}

	return true
}
