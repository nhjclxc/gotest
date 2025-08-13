package crontasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go_base_project/logger"
	"go_base_project/resource"
	"net"
	"net/http"
	"time"
)

// CloudAPITask 从云端API获取数据的任务
type ReportBoxNetIPTask struct {
	client     *http.Client
	interval   time.Duration
	requestURL string
	resource   *resource.Resource
}

// NewReportBoxNetIPTask 创建一个新的云端API请求任务
func NewReportBoxNetIPTask(resource *resource.Resource) *ReportBoxNetIPTask {
	cfg := resource.Config.Cronjob.ReportBoxNetIP
	return &ReportBoxNetIPTask{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		requestURL: cfg.RequestURL,
		interval:   time.Duration(cfg.Interval) * time.Second,
		resource:   resource,
	}
}

func (t *ReportBoxNetIPTask) GetBoxIP(ethName string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Error("Failed to get network interfaces", "error", err)
		return ""
	}

	for _, iface := range interfaces {
		if iface.Name == ethName {
			addrs, err := iface.Addrs()
			if err != nil {
				logger.Error("Failed to get interface addresses", "interface", ethName, "error", err)
				return ""
			}

			for _, addr := range addrs {
				// 检查是否为IP网络地址
				if ipnet, ok := addr.(*net.IPNet); ok {
					// 只返回IPv4地址且不是回环地址
					if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
						return ip4.String()
					}
				}
			}
		}
	}

	logger.Error("No valid IP address found", "interface", ethName)
	return ""
}

// Name 返回任务名称
func (t *ReportBoxNetIPTask) Name() string {
	return "report_box_netip_task"
}

func (t *ReportBoxNetIPTask) Enable() bool {
	return t.resource.Config.Cronjob.ReportBoxNetIP.Enable
}

// Execute 执行任务
func (t *ReportBoxNetIPTask) Execute(ctx context.Context) error {

	if !t.resource.Config.Cronjob.ReportBoxNetIP.Enable {
		logger.Info("ReportBoxNetIPTask is disabled")
		return nil
	}

	boxIP := t.GetBoxIP(t.resource.Config.Common.Eth)
	if boxIP == "" {
		return fmt.Errorf("failed to get box IP")
	}

	// 构建请求URL
	url := t.requestURL

	// 构建请求体
	requestBody := struct {
		IP                string `json:"ip"`
		Proprietarycoding string `json:"proprietarycoding"`
	}{
		IP:                boxIP,
		Proprietarycoding: t.resource.Proprietarycoding,
	}
	logger.Info("Box IP", "ip", boxIP, "proprietarycoding", t.resource.Proprietarycoding)

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("Failed to marshal request body", "error", err)
		return err
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to send request", "error", err)
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("Request failed", "status", resp.StatusCode)
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}
	logger.Info("Successfully reported box IP", "ip", boxIP)
	return nil
}

// GetInterval 返回任务执行间隔
func (t *ReportBoxNetIPTask) GetInterval() time.Duration {
	return t.interval
}
