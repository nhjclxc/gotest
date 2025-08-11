package main

//
//import (
//	"bytes"
//	"encoding/binary"
//	"errors"
//	"github.com/yutopp/go-flv/tag"
//	"log"
//	"net/http"
//	"sync"
//
//	"github.com/gin-gonic/gin"
//	flv "github.com/yutopp/go-flv"
//)
//
//type GOPCache struct {
//	mu        sync.RWMutex
//	data      [][]byte
//	sps, pps  []byte
//	startTime uint32
//}
//
//func (c *GOPCache) AddTag11(tag0 *tag.FlvTag) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	data := tag0.Data
//
//	// Video Tag
//	if tag0.TagType == tag.TagTypeVideo {
//		frameType := (data[0] >> 4) & 0x0F
//		codecID := tag.Data[0] & 0x0F
//
//		if codecID == 7 { // AVC
//			avcPacketType := tag.Data[1]
//			if avcPacketType == 0 {
//				// AVCDecoderConfigurationRecord => 提取 SPS/PPS
//				sps, pps := extractSPSPPS(tag.Data[5:])
//				if len(sps) > 0 {
//					c.sps = sps
//				}
//				if len(pps) > 0 {
//					c.pps = pps
//				}
//			}
//
//			if frameType == 1 && avcPacketType == 1 && isIDRFrame(tag.Data[5:]) {
//				// 新关键帧 => 重置缓存
//				c.data = nil
//				// 插入最新 SPS/PPS
//				if c.sps != nil && c.pps != nil {
//					c.data = append(c.data, makeVideoConfigTag(c.sps, c.pps))
//				}
//			}
//		}
//	}
//
//	// 添加当前 tag
//	buf := bytes.NewBuffer(nil)
//	if err := flv.WriteTag(buf, tag); err == nil {
//		c.data = append(c.data, buf.Bytes())
//	}
//}
//
//
//// AddTag 已修正版本
//func (c *GOPCache) AddTag(tag0 *tag.FlvTag) {
//	if tag0 == nil {
//		return
//	}
//
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	switch data := tag0.Data.(type) {
//	case *tag.AudioData:
//
//		// 只处理 video tag 的关键帧逻辑
//		if tag0.TagType == 9 && data.SoundSize > 0 { // video
//			// 第一字节：FrameType(高4位) | CodecID(低4位)
//			frameType := data.AACPacketType
//			codecID := data.SoundType
//
//			if codecID == 7 { // AVC (H.264)
//				// 需要至少 2 字节来读 avcPacketType，且通常 avc 数据从 offset 5 开始包含 NALUs
//				if data.SoundSize >= 2 {
//					avcPacketType := data.SoundType tag0.Data[1]
//					if avcPacketType == 0 {
//						// AVCDecoderConfigurationRecord（SPS/PPS）通常在 data[5:]
//						if len(tag0.Data) > 5 {
//							sps, pps := extractSPSPPS(tag0.Data[5:])
//							if len(sps) > 0 {
//								c.sps = sps
//							}
//							if len(pps) > 0 {
//								c.pps = pps
//							}
//						}
//					}
//					// 视频帧：avcPacketType==1 表示 NALUs（编码帧）
//					if frameType == 1 && avcPacketType == 1 {
//						// NALUs 从 data[5:] 开始（FLV 视频包的 AVC 格式）
//						if len(tag0.Data) > 5 && isIDRFrame(tag0.Data[5:]) {
//							// 遇到关键帧(IDR) -> 重置缓存并插入 SPS/PPS 配置 tag（如果有）
//							c.data = nil
//							if len(c.sps) > 0 && len(c.pps) > 0 {
//								if cfg, err := makeVideoConfigTag(c.sps, c.pps); err == nil {
//									c.data = append(c.data, cfg)
//								}
//							}
//						}
//					}
//				}
//			} else {
//				// 非 AVC codec：若 frameType==1 则也视为关键帧（退化方案）
//				if frameType == 1 {
//					c.data = nil
//					if len(c.sps) > 0 && len(c.pps) > 0 {
//						if cfg, err := makeVideoConfigTag(c.sps, c.pps); err == nil {
//							c.data = append(c.data, cfg)
//						}
//					}
//				}
//			}
//		}
//
//
//	}
//
//	// 把当前 tag 序列化为 FLV bytes 并追加到缓存
//	if raw, err := encodeFLVTag(tag0); err == nil {
//		c.data = append(c.data, raw)
//	}
//}
//
//// 把 FlvTag 序列化为完整 FLV tag bytes（header11 + data + prevTagSize(4)）
//func encodeFLVTag(t *tag.FlvTag) ([]byte, error) {
//	if t == nil {
//		return nil, errors.New("nil tag")
//	}
//
//
//	switch data := t.Data.(type) {
//	case *tag.AudioData:
//
//		var originData []byte = make([]byte, 0)
//
//		data.Data.Read(originData)
//
//		dataSize := data.SoundRate
//		buf := bytes.NewBuffer(nil)
//		// Tag header: 1(TagType)+3(DataSize)+3(TimestampLower)+1(TimestampExt)+3(StreamID)
//		buf.WriteByte(byte(t.TagType))
//		buf.WriteByte(originData[1])
//		buf.WriteByte(originData[2])
//		buf.WriteByte(originData[3])
//
//		// timestamp lower 24 bits
//		ts := t.Timestamp & 0xFFFFFF
//		buf.WriteByte(byte((ts >> 16) & 0xff))
//		buf.WriteByte(byte((ts >> 8) & 0xff))
//		buf.WriteByte(byte(ts & 0xff))
//		// timestamp extended
//		buf.WriteByte(byte((t.Timestamp >> 24) & 0xff))
//
//		// streamID (3 bytes) always 0
//		buf.Write([]byte{0x00, 0x00, 0x00})
//
//		// payload
//		if dataSize > 0 {
//			buf.Write()
//			data.Data.Read(buf)
//		}
//		// previousTagSize: header(11) + dataSize
//		prev := uint32(11 + dataSize)
//		b := make([]byte, 4)
//		binary.BigEndian.PutUint32(b, prev)
//		buf.Write(b)
//
//		return buf.Bytes(), nil
//
//
//	}
//	return nil, errors.New("不支持的类型")
//}
//
//func (c *GOPCache) Dump() [][]byte {
//	c.mu.RLock()
//	defer c.mu.RUnlock()
//	return append([][]byte(nil), c.data...)
//}
//
//// ------------------ 工具函数 ------------------
//
//// 提取 SPS/PPS
//func extractSPSPPS(data []byte) (sps, pps []byte) {
//	// 按 AVCDecoderConfigurationRecord 结构解析
//	if len(data) < 7 {
//		return
//	}
//	spsCount := int(data[5] & 0x1F)
//	pos := 6
//	for i := 0; i < spsCount; i++ {
//		if pos+2 > len(data) {
//			return
//		}
//		spsLen := int(data[pos])<<8 | int(data[pos+1])
//		pos += 2
//		if pos+spsLen > len(data) {
//			return
//		}
//		sps = append([]byte(nil), data[pos:pos+spsLen]...)
//		pos += spsLen
//	}
//	if pos >= len(data) {
//		return
//	}
//	ppsCount := int(data[pos])
//	pos++
//	for i := 0; i < ppsCount; i++ {
//		if pos+2 > len(data) {
//			return
//		}
//		ppsLen := int(data[pos])<<8 | int(data[pos+1])
//		pos += 2
//		if pos+ppsLen > len(data) {
//			return
//		}
//		pps = append([]byte(nil), data[pos:pos+ppsLen]...)
//		pos += ppsLen
//	}
//	return
//}
//
//// 判断 NAL 单元是否包含 IDR
//func isIDRFrame(naluPayload []byte) bool {
//	pos := 0
//	for pos+4 < len(naluPayload) {
//		naluLen := int(naluPayload[pos])<<24 | int(naluPayload[pos+1])<<16 | int(naluPayload[pos+2])<<8 | int(naluPayload[pos+3])
//		pos += 4
//		if pos >= len(naluPayload) {
//			return false
//		}
//		nalType := naluPayload[pos] & 0x1F
//		if nalType == 5 {
//			return true
//		}
//		pos += naluLen
//	}
//	return false
//}
//
//// 生成 AVCDecoderConfigurationRecord tag
//func makeVideoConfigTag(sps, pps []byte) []byte {
//	// 这里只是简单示例，可以根据 FLV 格式拼
//	// 生产环境建议用 go-flv 或 ffmpeg 生成
//	return nil
//}
//
//func main() {
//	cache := &GOPCache{}
//
//	srcFlvURL := "http://192.168.203.182:8080/live/livestream.flv"
//
//	// 拉取上游 HTTP-FLV 流并解析
//	go func() {
//		resp, err := http.Get(srcFlvURL)
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer resp.Body.Close()
//
//		d, _ := flv.NewDecoder(resp.Body)
//		for {
//			var t = tag.FlvTag{}
//			if err := d.Decode(&t); err != nil {
//				log.Println("stream end:", err)
//				return
//			}
//			cache.AddTag(&t)
//		}
//	}()
//
//	// Gin HTTP-FLV 输出
//	r := gin.Default()
//	r.GET("/live.flv", func(c *gin.Context) {
//		c.Header("Content-Type", "video/x-flv")
//		// 写 FLV header
//		c.Writer.Write([]byte{'F', 'L', 'V', 0x01, 0x05, 0, 0, 0, 9, 0, 0, 0, 0})
//
//		// 先发缓存
//		for _, pkt := range cache.Dump() {
//			c.Writer.Write(pkt)
//			c.Writer.Flush()
//		}
//		// 实时推
//		notify := c.Writer.CloseNotify()
//		ticker := make(chan []byte, 1024)
//		// TODO: 注册订阅推流
//		for {
//			select {
//			case <-notify:
//				return
//			case pkt := <-ticker:
//				c.Writer.Write(pkt)
//				c.Writer.Flush()
//			}
//		}
//	})
//	r.Run(":8080")
//}
