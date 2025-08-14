package flv

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"pull2push/core/client"
	"sync"
	"time"
)

// ====================== FLVStreamBroker ======================

// FLVStreamBroker 每个 直播地址 用一个 Broker 管理，里面管理了多个当前直播链接的客户端
type FLVStreamBroker struct {
	BrokerKey   string      // 直播房间的唯一编号
	UpstreamURL string      // 直播房间的上游拉流地址
	DataCh      chan []byte // 上游拉流缓存的数据
	GOPCache    *GOPCache   // 保留关键帧数据
	flvParser   *FLVParser  // 创建FLV解析器 (启用调试模式)

	HeaderMutex  sync.RWMutex
	HeaderBytes  []byte
	HeaderParsed bool

	clientMutex sync.Mutex                   // 客户端的异步操作控制器
	clientMap   map[string]client.LiveClient // map[clientId]LiveFLVClient 存储这个broker里面所有的客户端

	stopSig chan struct{} // 控制当前这个直播是否被关闭
	once    sync.Once
}

func NewFLVStreamBroker(brokerKey, upstreamURL string) *FLVStreamBroker {

	b := FLVStreamBroker{
		BrokerKey:   brokerKey,
		UpstreamURL: upstreamURL,
		GOPCache:    &GOPCache{tags: make([]*FlvTag, 0)},
		DataCh:      make(chan []byte, 4096), // 带缓冲，缓存 4096 个数据包
		clientMap:   make(map[string]client.LiveClient),
		stopSig:     make(chan struct{}),
	}

	// start pulling loop
	go b.PullLoop()
	fmt.Printf("\n newBroker = %#v \n", &b)

	return &b
}

func (sb *FLVStreamBroker) AddLiveClient(clientId string, client client.LiveClient) {
	sb.clientMutex.Lock()
	defer sb.clientMutex.Unlock()

	sb.clientMap[clientId] = client
}

// RemoveClient 移除客户端
func (sb *FLVStreamBroker) RemoveLiveClient(clientId string) {
	sb.clientMutex.Lock()
	defer sb.clientMutex.Unlock()
	liveClient := sb.clientMap[clientId]
	if liveClient == nil {
		return
	}
	delete(sb.clientMap, clientId)

	//// 如果没有客户端并且想释放 broker，可关闭 stopCh 让 PullLoop 停止（本示例保留 broker，防止频繁断开上游）
	//if remaining == 0 {
	//	// optionally stop pulling after idle timeout. For simplicity we keep running.
	//}
}

// FindLiveClient 查询 LiveClient
func (fsb *FLVStreamBroker) FindLiveClient(clientId string) (client.LiveClient, error) {
	if val, ok := fsb.clientMap[clientId]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("未找到 %s 对应的 LiveClient", clientId))
}

// UpdateSourceURL 支持切换直播原地址
func (b *FLVStreamBroker) UpdateSourceURL(newSourceURL string) {

}

// ListenStatus 监听当前直播的必要状态
func (b *FLVStreamBroker) ListenStatus() {

}

// PullLoop 持续去服务端拉流
func (b *FLVStreamBroker) PullLoop() {
	backoff := time.Second
	for {
		log.Println("dial upstream", b.UpstreamURL)
		req, _ := http.NewRequest("GET", b.UpstreamURL, nil)
		// add headers typical for FLV
		req.Header.Set("User-Agent", "Go-Relay-Flv/1.0")
		req.Header.Set("Accept", "*/*")
		client := &http.Client{
			Timeout: 0, // streaming
			Transport: &http.Transport{
				// keep-alive
				DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			},
		}
		// 拉流
		resp, err := client.Do(req)
		// 失败重试
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Println("upstream bad status:", resp.Status)
			resp.Body.Close()
			time.Sleep(backoff)
			continue
		}

		//b.doFlvParse(resp.Body)

		// 成功连接，重置 backoff
		backoff = time.Second

		first := true
		// 读取本次拉到的流数据，并且进行数据分发
		buf := make([]byte, 4096)
		//headerTemp := make([]byte, 0)
		//totalSize := 0
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if first {
					//if totalSize == 0 {
					//	dataSize := uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
					//	totalSize = 11 + int(dataSize) + 4 // tag header + data + prevTagSize
					//}
					//
					//headerTemp = append(headerTemp, buf[:n]...)
					//fmt.Println("构造headerTemp， ", len(headerTemp), "totalSize = ", totalSize)
					//if len(headerTemp) >= totalSize {
					//	err = b.readOneFlvTag(headerTemp[:totalSize])
					//	if err != nil {
					//		log.Println("关键帧读取失败 readOneFlvTag error:", err)
					//	}
					//
					//	first = false
					//}

				}
				// copy bytes to avoid race
				cp := make([]byte, n)
				copy(cp, buf[:n])
				b.Broadcast2LiveClient(cp)
			}
			if err != nil {
				if err == io.EOF {
					log.Println("upstream EOF, reconnecting", "退出拉流过程")
					break
				} else if "unexpected EOF" == err.Error() {
					log.Println("upstream read error:", "上游直播关闭", "退出拉流过程")
					break
				} else {
					log.Println("upstream read error:", err)
				}
				resp.Body.Close()
				break
			}
		}

		// 如果 stop 信号被触发，可以退出（此实现未触发 stop）
		select {
		case <-b.stopSig:
			return
		default:
		}

		// small backoff before reconnect
		time.Sleep(500 * time.Millisecond)
	}
}

func (sb *FLVStreamBroker) doFlvParse(body io.ReadCloser) {

	ctx := context.Background()
	_ = ctx
	// 创建FLV解析器 (启用调试模式)
	sb.flvParser = NewFLVParser(true)

	fmt.Println("开始解析FLV流...")
	fmt.Println("等待解析视频和音频标签...")

	// 使用带超时的方法解析FLV头部和初始标签
	err := sb.flvParser.ParseInitialTags(ctx, body)
	if err != nil {
		fmt.Printf("解析FLV失败: %v\n", err)

		panic(err)
	}
	sb.flvParser.PrintRequiredTags()

	// 获取必要的FLV头和标签字节
	headerBytes, err := sb.flvParser.GetRequiredTagsBytes()
	if err != nil {
		body.Close()
		fmt.Println("获取FLV头部字节失败: %v", err)
	}

	// 保存头部字节供后续使用
	sb.HeaderMutex.Lock()
	sb.HeaderBytes = headerBytes
	sb.HeaderParsed = true
	sb.HeaderMutex.Unlock()

	slog.Info("FLV头部解析完成，等待客户端连接开始转发", "bytes", len(headerBytes))

}

// readOneFlvTag 从 src 读取一个完整的 FLV Tag，返回 Tag 和读取的字节数
func (sb *FLVStreamBroker) readOneFlvTag(src []byte) error {
	if len(src) < 15 {
		return fmt.Errorf("data too short for FLV tag header")
	}

	// FLV tag header 11字节
	tagType := src[0]
	dataSize := uint32(src[1])<<16 | uint32(src[2])<<8 | uint32(src[3])
	timestamp := uint32(src[4])<<16 | uint32(src[5])<<8 | uint32(src[6]) | (uint32(src[7]) << 24)

	streamID := uint32(src[8])<<16 | uint32(src[9])<<8 | uint32(src[10])

	totalSize := 11 + int(dataSize) + 4 // tag header + data + prevTagSize

	if len(src) < totalSize {
		return fmt.Errorf("data too short for full FLV tag, need %d got %d", totalSize, len(src))
	}

	data := make([]byte, dataSize)
	copy(data, src[11:11+dataSize])

	prevTagSize := uint32(src[11+dataSize])<<24 | uint32(src[11+dataSize+1])<<16 | uint32(src[11+dataSize+2])<<8 | uint32(src[11+dataSize+3])

	tag := &FlvTag{
		TagType:     tagType,
		DataSize:    dataSize,
		Timestamp:   timestamp,
		StreamID:    streamID,
		Data:        data,
		PrevTagSize: prevTagSize,
	}

	sb.GOPCache.AddTag(tag)

	fmt.Println("一个tag构造完成！！！")

	return nil
}

func (b *FLVStreamBroker) Broadcast2LiveClient(data []byte) {
	//log.Println("FLVStreamBroker.Broadcast.BrokerKey:", b.BrokerKey)
	if len(b.clientMap) == 0 {
		return
	}
	b.clientMutex.Lock()
	// 复制 client list to avoid holding lock during send
	clients := make([]client.LiveClient, 0, len(b.clientMap))
	for _, c := range b.clientMap {
		clients = append(clients, c)
	}
	b.clientMutex.Unlock()

	// 广播数据 send non-blocking (drop if client's channel full)
	for _, c := range clients {
		c.Broadcast(data)
		////fmt.Printf("send non-blocking %#v \n\n", c.dataCh)
		//select {
		//case c.DataCh <- data:
		//	//fmt.Println("发送数据成功")
		//default:
		//	// 客户端慢，丢包并继续
		//}
	}
}

//1. 原因
//视频解码必须要从**关键帧（I-frame）**开始，否则画面会花屏或无法显示。
//如果新客户端连接时，没等到关键帧，就会卡在黑屏，直到下一个关键帧到来。
//所以，我们要在内存里保存最近的一组 GOP（关键帧 + 关键帧之后的所有非关键帧），新客户端连接时，先把这一组发过去。

// GOPCache 要让晚加入的前端也能播放，你的 Go 服务端必须缓存起始包（FLV header + metadata + 第一个关键帧），每次新客户端连上来都先推送它们。
// type GOPCache struct
// FLV 里 Video Tag 的 FrameType = 1 表示关键帧（I-frame）。
// 步骤 1：解析上游流数据
type GOPCache struct {
	mu   sync.Mutex
	tags []*FlvTag // 存储一组 GOP（关键帧 + 之后的帧）
}

// 步骤 2：建立 GOP 缓存
// AddTag：每来一帧都存起来，如果是关键帧，就清空之前的，重新开始存。
func (c *GOPCache) AddTag(tag *FlvTag) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果是关键帧，清空之前的缓存
	if tag.IsKeyFrame() {
		c.tags = nil
	}
	c.tags = append(c.tags, tag)
}

// 步骤 3：新客户端连接时发 GOP 缓存
// GetTags：新客户端连接时，先把缓存的 GOP 发过去。
func (c *GOPCache) GetTags() []*FlvTag {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*FlvTag(nil), c.tags...)
}

type FlvTag struct {
	TagType     uint8  // 1 byte: Tag Type (8 = audio, 9 = video, 18 = script data)
	DataSize    uint32 // 3 bytes, 实际是24bit，表示数据大小（不含11字节头和4字节PreviousTagSize）
	Timestamp   uint32 // 4 bytes, 其中最高字节是扩展时间戳
	StreamID    uint32 // 3 bytes, 总是0
	Data        []byte // DataSize大小的有效载荷
	PrevTagSize uint32 // 4 bytes, 上一个tag的总大小（含头和数据）
}

// ToBytes 序列化 FlvTag 成字节切片
func (tag *FlvTag) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)

	// 写TagType 1字节
	buf.WriteByte(tag.TagType)

	// 写DataSize 3字节（24bit）
	dataSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(dataSizeBytes, tag.DataSize)
	buf.Write(dataSizeBytes[1:4]) // 取后三字节

	// 写Timestamp 3字节 + 1字节扩展
	timestampBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timestampBytes, tag.Timestamp)
	buf.Write(timestampBytes[1:4])   // 3字节时间戳低位
	buf.WriteByte(timestampBytes[0]) // 1字节扩展时间戳

	// 写StreamID 3字节 总是0
	buf.Write([]byte{0x00, 0x00, 0x00})

	// 写Data有效载荷
	buf.Write(tag.Data)

	// 写PreviousTagSize 4字节，等于 tag头(11) + 数据大小
	prevTagSize := uint32(11 + tag.DataSize)
	prevTagSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(prevTagSizeBytes, prevTagSize)
	buf.Write(prevTagSizeBytes)

	return buf.Bytes()
}

func (tag *FlvTag) IsKeyFrame() bool {
	if tag.TagType != 9 { // 9 是视频Tag
		return false
	}
	if len(tag.Data) == 0 {
		return false
	}
	frameType := (tag.Data[0] & 0xF0) >> 4
	return frameType == 1
}
