package flvRelay

import (
	"io"
	"log/slog"
	"sync"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	writerChannels map[io.Writer]chan []byte
	mutex          sync.RWMutex
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		writerChannels: make(map[io.Writer]chan []byte),
	}
}

// AddWriter 添加writer并返回其专属通道
func (cm *ConnectionManager) AddWriter(writer io.Writer) chan []byte {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 为writer创建专属的数据通道
	dataChan := make(chan []byte, 1024*1024) // 1MB缓冲区
	cm.writerChannels[writer] = dataChan

	return dataChan
}

// RemoveWriter 移除writer并关闭其通道
func (cm *ConnectionManager) RemoveWriter(writer io.Writer) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 如果存在该writer的通道，关闭并删除
	if ch, exists := cm.writerChannels[writer]; exists {
		close(ch)                         // 关闭通道，通知处理goroutine退出
		delete(cm.writerChannels, writer) // 从映射表删除
	}
}

// GetActiveChannels 获取当前活跃的writer通道映射
func (cm *ConnectionManager) GetActiveChannels() map[io.Writer]chan []byte {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// 创建副本避免并发访问问题
	activeChannels := make(map[io.Writer]chan []byte, len(cm.writerChannels))
	for writer, ch := range cm.writerChannels {
		activeChannels[writer] = ch
	}

	return activeChannels
}

// GetActiveWritersCount 返回当前活跃的输出流数量
func (cm *ConnectionManager) GetActiveWritersCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.writerChannels)
}

// BroadcastToChannel 向指定通道发送数据（非阻塞）
func (cm *ConnectionManager) BroadcastToChannel(ch chan []byte, data []byte) {
	select {
	case ch <- data:
		// 成功发送
	default:
		// 通道已满，丢弃此次发送
		slog.Warn("写入通道已满，丢弃数据包")
	}
}

// BroadcastToAll 向所有活跃连接广播数据
func (cm *ConnectionManager) BroadcastToAll(data []byte) {
	activeChannels := cm.GetActiveChannels()

	for _, ch := range activeChannels {
		cm.BroadcastToChannel(ch, data)
	}
}

// Clear 清空所有连接
func (cm *ConnectionManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 先复制所有通道引用，然后清空映射，最后关闭通道
	channelsToClose := make([]chan []byte, 0, len(cm.writerChannels))
	for _, ch := range cm.writerChannels {
		channelsToClose = append(channelsToClose, ch)
	}

	// 清空映射
	cm.writerChannels = make(map[io.Writer]chan []byte)

	// 关闭所有通道（在锁外执行，避免死锁）
	go func() {
		for _, ch := range channelsToClose {
			close(ch)
		}
	}()
}
