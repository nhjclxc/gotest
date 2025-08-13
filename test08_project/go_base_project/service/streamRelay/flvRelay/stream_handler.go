package flvRelay

import (
	"context"
	"encoding/binary"
	"fmt"
	"go_base_project/flvparser"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	MAX_RETRY_COUNT = 10
	RETRY_INTERVAL  = 10 * time.Second
)

// 对单个 live url 进行转发处理
type StreamHanlder struct {
	sourceURL     string
	stopCh        chan struct{}
	stopped       bool
	flvParser     *flvparser.FLVParser
	headerParsed  bool
	headerBytes   []byte
	headerMutex   sync.RWMutex
	streamRunning bool
	client        *http.Client

	// 重试相关字段
	retryCount int
	retryMutex sync.RWMutex

	// 组合各个管理器
	cache       *StreamCache
	timestamps  *TimestampManager
	connections *ConnectionManager

	// 简化的时间戳管理（保留用于兼容性）
	timestampMutex sync.Mutex
	audioTimestamp uint32 // 音频时间戳计数器
	videoTimestamp uint32 // 视频时间戳计数器
	lastDts        uint32 // 上一个解码时间戳
}

func NewStreamHandler(sourceURL string) *StreamHanlder {
	// 优化的HTTP传输配置
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    false,
		MaxIdleConnsPerHost:   100,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	client := &http.Client{
		Timeout:   0, // 对于流媒体不设置超时，由具体的读写操作控制
		Transport: transport,
	}

	return &StreamHanlder{
		sourceURL:      sourceURL,
		stopCh:         make(chan struct{}),
		stopped:        false,
		flvParser:      flvparser.NewFLVParser(false), // 关闭调试模式，避免过多日志
		client:         client,
		cache:          NewStreamCache(5 * time.Second), // 5秒缓存
		timestamps:     NewTimestampManager(),
		connections:    NewConnectionManager(),
		audioTimestamp: 0,
		videoTimestamp: 0,
		lastDts:        0,
	}
}

func (s *StreamHanlder) Stop() {
	if !s.stopped {
		close(s.stopCh)
		s.stopped = true
	}
}

// Start 开始解析FLV头信息并将其保存，成功后返回，然后启动后续数据转发goroutine
func (s *StreamHanlder) Start() error {
	if s.streamRunning {
		return fmt.Errorf("stream is already running, sourceURL: %s", s.sourceURL)
	}

	// 同步建立连接和解析头部
	resp, err := s.connectAndParseHeader()
	if err != nil {
		return fmt.Errorf("failed to connect and parse header: %v, sourceURL: %s", err, s.sourceURL)
	}

	s.streamRunning = true
	slog.Info("FLV头部解析完成，开始启动数据转发goroutine")

	// 启动后续数据转发的goroutine
	go s.startStreamForwarding(resp)
	return nil
}

// connectAndParseHeader 同步地连接并解析FLV头部
func (s *StreamHanlder) connectAndParseHeader() (*http.Response, error) {
	// 创建HTTP请求
	req, err := http.NewRequest("GET", s.sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加适合流媒体传输的头部
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Encoding", "identity") // 避免压缩，减少CPU使用

	// 执行请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("服务器返回非200状态码: %d", resp.StatusCode)
	}

	// 上下文用于头部解析超时处理
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 解析FLV头部和必要标签
	if err := s.flvParser.ParseInitialTags(ctx, resp.Body); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("解析FLV标签失败: %v", err)
	}

	// 获取必要的FLV头和标签字节
	headerBytes, err := s.flvParser.GetRequiredTagsBytes()
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("获取FLV头部字节失败: %v", err)
	}

	// 保存头部字节供后续使用
	s.headerMutex.Lock()
	s.headerBytes = headerBytes
	s.headerParsed = true
	s.headerMutex.Unlock()

	slog.Info("FLV头部解析完成，等待客户端连接开始转发", "bytes", len(headerBytes))
	return resp, nil
}

// startStreamForwarding 启动流媒体数据转发goroutine
func (s *StreamHanlder) startStreamForwarding(resp *http.Response) {
	defer func() {
		s.streamRunning = false
		// 清理所有管理器
		s.connections.Clear()
		s.timestamps.Clear()
		if resp != nil {
			resp.Body.Close()
		}
	}()

	currentResp := resp
	for {
		// 检查是否收到停止信号
		select {
		case <-s.stopCh:
			return
		default:
		}

		// 开始处理数据流
		err := s.processDataStream(currentResp)
		if err != nil {
			slog.Error("数据流处理失败", "error", err, "sourceURL", s.sourceURL)

			// 关闭当前连接
			if currentResp != nil {
				currentResp.Body.Close()
				currentResp = nil
			}

			// 检查是否需要重试
			s.retryMutex.Lock()
			if s.retryCount >= MAX_RETRY_COUNT {
				s.retryMutex.Unlock()
				slog.Error("达到最大重试次数，停止重连", "retryCount", s.retryCount, "sourceURL", s.sourceURL)
				return
			}
			s.retryCount++
			currentRetry := s.retryCount
			s.retryMutex.Unlock()

			slog.Info("准备重连源流", "retryCount", currentRetry, "maxRetry", MAX_RETRY_COUNT, "sourceURL", s.sourceURL)

			// 等待重试间隔
			select {
			case <-time.After(RETRY_INTERVAL):
			case <-s.stopCh:
				return
			}

			// 尝试重新连接
			newResp, err := s.reconnectToSource()
			if err != nil {
				slog.Error("重连失败", "error", err, "retryCount", currentRetry, "sourceURL", s.sourceURL)
				continue // 继续重试循环
			}

			slog.Info("重连成功", "retryCount", currentRetry, "sourceURL", s.sourceURL)
			currentResp = newResp

			// 重置重试计数器
			s.retryMutex.Lock()
			s.retryCount = 0
			s.retryMutex.Unlock()
		} else {
			// 正常结束，无需重试
			return
		}
	}
}

// processDataStream 处理数据流的核心逻辑
func (s *StreamHanlder) processDataStream(resp *http.Response) error {
	// 持续解析Tag
	for {
		select {
		case <-s.stopCh:
			return nil // 正常停止，不是错误
		default:
			// 解析下一个tag
			tag, err := s.flvParser.ParseNextTag(resp.Body)
			if err != nil {
				if err == io.EOF {
					return fmt.Errorf("流媒体源连接已关闭")
				}
				return fmt.Errorf("解析Tag失败: %v", err)
			}

			// 构建tag字节（无论是否有活跃连接都要构建，因为需要缓存）
			tagBytes := s.buildTagBytes(tag)

			// 将tag添加到缓存
			s.cache.AddTag(tagBytes, tag)

			// 获取活跃连接
			activeChannels := s.connections.GetActiveChannels()
			if len(activeChannels) > 0 {
				// 为每个writer应用时间戳偏移并发送数据
				for writer, ch := range activeChannels {
					adjustedTagBytes := s.timestamps.AdjustTimestamp(writer, tagBytes, tag.Timestamp, tag.TagType)
					s.connections.BroadcastToChannel(ch, adjustedTagBytes)
				}
			}
		}
	}
}

// buildTagBytes 构建完整的tag字节数据
func (s *StreamHanlder) buildTagBytes(tag *flvparser.FLVTag) []byte {
	tagBytes := make([]byte, flvparser.FLVTagHeaderSize+tag.DataSize+flvparser.PrevTagSizeLength)

	// Tag Header
	tagBytes[0] = tag.TagType
	tagBytes[1] = byte(tag.DataSize >> 16)
	tagBytes[2] = byte(tag.DataSize >> 8)
	tagBytes[3] = byte(tag.DataSize)

	timestamp := tag.Timestamp
	tagBytes[4] = byte(timestamp >> 16)
	tagBytes[5] = byte(timestamp >> 8)
	tagBytes[6] = byte(timestamp)
	tagBytes[7] = byte(timestamp >> 24)

	// StreamID (固定为0)
	tagBytes[8] = 0
	tagBytes[9] = 0
	tagBytes[10] = 0

	// Tag Data
	copy(tagBytes[flvparser.FLVTagHeaderSize:], tag.RawData)

	// Previous Tag Size
	prevTagSize := uint32(flvparser.FLVTagHeaderSize) + tag.DataSize
	binary.BigEndian.PutUint32(tagBytes[flvparser.FLVTagHeaderSize+tag.DataSize:], prevTagSize)

	return tagBytes
}

// ForwardStream 将指定的Writer添加到广播列表，并发送FLV头信息和缓存数据
func (s *StreamHanlder) ForwardStream(w io.Writer) error {
	// 确保头部信息已解析
	s.headerMutex.RLock()
	headerParsed := s.headerParsed
	headerBytes := s.headerBytes
	s.headerMutex.RUnlock()

	if !headerParsed {
		return fmt.Errorf("FLV头部尚未解析完成，请先调用Start()")
	}

	// 一次性写入头部数据
	if _, err := w.Write(headerBytes); err != nil {
		return fmt.Errorf("写入FLV头部失败: %v", err)
	}

	// 重置时间戳计数器
	s.timestampMutex.Lock()
	s.audioTimestamp = 0
	s.videoTimestamp = 0
	s.lastDts = 0
	s.timestampMutex.Unlock()

	// 处理头部tag的时间戳（保持原有逻辑）
	s.processHeaderTimestamps(headerBytes)

	slog.Info("FLV头部数据写入完成")

	// 发送缓存的数据（从最近的关键帧开始）
	if cachedDataInfo := s.cache.GetDataFromKeyFrame(); cachedDataInfo != nil {
		slog.Info("发送缓存数据",
			"bytes", len(cachedDataInfo.Data),
			"audioBaseTimestamp", cachedDataInfo.AudioBaseTimestamp,
			"audioLastTimestamp", cachedDataInfo.AudioLastTimestamp,
			"videoBaseTimestamp", cachedDataInfo.VideoBaseTimestamp,
			"videoLastTimestamp", cachedDataInfo.VideoLastTimestamp)

		// 注册时间戳偏移信息
		s.timestamps.RegisterWriter(w, cachedDataInfo)

		if _, err := w.Write(cachedDataInfo.Data); err != nil {
			return fmt.Errorf("写入缓存数据失败: %v", err)
		}
	} else {
		slog.Info("没有可用的缓存数据，发送最新tag数据")
		// 如果没有缓存数据，获取并写入最新的tag数据
		if latestTagsBytes := s.flvParser.GetLatestTagsBytes(); latestTagsBytes != nil {
			if _, err := w.Write(latestTagsBytes); err != nil {
				return fmt.Errorf("写入最新Tag数据失败: %v", err)
			}
		}
	}

	// 确保立即刷新
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 添加到连接管理器
	dataChan := s.connections.AddWriter(w)

	// 启动专属的处理goroutine
	go func() {
		defer s.RemoveWriter(w)

		defer func() {
			if r := recover(); r != nil {
				slog.Error("处理流数据时发生panic", "error", r)
			}
		}()

		for data := range dataChan {
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("写入数据时发生panic", "error", r)
						panic(r)
					}
				}()

				if _, err := w.Write(data); err != nil {
					slog.Error("写入输出流失败", "error", err)
					panic("写入失败")
				}

				if flusher, ok := w.(http.Flusher); ok && flusher != nil {
					flusher.Flush()
				}
			}()
		}
	}()

	return nil
}

// processHeaderTimestamps 处理头部tag的时间戳（保持原有逻辑）
func (s *StreamHanlder) processHeaderTimestamps(headerBytes []byte) {
	// 重新映射头部tag的时间戳
	currentPos := flvparser.FLVHeaderSize + flvparser.PrevTagSizeLength
	for currentPos < len(headerBytes) {
		if currentPos+flvparser.FLVTagHeaderSize > len(headerBytes) {
			break
		}

		// 解析tag头部
		tagType := headerBytes[currentPos]
		dataSize := uint32(headerBytes[currentPos+1])<<16 | uint32(headerBytes[currentPos+2])<<8 | uint32(headerBytes[currentPos+3])

		// 检查是否是视频关键帧和配置帧
		isConfig := false
		if tagType == flvparser.TagTypeVideo && currentPos+flvparser.FLVTagHeaderSize < len(headerBytes) {
			if currentPos+flvparser.FLVTagHeaderSize+1 < len(headerBytes) {
				isConfig = (headerBytes[currentPos+flvparser.FLVTagHeaderSize+1] == 0)
			}
		} else if tagType == flvparser.TagTypeAudio && currentPos+flvparser.FLVTagHeaderSize+1 < len(headerBytes) {
			isConfig = (headerBytes[currentPos+flvparser.FLVTagHeaderSize+1] == 0)
		}

		// 设置时间戳
		var newTimestamp uint32
		if tagType == flvparser.TagTypeScript || (tagType == flvparser.TagTypeVideo && isConfig) || tagType == flvparser.TagTypeAudio && isConfig {
			// script tag和video config tag使用0时间戳
			newTimestamp = 0
		}

		// 更新时间戳
		headerBytes[currentPos+4] = byte(newTimestamp >> 16)
		headerBytes[currentPos+5] = byte(newTimestamp >> 8)
		headerBytes[currentPos+6] = byte(newTimestamp)
		headerBytes[currentPos+7] = byte(newTimestamp >> 24)

		// 移动到下一个tag
		currentPos += flvparser.FLVTagHeaderSize + int(dataSize) + flvparser.PrevTagSizeLength
	}
}

// RemoveWriter 从广播列表中移除指定的Writer
func (s *StreamHanlder) RemoveWriter(w io.Writer) {
	s.connections.RemoveWriter(w)
	s.timestamps.RemoveWriter(w)
}

// GetFLVHeaderInfo 返回已解析的FLV头部信息的可读性描述
func (s *StreamHanlder) GetFLVHeaderInfo() string {
	s.headerMutex.RLock()
	defer s.headerMutex.RUnlock()

	if !s.headerParsed {
		return "FLV头部尚未解析"
	}

	return fmt.Sprintf("FLV头部已解析, 大小: %d 字节\n", len(s.headerBytes))
}

// GetActiveWritersCount 返回当前活跃的输出流数量
func (s *StreamHanlder) GetActiveWritersCount() int {
	return s.connections.GetActiveWritersCount()
}

// GetCacheInfo 返回缓存统计信息
func (s *StreamHanlder) GetCacheInfo() map[string]interface{} {
	return s.cache.GetCacheInfo()
}

// reconnectToSource 重新连接到源流
func (s *StreamHanlder) reconnectToSource() (*http.Response, error) {
	// 创建新的HTTP请求
	req, err := http.NewRequest("GET", s.sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建重连请求失败: %v", err)
	}

	// 添加适合流媒体传输的头部
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Encoding", "identity")

	// 执行请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行重连请求失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("重连时服务器返回非200状态码: %d", resp.StatusCode)
	}

	// 重新解析FLV头部（因为重连后可能需要重新同步）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.flvParser.ParseInitialTags(ctx, resp.Body); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("重连后解析FLV标签失败: %v", err)
	}

	// 重连成功后清理所有缓存状态，确保新流从干净状态开始
	s.cache.Clear()      // 清理流缓存，移除断线前的旧数据
	s.timestamps.Clear() // 清理时间戳管理器，重置时间戳偏移

	slog.Info("重连成功，已清理缓存状态", "sourceURL", s.sourceURL)
	return resp, nil
}
