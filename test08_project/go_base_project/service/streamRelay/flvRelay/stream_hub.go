package flvRelay

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// 对多个 live url 进行转发处理
// 底层使用 StreamHanlder 进行转发处理
type StreamHub struct {
	streams map[int]*StreamHanlder
	mutex   sync.RWMutex
}

func NewStreamHub() *StreamHub {

	res := &StreamHub{
		streams: make(map[int]*StreamHanlder),
	}
	return res
}

func (h *StreamHub) AddStream(liveID int, sourceURL string) (distURLPath string, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, exists := h.streams[liveID]; exists {
		slog.Info("Stream already exists, skipping creation", "liveID", liveID, "sourceURL", sourceURL)
		return "", fmt.Errorf("stream with ID %d already exists", liveID)
	}

	slog.Info("Creating new stream", "liveID", liveID, "sourceURL", sourceURL)
	handler := NewStreamHandler(sourceURL)

	// 同步启动流处理器，如果失败则返回错误
	if err := handler.Start(); err != nil {
		slog.Error("Failed to start stream handler", "liveID", liveID, "error", err)
		return "", fmt.Errorf("failed to start stream %d: %v", liveID, err)
	}

	// 只有成功启动后才添加到streams map中
	h.streams[liveID] = handler
	slog.Info("Stream created and started successfully", "liveID", liveID, "totalStreams", len(h.streams))
	path := fmt.Sprintf("/live/%d.flv", liveID)
	return path, nil
}

func (h *StreamHub) RemoveStream(liveID int) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	stream, exists := h.streams[liveID]
	if !exists {
		return fmt.Errorf("stream with ID %d not found", liveID)
	}

	// 先停止流
	stream.Stop()

	// 从map中删除
	delete(h.streams, liveID)
	slog.Info("Stream removed", "liveID", liveID)
	return nil
}

func (h *StreamHub) GetStream(liveID int) (*StreamHanlder, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stream, exists := h.streams[liveID]
	if !exists {
		return nil, fmt.Errorf("stream with ID %d not found", liveID)
	}

	return stream, nil
}

func (h *StreamHub) ListStreams() []int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	streams := make([]int, 0, len(h.streams))
	for id := range h.streams {
		streams = append(streams, (id))
	}
	return streams
}

func (h *StreamHub) StreamExists(liveID int) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, exists := h.streams[liveID]
	return exists
}

func (h *StreamHub) GetStreamCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.streams)
}

func (h *StreamHub) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Stop all streams
	for _, stream := range h.streams {
		stream.Stop()
	}

	// Clear the streams map
	h.streams = make(map[int]*StreamHanlder)
}

func (h *StreamHub) Forward(liveID int, w io.Writer) error {
	stream, err := h.GetStream(liveID)
	if err != nil {
		return err
	}

	return stream.ForwardStream(w)
}

// GetCacheInfo 获取指定流的缓存信息
func (h *StreamHub) GetCacheInfo(liveID int) map[string]interface{} {
	stream, err := h.GetStream(liveID)
	if err != nil {
		return nil
	}

	return stream.GetCacheInfo()
}

// GetHandler 获取指定流的处理器，用于检查流是否存在
func (h *StreamHub) GetHandler(liveID int) (*StreamHanlder, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stream, exists := h.streams[liveID]
	return stream, exists
}

// GetStreamStatus 获取流状态信息
func (h *StreamHub) GetStreamStatus(liveID int) map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stream, exists := h.streams[liveID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
			"error":  "stream not found",
		}
	}

	return map[string]interface{}{
		"exists":        true,
		"streamRunning": stream.streamRunning,
		"headerParsed":  stream.headerParsed,
		"activeWriters": stream.GetActiveWritersCount(),
		"sourceURL":     stream.sourceURL,
	}
}

// GetAllStreamsStatus 获取所有流的状态
func (h *StreamHub) GetAllStreamsStatus() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make(map[string]interface{})
	result["totalStreams"] = len(h.streams)

	streams := make([]interface{}, 0, len(h.streams))
	for liveID, stream := range h.streams {
		streamInfo := map[string]interface{}{
			"id":             liveID,
			"stream_running": stream.streamRunning,
			"header_parsed":  stream.headerParsed,
			"active_writers": stream.GetActiveWritersCount(),
			"source_url":     stream.sourceURL,
		}
		streams = append(streams, streamInfo)
	}
	result["streams"] = streams

	return result
}
