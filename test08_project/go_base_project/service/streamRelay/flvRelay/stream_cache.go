package flvRelay

import (
	"go_base_project/flvparser"
	"log/slog"
	"sync"
	"time"
)

// CachedTag 缓存的tag结构
type CachedTag struct {
	TagBytes   []byte    // 完整的tag字节数据（包含header + data + prev tag size）
	Timestamp  uint32    // 原始时间戳
	TagType    uint8     // tag类型
	IsKeyFrame bool      // 是否为关键帧
	CacheTime  time.Time // 缓存时间
}

// CachedDataInfo 缓存数据信息
type CachedDataInfo struct {
	Data               []byte
	AudioBaseTimestamp uint32 // 音频开始时间戳
	AudioLastTimestamp uint32 // 音频结束时间戳
	VideoBaseTimestamp uint32 // 视频开始时间戳
	VideoLastTimestamp uint32 // 视频结束时间戳
}

// StreamCache 流缓存管理器
type StreamCache struct {
	cachedTags        []CachedTag   // 缓存的tag列表
	lastKeyFrameIndex int           // 最后一个关键帧在缓存中的索引
	cacheMaxDuration  time.Duration // 缓存最大持续时间
	mutex             sync.RWMutex  // 读写锁
}

// NewStreamCache 创建新的流缓存管理器
func NewStreamCache(maxDuration time.Duration) *StreamCache {
	return &StreamCache{
		cachedTags:        make([]CachedTag, 0, 1000), // 预分配空间
		lastKeyFrameIndex: -1,
		cacheMaxDuration:  maxDuration,
	}
}

// AddTag 将tag添加到缓存中
func (c *StreamCache) AddTag(tagBytes []byte, tag *flvparser.FLVTag) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查是否为关键帧
	isKeyFrame := false
	if tag.TagType == flvparser.TagTypeVideo && len(tag.RawData) > 1 {
		// H.264/H.265: frame type在第一个字节的高4位，1表示关键帧
		frameType := (tag.RawData[0] >> 4) & 0x0F
		isKeyFrame = (frameType == 1)
	}

	// 创建缓存项
	cachedTag := CachedTag{
		TagBytes:   make([]byte, len(tagBytes)),
		Timestamp:  tag.Timestamp,
		TagType:    tag.TagType,
		IsKeyFrame: isKeyFrame,
		CacheTime:  time.Now(),
	}
	copy(cachedTag.TagBytes, tagBytes)

	// 添加到缓存
	c.cachedTags = append(c.cachedTags, cachedTag)

	// 更新最后关键帧索引
	if isKeyFrame {
		c.lastKeyFrameIndex = len(c.cachedTags) - 1
	}

	// 清理过期的缓存
	c.cleanExpired()
}

// cleanExpired 清理过期的缓存数据
func (c *StreamCache) cleanExpired() {
	now := time.Now()
	removeIndex := -1

	// 找到第一个未过期的tag索引
	for i, tag := range c.cachedTags {
		if now.Sub(tag.CacheTime) <= c.cacheMaxDuration {
			removeIndex = i
			break
		}
	}

	// 如果所有tag都过期了
	if removeIndex == -1 {
		c.cachedTags = c.cachedTags[:0]
		c.lastKeyFrameIndex = -1
		return
	}

	// 移除过期的tag，但确保保留至少一个关键帧
	if c.lastKeyFrameIndex >= 0 && removeIndex > c.lastKeyFrameIndex {
		// 如果要移除的范围包含最后的关键帧，则从关键帧开始保留
		removeIndex = c.lastKeyFrameIndex
	}

	if removeIndex > 0 {
		// 移除过期的tag
		copy(c.cachedTags, c.cachedTags[removeIndex:])
		c.cachedTags = c.cachedTags[:len(c.cachedTags)-removeIndex]

		// 更新关键帧索引
		if c.lastKeyFrameIndex >= 0 {
			c.lastKeyFrameIndex -= removeIndex
			if c.lastKeyFrameIndex < 0 {
				c.lastKeyFrameIndex = -1
			}
		}
	}
}

// GetDataFromKeyFrame 获取从最近关键帧开始的缓存数据
func (c *StreamCache) GetDataFromKeyFrame() *CachedDataInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.cachedTags) == 0 || c.lastKeyFrameIndex < 0 {
		return nil
	}

	// 从最后一个关键帧开始获取数据
	startIndex := c.lastKeyFrameIndex
	totalSize := 0
	for i := startIndex; i < len(c.cachedTags); i++ {
		totalSize += len(c.cachedTags[i].TagBytes)
	}

	if totalSize == 0 {
		return nil
	}

	// 组装数据，使用统一的全局基准时间戳确保音视频同步
	result := make([]byte, 0, totalSize)

	// 分别记录音频和视频的时间戳信息，但使用统一的全局基准
	var audioLastTimestamp, videoLastTimestamp uint32 = 0, 0
	var globalBaseTimestamp uint32 = 0xFFFFFFFF
	audioFound, videoFound := false, false

	// 首先确定全局基准时间戳（取音视频最小值）
	for i := startIndex; i < len(c.cachedTags); i++ {
		originalTimestamp := c.cachedTags[i].Timestamp

		switch c.cachedTags[i].TagType {
		case flvparser.TagTypeAudio:
			if !audioFound {
				audioFound = true
			}
		case flvparser.TagTypeVideo:
			if !videoFound {
				videoFound = true
			}
		}

		// 更新全局基准时间戳为最小值
		if originalTimestamp < globalBaseTimestamp {
			globalBaseTimestamp = originalTimestamp
		}

		if audioFound && videoFound {
			break
		}
	}

	// 计算平滑的时间戳递增
	var lastAudioOutput, lastVideoOutput uint32 = 0, 0

	for i := startIndex; i < len(c.cachedTags); i++ {
		tagBytes := make([]byte, len(c.cachedTags[i].TagBytes))
		copy(tagBytes, c.cachedTags[i].TagBytes)

		originalTimestamp := c.cachedTags[i].Timestamp
		var newTimestamp uint32

		// 根据tag类型处理时间戳，使用统一的全局基准确保音视频同步
		switch c.cachedTags[i].TagType {
		case flvparser.TagTypeAudio:
			if audioFound && globalBaseTimestamp != 0xFFFFFFFF {
				// 使用统一的全局基准计算相对偏移
				if originalTimestamp >= globalBaseTimestamp {
					relativeDelta := originalTimestamp - globalBaseTimestamp
					newTimestamp = relativeDelta

					// 检查时间戳连续性，避免跳跃过大
					if lastAudioOutput > 0 {
						if newTimestamp < lastAudioOutput {
							// 时间戳回退，强制递增
							newTimestamp = lastAudioOutput + 23
						} else if newTimestamp-lastAudioOutput > 200 {
							// 跳跃过大，限制增量
							newTimestamp = lastAudioOutput + 23
						}
					}
				} else {
					// 原始时间戳小于全局基准，使用递增
					newTimestamp = lastAudioOutput + 23
				}

				lastAudioOutput = newTimestamp
				audioLastTimestamp = newTimestamp
			} else {
				newTimestamp = 0
			}

		case flvparser.TagTypeVideo:
			if videoFound && globalBaseTimestamp != 0xFFFFFFFF {
				// 使用统一的全局基准计算相对偏移
				if originalTimestamp >= globalBaseTimestamp {
					relativeDelta := originalTimestamp - globalBaseTimestamp
					newTimestamp = relativeDelta

					// 检查时间戳连续性，避免跳跃过大
					if lastVideoOutput > 0 {
						if newTimestamp < lastVideoOutput {
							// 时间戳回退，强制递增
							newTimestamp = lastVideoOutput + 40
						} else if newTimestamp-lastVideoOutput > 200 {
							// 跳跃过大，限制增量
							newTimestamp = lastVideoOutput + 40
						}
					}
				} else {
					// 原始时间戳小于全局基准，使用递增
					newTimestamp = lastVideoOutput + 40
				}

				lastVideoOutput = newTimestamp
				videoLastTimestamp = newTimestamp
			} else {
				newTimestamp = 0
			}

		default:
			// Script tag等，使用0时间戳
			newTimestamp = 0
		}

		// 更新tag header中的时间戳
		if len(tagBytes) >= 11 { // 确保有完整的tag header
			tagBytes[4] = byte(newTimestamp >> 16)
			tagBytes[5] = byte(newTimestamp >> 8)
			tagBytes[6] = byte(newTimestamp)
			tagBytes[7] = byte(newTimestamp >> 24)
		}

		result = append(result, tagBytes...)
	}

	// 计算返回的基准时间戳：如果某种媒体类型不存在，使用全局基准
	resultAudioBaseTimestamp := globalBaseTimestamp
	resultVideoBaseTimestamp := globalBaseTimestamp
	if !audioFound {
		resultAudioBaseTimestamp = globalBaseTimestamp
	}
	if !videoFound {
		resultVideoBaseTimestamp = globalBaseTimestamp
	}

	slog.Info("缓存数据时间戳处理完成",
		"globalBaseTimestamp", globalBaseTimestamp,
		"audioLastTimestamp", audioLastTimestamp,
		"videoLastTimestamp", videoLastTimestamp,
		"audioFound", audioFound,
		"videoFound", videoFound,
		"totalCachedTags", len(c.cachedTags)-startIndex)

	return &CachedDataInfo{
		Data:               result,
		AudioBaseTimestamp: resultAudioBaseTimestamp,
		AudioLastTimestamp: audioLastTimestamp,
		VideoBaseTimestamp: resultVideoBaseTimestamp,
		VideoLastTimestamp: videoLastTimestamp,
	}
}

// GetCacheInfo 返回缓存统计信息
func (c *StreamCache) GetCacheInfo() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	info := map[string]interface{}{
		"total_cached_tags":          len(c.cachedTags),
		"last_keyframe_index":        c.lastKeyFrameIndex,
		"cache_max_duration_seconds": c.cacheMaxDuration.Seconds(),
	}

	if len(c.cachedTags) > 0 {
		oldestTime := c.cachedTags[0].CacheTime
		newestTime := c.cachedTags[len(c.cachedTags)-1].CacheTime
		info["cache_duration_seconds"] = newestTime.Sub(oldestTime).Seconds()
		info["oldest_cache_timestamp"] = c.cachedTags[0].Timestamp
		info["newest_cache_timestamp"] = c.cachedTags[len(c.cachedTags)-1].Timestamp

		// 统计关键帧数量
		keyFrameCount := 0
		for _, tag := range c.cachedTags {
			if tag.IsKeyFrame {
				keyFrameCount++
			}
		}
		info["keyframe_count"] = keyFrameCount
	} else {
		info["cache_duration_seconds"] = 0
		info["keyframe_count"] = 0
	}

	return info
}

// Clear 清空所有缓存
func (c *StreamCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cachedTags = c.cachedTags[:0]
	c.lastKeyFrameIndex = -1
}
