package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CommonConfig 包含共享的配置项
type CommonConfig struct {
	Debug                       bool      `yaml:"debug"`
	DBConn                      string    `yaml:"db"`
	RedisAddr                   string    `yaml:"redis"`
	Log                         LogConfig `yaml:"log"`
	Eth                         string    `yaml:"eth"`
	ProprietarycodingRequestURL string    `yaml:"proprietarycoding_request_url"`
}

// HTTPConfig HTTP服务特定配置
type HTTPConfig struct {
	Port      string `yaml:"port"`
	ProxyHost string `yaml:"proxy_host"` // 代理目标主机地址
}

type LogConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	FilePath   string `yaml:"file_path"`   // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大尺寸（MB）
	MaxBackups int    `yaml:"max_backups"` // 保留的旧日志文件数量
}

// Config 总配置结构
type Config struct {
	Common CommonConfig `yaml:"common"`
	HTTP   HTTPConfig   `yaml:"http"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
