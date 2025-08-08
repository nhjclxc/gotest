package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// 配置结构体
type Config struct {
	Name    string `mapstructure:"name" json:"name"`
	Port    int    `mapstructure:"port" json:"port"`
	Env     string `mapstructure:"env" json:"env"`
	Version string `mapstructure:"version" json:"version"`
}

// 全局配置变量
var GlobalConfig *Config

// InitConfig 读取配置文件
func InitConfig(configPath string) *Config {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}

	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		panic(fmt.Errorf("解析配置文件失败: %w", err))
	}

	GlobalConfig = &conf

	fmt.Println("✅ 配置加载成功:", configPath)
	return &conf
}
