package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"go_base_project/config"
	"go_base_project/logger"
	"go_base_project/repository"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Resource 管理全局资源
type Resource struct {
	DB                *gorm.DB
	Redis             *redis.Client
	DeviceID          string
	Proprietarycoding string
	Config            *config.Config
	LiveRepository    *repository.LiveRepository
	ShutdownSignal    chan struct{}
}

// NewResource 创建并初始化资源
func NewResource(config *config.Config) (*Resource, error) {
	r := &Resource{
		Config:         config,
		ShutdownSignal: make(chan struct{}),
	}

	// 初始化数据库
	if config.Common.DBConn != "" {
		db, err := gorm.Open(mysql.Open(config.Common.DBConn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		r.DB = db
	}

	// 初始化 Redis
	if config.Common.RedisAddr != "" {
		r.Redis = redis.NewClient(&redis.Options{
			Addr: config.Common.RedisAddr,
		})
		if err := r.Redis.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to redis: %w", err)
		}
	}

	// 初始化 LiveRepository
	r.LiveRepository = repository.NewLiveRepository()

	// 初始化设备ID
	deviceID, err := r.initDeviceID(config.Common.Device)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize device ID: %w", err)
	}
	r.DeviceID = deviceID
	r.Proprietarycoding = r.GetProprietarycoding()
	return r, nil
}

// 使用 mac 或者 imei 请求云端获取 proprietarycoding
func (r *Resource) GetProprietarycoding() string {
	// 构建请求URL
	url := fmt.Sprintf("%s?mac=%s", r.Config.Common.ProprietarycodingRequestURL, r.DeviceID)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送GET请求
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("Failed to get proprietarycoding", "error", err)
		return ""
	}
	defer resp.Body.Close()

	// 定义响应结构
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Proprietarycoding string `json:"proprietarycoding"`
		} `json:"data"`
	}

	// 解析响应
	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Failed to decode response", "error", err)
		return ""
	}

	// 检查响应状态
	if result.Code != 0 {
		logger.Error("API returned error", "code", result.Code, "message", result.Message)
		return ""
	}
	logger.Info("Successfully got proprietarycoding", "proprietarycoding", result.Data.Proprietarycoding)
	return result.Data.Proprietarycoding
}

// 初始化设备ID
func (r *Resource) initDeviceID(config config.DeviceConfig) (string, error) {
	switch config.Type {
	case "mac":
		// 从指定网卡获取MAC地址
		mac, err := getMacAddress(config.NIC)
		if err != nil {
			return "", fmt.Errorf("failed to get MAC address: %w", err)
		}
		logger.Info("Device ID set from MAC address", "deviceID", mac, "interface", config.NIC)
		return mac, nil
	case "imei":
		// 使用配置中指定的IMEI
		if config.IMEIPath == "" {
			return "", fmt.Errorf("IMEI path is empty in configuration")
		}
		imei, err := os.ReadFile(config.IMEIPath)
		if err != nil {
			return "", fmt.Errorf("failed to read IMEI file: %w", err)
		}
		deviceID := strings.TrimSpace(string(imei))
		logger.Info("Device ID set from IMEI", "deviceID", deviceID)
		return deviceID, nil
	default:
		return "", fmt.Errorf("invalid device type")
	}
}

func getMacAddress(nicName string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Name == nicName {
			mac := iface.HardwareAddr.String()
			if mac == "" {
				return "", fmt.Errorf("empty MAC address for interface %s", nicName)
			}
			return mac, nil
		}
	}
	return "", fmt.Errorf("network interface %s not found", nicName)
}

func (r *Resource) GetDeviceID() string {
	return r.DeviceID
}

// Close 关闭所有资源
func (r *Resource) Close() error {
	if r.Redis != nil {
		if err := r.Redis.Close(); err != nil {
			return fmt.Errorf("failed to close redis: %w", err)
		}
	}

	if r.DB != nil {
		sqlDB, err := r.DB.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB: %w", err)
		}
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	// 发送关闭信号
	close(r.ShutdownSignal)

	return nil
}
