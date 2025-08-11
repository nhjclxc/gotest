package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

/*
使用ffmpeg推流：ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream

拉流
● RTMP (by VLC): rtmp://192.168.203.182/live/livestream
● H5(HTTP-FLV): http://192.168.203.182:8080/live/livestream.flv
● H5(HLS): http://192.168.203.182:8080/live/livestream.m3u8



ffmpeg -re -i demo.flv \
    -c:v libx264 -preset veryfast -tune zerolatency \
    -g 25 -keyint_min 25 \
    -c:a aac -ar 44100 -b:a 128k \
    -f flv rtmp://192.168.203.182/live/livestream

*/

// ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream

var broadcaster *Broadcaster
var streamBroker *StreamBroker

func main() {

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	//r.Use(gin.Recovery())
	r.Use(CustomRecovery())

	upstreamURL := "http://192.168.203.182:8080/live/livestream.flv"
	streamBroker = NewStreamBroker("test1", upstreamURL)

	broadcaster = NewBroadcaster()
	broadcaster.AddStreamBroker(streamBroker)

	// health
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	// http://localhost:8080/live
	r.GET("/live/:brokerKey/:clientId", func(c *gin.Context) {
		c.Header("Content-Type", "video/x-flv")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		c.Writer.Header().Set("Content-Type", "video/x-flv")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.WriteHeader(http.StatusOK)

		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")

		streamBroker.RemoveHttpStreamClient(clientId)

		// 阻塞客户端
		//<-c.Request.Context().Done()

		//// 或者使用以下逻辑
		c.Stream(func(w io.Writer) bool {

			httpStreamClient, err := NewHttpStreamClient(c, brokerKey, clientId, streamBroker.dataCh, streamBroker.gOPCache)
			if err != nil {
				c.JSON(500, err)
				return false
			}

			streamBroker.AddHttpStreamClient(httpStreamClient)

			// 这里写数据推送逻辑，或者直接阻塞直到连接关闭
			<-c.Request.Context().Done()
			return false
		})

	})

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 这里写自定义日志处理
				log.Printf("panic recovered: %v\n", err)

				//		// 打印堆栈
				log.Printf("stack trace:\n%s", debug.Stack())
				// 发送告警、写日志等

				// 返回自定义错误响应
				c.AbortWithStatusJSON(500, gin.H{
					"message": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}

// ====================== Broadcaster ======================

// Broadcaster 广播器，负责协调所有的拉流链接 StreamBroker 直接的工作
type Broadcaster struct {
	brokers map[string]*StreamBroker // key为直播房间号
	mutex   sync.Mutex
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		brokers: make(map[string]*StreamBroker),
	}
}
func (b *Broadcaster) AddStreamBroker(streamBroker *StreamBroker) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.brokers[streamBroker.brokerKey] = streamBroker
}

// ====================== StreamBroker ======================

// StreamBroker 每个 直播地址 用一个 Broker 管理，里面管理了多个当前直播链接的客户端
type StreamBroker struct {
	brokerKey   string      // 直播房间的唯一编号
	upstreamURL string      // 直播房间的上游拉流地址
	dataCh      chan []byte // 上游拉流缓存的数据
	gOPCache    *GOPCache   // 保留关键帧数据

	clientMutex sync.Mutex                   // 客户端的异步操作控制器
	clientMap   map[string]*HttpStreamClient // map[clientId]HttpStreamClient 存储这个broker里面所有的客户端

	stopSig chan struct{} // 控制当前这个直播是否被关闭
	once    sync.Once
}

func NewStreamBroker(brokerKey, upstreamURL string) *StreamBroker {

	sb := StreamBroker{
		brokerKey:   brokerKey,
		upstreamURL: upstreamURL,
		gOPCache:    &GOPCache{tags: make([]*FlvTag, 0)},
		dataCh:      make(chan []byte, 4096), // 带缓冲，缓存 4096 个数据包
		clientMap:   make(map[string]*HttpStreamClient),
		stopSig:     make(chan struct{}),
	}

	// start pulling loop
	go sb.pullLoop()
	fmt.Printf("\n newBroker = %#v \n", &sb)

	return &sb
}

func (sb *StreamBroker) AddHttpStreamClient(httpStreamClient *HttpStreamClient) {
	sb.clientMutex.Lock()
	defer sb.clientMutex.Unlock()
	sb.clientMap[httpStreamClient.clientId] = httpStreamClient

	// when client closes, remove it
	go func() {
		<-httpStreamClient.closeSig
		sb.RemoveHttpStreamClient(httpStreamClient.clientId)
	}()
}

// RemoveHttpStreamClient 移除客户端
func (sb *StreamBroker) RemoveHttpStreamClient(clientId string) {
	sb.clientMutex.Lock()
	httpStreamClient := sb.clientMap[clientId]
	if httpStreamClient == nil {
		sb.clientMutex.Unlock()
		return
	}
	delete(sb.clientMap, clientId)
	sb.clientMutex.Unlock()

	close(httpStreamClient.dataCh)

	//// 如果没有客户端并且想释放 broker，可关闭 stopCh 让 pullLoop 停止（本示例保留 broker，防止频繁断开上游）
	//if remaining == 0 {
	//	// optionally stop pulling after idle timeout. For simplicity we keep running.
	//}
}

// pullLoop 持续去服务端拉流
func (b *StreamBroker) pullLoop() {
	backoff := time.Second
	for {
		log.Println("dial upstream", b.upstreamURL)
		req, _ := http.NewRequest("GET", b.upstreamURL, nil)
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

		// 成功连接，重置 backoff
		backoff = time.Second

		first := true
		// 读取本次拉到的流数据，并且进行数据分发
		buf := make([]byte, 4096)
		headerTemp := make([]byte, 0)
		totalSize := 0
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if first {
					if totalSize == 0 {
						dataSize := uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
						totalSize = 11 + int(dataSize) + 4 // tag header + data + prevTagSize
					}

					headerTemp = append(headerTemp, buf[:n]...)
					fmt.Println("构造headerTemp， ", len(headerTemp), "totalSize = ", totalSize)
					if len(headerTemp) >= totalSize {
						err = b.readOneFlvTag(headerTemp[:totalSize])
						if err != nil {
							log.Println("关键帧读取失败 readOneFlvTag error:", err)
						}

						first = false
					}
				}
				// copy bytes to avoid race
				cp := make([]byte, n)
				copy(cp, buf[:n])
				b.broadcast(cp)
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

// readOneFlvTag 从 src 读取一个完整的 FLV Tag，返回 Tag 和读取的字节数
func (sb *StreamBroker) readOneFlvTag(src []byte) error {
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

	sb.gOPCache.AddTag(tag)

	fmt.Println("一个tag构造完成！！！")

	return nil
}

func (b *StreamBroker) broadcast(data []byte) {
	//log.Println("StreamBroker.broadcast.brokerKey:", b.brokerKey)
	if len(b.clientMap) == 0 {
		return
	}
	b.clientMutex.Lock()
	// 复制 client list to avoid holding lock during send
	clients := make([]*HttpStreamClient, 0, len(b.clientMap))
	for _, c := range b.clientMap {
		clients = append(clients, c)
	}
	b.clientMutex.Unlock()

	// 广播数据 send non-blocking (drop if client's channel full)
	for _, c := range clients {
		//fmt.Printf("send non-blocking %#v \n\n", c.dataCh)
		select {
		case c.dataCh <- data:
			//fmt.Println("发送数据成功")
		default:
			// 客户端慢，丢包并继续
		}
	}
}

//1. 原因
//视频解码必须要从**关键帧（I-frame）**开始，否则画面会花屏或无法显示。
//如果新客户端连接时，没等到关键帧，就会卡在黑屏，直到下一个关键帧到来。
//所以，我们要在内存里保存最近的一组 GOP（关键帧 + 关键帧之后的所有非关键帧），新客户端连接时，先把这一组发过去。

// todo
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

// ====================== HttpStreamClient ======================

// HttpStreamClient 每一个前端页面有持有一个客户端对象
type HttpStreamClient struct {
	brokerKey string        // 这个客户端的直播房间的唯一编号
	clientId  string        // 这个客户端的id
	dataCh    chan []byte   // 这个客户端的一个只写通道
	closeSig  chan struct{} // broker被关闭时，同时通知客户端关闭

	// http连接相关
	httpRequest         *http.Request
	responseWriter      gin.ResponseWriter
	flusher             http.Flusher
	httpCloseSig        <-chan struct{} // 当这个请求被客户端主动被关闭时触发
	httpRequestCloseSig <-chan struct{} // 当这个请求被客户端主动被关闭时触发
}

func NewHttpStreamClient(c *gin.Context, brokerKey, clientId string, dataCh1 chan []byte, gOPCache *GOPCache) (*HttpStreamClient, error) {
	// 创建一个带缓冲的双向通道，缓冲大小根据需求调节
	dataCh := make(chan []byte, 4096)

	// gin.ResponseWriter 是接口，不能用指针
	writer := c.Writer

	// 断言出 http.Flusher 接口，方便主动刷新数据
	flusher, ok := writer.(http.Flusher)
	if !ok {
		// 不支持刷新，可能无法做到流式推送
		log.Println("ResponseWriter does not support Flusher interface")
	}

	hc := HttpStreamClient{
		brokerKey:           brokerKey,
		clientId:            clientId,
		dataCh:              dataCh,
		closeSig:            make(chan struct{}),
		httpRequest:         c.Request,
		responseWriter:      writer,
		flusher:             flusher,
		httpCloseSig:        c.Done(),
		httpRequestCloseSig: c.Request.Context().Done(),
	}

	fmt.Println("客户端连接成功 clientId = ", clientId)

	// 先发 GOP 缓存数据
	gop := gOPCache.GetTags()
	if len(gop) > 0 {
		data := gop[len(gop)-1].Data
		chunkSize := 4096
		for i := 0; i < len(data); i += chunkSize {
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}
			dataTemp := data[i:end]

			_, err := hc.responseWriter.Write(dataTemp)
			if err != nil {
				// 写出错，关闭连接
				return nil, errors.New("发送关键帧失败！！！" + err.Error())
			}
			hc.flusher.Flush()
		}

		_, err := hc.responseWriter.Write(gop[len(gop)-1].ToBytes())
		if err != nil {
			// 写出错，关闭连接
			return nil, errors.New("发送关键帧失败！！！" + err.Error())
		}
		hc.flusher.Flush()
		fmt.Println("关键帧发送完成！！！")
	}
	//
	//for _, tag := range gop {
	//	//b.broadcast(tag.ToBytes()) // 先广播给所有客户端
	//	_, err := hc.responseWriter.Write(tag.ToBytes())
	//	if err != nil {
	//		// 写出错，关闭连接
	//		return nil, errors.New("发送关键帧失败！！！")
	//	}
	//	hc.flusher.Flush()
	//	fmt.Println("关键帧发送完成！！！")
	//}

	// 持续监控是否有数据来，来数据就推送的前端
	go hc.PushStream()

	return &hc, nil
}

func (hc *HttpStreamClient) PushStream() {
	//defer func(hc0 *HttpStreamClient, responseWriter0 gin.ResponseWriter) {
	//	if err := recover(); err != nil {
	//		// 这里写自定义日志处理
	//		log.Printf("panic recovered: %v: %v\n", err, streamBroker)
	//		// 打印堆栈
	//		log.Printf("stack trace:\n%s", debug.Stack())
	//	}
	//}(hc, hc.responseWriter)
	for {
		select {
		case data, ok := <-hc.dataCh:
			//fmt.Printf("接收到数据 len = %d \n", len(data))
			if ok {

				if hc.responseWriter == nil || hc.flusher == nil {
					log.Println("responseWriter 或 flusher 已经无效，退出")
					return
				}

				_, err := hc.responseWriter.Write(data)
				if err != nil {
					// 写出错，关闭连接
					return
				}
				hc.flusher.Flush()
			}
		case <-hc.httpCloseSig:
			hc.closeSig <- struct{}{}
			// 收到关闭信号，退出循环
			fmt.Println("hc.httpCloseSig 收到客户端关闭信号，退出循环 ", hc.clientId)
			return
		case <-hc.httpRequestCloseSig:
			hc.closeSig <- struct{}{}
			// 收到关闭信号，退出循环
			fmt.Println("<-hc.httpRequestCloseSig 收到客户端关闭信号，退出循环 ", hc.clientId)
			return
		}
	}
}
