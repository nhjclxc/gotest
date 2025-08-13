package flvRelay

import (
	"go_base_project/flvparser"
	"io"
	"log/slog"
	"sync"
)

// TimestampOffset 为每个客户端连接维护时间戳偏移信息
type TimestampOffset struct {
	audioBaseTimestamp      uint32 // 音频基准时间戳
	audioLastCacheTimestamp uint32 // 音频缓存数据的最后时间戳
	audioOffsetApplied      bool   // 音频是否已应用时间戳偏移

	videoBaseTimestamp      uint32 // 视频基准时间戳
	videoLastCacheTimestamp uint32 // 视频缓存数据的最后时间戳
	videoOffsetApplied      bool   // 视频是否已应用时间戳偏移

	globalBaseTimestamp uint32 // 全局基准时间戳（取音视频最小值）
	lastAudioTimestamp  uint32 // 上一个音频帧的输出时间戳
	lastVideoTimestamp  uint32 // 上一个视频帧的输出时间戳

	mutex sync.Mutex
}

// TimestampManager 时间戳管理器
type TimestampManager struct {
	writerOffsets map[io.Writer]*TimestampOffset
	mutex         sync.RWMutex
}

// NewTimestampManager 创建新的时间戳管理器
func NewTimestampManager() *TimestampManager {
	return &TimestampManager{
		writerOffsets: make(map[io.Writer]*TimestampOffset),
	}
}

// RegisterWriter 为writer注册时间戳偏移信息
func (tm *TimestampManager) RegisterWriter(writer io.Writer, cacheInfo *CachedDataInfo) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 计算全局基准时间戳（取音视频最小值）
	globalBase := cacheInfo.AudioBaseTimestamp
	if cacheInfo.VideoBaseTimestamp < globalBase {
		globalBase = cacheInfo.VideoBaseTimestamp
	}

	timestampOffset := &TimestampOffset{
		audioBaseTimestamp:      cacheInfo.AudioBaseTimestamp,
		audioLastCacheTimestamp: cacheInfo.AudioLastTimestamp,
		audioOffsetApplied:      false,
		videoBaseTimestamp:      cacheInfo.VideoBaseTimestamp,
		videoLastCacheTimestamp: cacheInfo.VideoLastTimestamp,
		videoOffsetApplied:      false,
		globalBaseTimestamp:     globalBase,
		lastAudioTimestamp:      cacheInfo.AudioLastTimestamp,
		lastVideoTimestamp:      cacheInfo.VideoLastTimestamp,
	}

	tm.writerOffsets[writer] = timestampOffset
}

// RemoveWriter 移除writer的时间戳偏移信息
func (tm *TimestampManager) RemoveWriter(writer io.Writer) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	delete(tm.writerOffsets, writer)
}

// AdjustTimestamp 为特定writer调整tag的时间戳
func (tm *TimestampManager) AdjustTimestamp(writer io.Writer, tagBytes []byte, originalTimestamp uint32, tagType uint8) []byte {
	tm.mutex.RLock()
	timestampOffset := tm.writerOffsets[writer]
	tm.mutex.RUnlock()

	// 如果没有时间戳偏移信息，直接返回原始数据
	if timestampOffset == nil {
		return tagBytes
	}

	timestampOffset.mutex.Lock()
	defer timestampOffset.mutex.Unlock()

	// 复制原始数据
	adjustedTagBytes := make([]byte, len(tagBytes))
	copy(adjustedTagBytes, tagBytes)

	// 使用统一的全局基准时间戳来计算调整后的时间戳
	var adjustedTimestamp uint32

	switch tagType {
	case flvparser.TagTypeAudio:
		// 音频时间戳处理 - 使用全局基准确保与视频同步
		if originalTimestamp >= timestampOffset.globalBaseTimestamp {
			// 正常情况：使用统一基准计算相对偏移
			relativeDelta := originalTimestamp - timestampOffset.globalBaseTimestamp
			// 使用音频缓存的最后时间戳作为起点，但确保与视频时间戳保持相对关系
			adjustedTimestamp = relativeDelta
		} else {
			// 时间戳回退：基于上一帧时间戳递增
			if timestampOffset.lastAudioTimestamp > 0 {
				adjustedTimestamp = timestampOffset.lastAudioTimestamp + 23 // ~23ms AAC frame duration
			} else {
				adjustedTimestamp = 0
			}
			slog.Debug("音频时间戳回退",
				"originalTimestamp", originalTimestamp,
				"globalBaseTimestamp", timestampOffset.globalBaseTimestamp,
				"adjustedTimestamp", adjustedTimestamp)
		}

		// 检查时间戳是否向后跳跃（允许适度向前跳跃）
		if timestampOffset.lastAudioTimestamp > 0 && adjustedTimestamp < timestampOffset.lastAudioTimestamp {
			// 时间戳向后跳跃，使用递增方式
			adjustedTimestamp = timestampOffset.lastAudioTimestamp + 23
			slog.Debug("音频时间戳向后跳跃，强制递增",
				"originalTimestamp", originalTimestamp,
				"lastAudioTimestamp", timestampOffset.lastAudioTimestamp,
				"adjustedTimestamp", adjustedTimestamp)
		} else if timestampOffset.lastAudioTimestamp > 0 {
			// 检查向前跳跃是否过大（超过500ms认为异常）
			timeDiff := adjustedTimestamp - timestampOffset.lastAudioTimestamp
			if timeDiff > 500 {
				// 限制跳跃幅度，避免播放器告警
				oldAdjusted := adjustedTimestamp
				adjustedTimestamp = timestampOffset.lastAudioTimestamp + 23
				slog.Warn("检测到音频时间戳大跳跃，已限制",
					"originalTimestamp", originalTimestamp,
					"oldAdjustedTimestamp", oldAdjusted,
					"newAdjustedTimestamp", adjustedTimestamp,
					"timeDiff", timeDiff)
			}
		}

		timestampOffset.lastAudioTimestamp = adjustedTimestamp

	case flvparser.TagTypeVideo:
		// 视频时间戳处理 - 使用全局基准确保与音频同步
		if originalTimestamp >= timestampOffset.globalBaseTimestamp {
			// 正常情况：使用统一基准计算相对偏移
			relativeDelta := originalTimestamp - timestampOffset.globalBaseTimestamp
			adjustedTimestamp = relativeDelta
		} else {
			// 时间戳回退：基于上一帧时间戳递增
			if timestampOffset.lastVideoTimestamp > 0 {
				adjustedTimestamp = timestampOffset.lastVideoTimestamp + 40 // ~40ms for 25fps
			} else {
				adjustedTimestamp = 0
			}
			slog.Debug("视频时间戳回退",
				"originalTimestamp", originalTimestamp,
				"globalBaseTimestamp", timestampOffset.globalBaseTimestamp,
				"adjustedTimestamp", adjustedTimestamp)
		}

		// 检查时间戳是否向后跳跃
		if timestampOffset.lastVideoTimestamp > 0 && adjustedTimestamp < timestampOffset.lastVideoTimestamp {
			// 时间戳向后跳跃，使用递增方式
			adjustedTimestamp = timestampOffset.lastVideoTimestamp + 40
			slog.Debug("视频时间戳向后跳跃，强制递增",
				"originalTimestamp", originalTimestamp,
				"lastVideoTimestamp", timestampOffset.lastVideoTimestamp,
				"adjustedTimestamp", adjustedTimestamp)
		} else if timestampOffset.lastVideoTimestamp > 0 {
			// 检查向前跳跃是否过大（超过500ms认为异常）
			timeDiff := adjustedTimestamp - timestampOffset.lastVideoTimestamp
			if timeDiff > 500 {
				oldAdjusted := adjustedTimestamp
				adjustedTimestamp = timestampOffset.lastVideoTimestamp + 40
				slog.Warn("检测到视频时间戳大跳跃，已限制",
					"originalTimestamp", originalTimestamp,
					"oldAdjustedTimestamp", oldAdjusted,
					"newAdjustedTimestamp", adjustedTimestamp,
					"timeDiff", timeDiff)
			}
		}

		timestampOffset.lastVideoTimestamp = adjustedTimestamp

	default:
		// Script tag等，保持原始时间戳或使用0
		adjustedTimestamp = 0 // 脚本标签使用0时间戳
	}

	// 更新tag header中的时间戳
	if len(adjustedTagBytes) >= 11 { // 确保有完整的tag header
		adjustedTagBytes[4] = byte(adjustedTimestamp >> 16)
		adjustedTagBytes[5] = byte(adjustedTimestamp >> 8)
		adjustedTagBytes[6] = byte(adjustedTimestamp)
		adjustedTagBytes[7] = byte(adjustedTimestamp >> 24)
	}

	return adjustedTagBytes
}

// Clear 清空所有时间戳偏移信息
func (tm *TimestampManager) Clear() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.writerOffsets = make(map[io.Writer]*TimestampOffset)
}
