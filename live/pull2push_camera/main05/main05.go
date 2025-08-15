package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ----------------------------
// CameraBroadcaster
// ----------------------------
type CameraBroadcaster struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	gop     [][]byte // 缓存最近一个 GOP（关键帧 + 后续帧）
}

func NewCameraBroadcaster() *CameraBroadcaster {
	return &CameraBroadcaster{
		clients: make(map[chan []byte]struct{}),
		gop:     make([][]byte, 0),
	}
}

// 添加客户端
func (b *CameraBroadcaster) AddClient() chan []byte {
	ch := make(chan []byte, 1024)
	b.mu.Lock()
	defer b.mu.Unlock()

	// 先把缓存的 GOP 发送给新客户端
	for _, pkt := range b.gop {
		ch <- pkt
	}

	b.clients[ch] = struct{}{}
	return ch
}

// 移除客户端
func (b *CameraBroadcaster) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

// 广播数据
func (b *CameraBroadcaster) Broadcast(data []byte, isKeyFrame bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 如果是关键帧，重置 GOP 缓存
	if isKeyFrame {
		b.gop = b.gop[:0]
	}
	// 缓存
	b.gop = append(b.gop, data)

	// 广播给所有客户端
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// 如果客户端慢导致缓存满，丢帧
		}
	}
}

// ----------------------------
// 全局管理器
// ----------------------------
var broadcasters = map[string]*CameraBroadcaster{}
var mu sync.Mutex

func getBroadcaster(name string) *CameraBroadcaster {
	mu.Lock()
	defer mu.Unlock()
	b, ok := broadcasters[name]
	if !ok {
		b = NewCameraBroadcaster()
		broadcasters[name] = b
	}
	return b
}

// ----------------------------
// main
// ----------------------------
func main() {
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "android-device-id", "Content-Type", "Accept", "Authorization", "X-Token", "request-time", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 推流接口：FFmpeg 通过 POST 推送 FLV
	r.POST("/live/ingest/:stream", func(c *gin.Context) {
		streamName := c.Param("stream")
		b := getBroadcaster(streamName)

		//if c.GetHeader("Content-Type") != "video/x-flv" {
		//	c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		//	return
		//}

		data := make([]byte, 4096)
		for {
			n, err := c.Request.Body.Read(data)
			if err != nil {
				fmt.Println("推流断开:", err)
				break
			}
			packet := make([]byte, n)
			copy(packet, data[:n])

			// 简化：这里假设每个 packet 都是关键帧或帧数据
			// 实际项目中需要解析 FLV tag 判断关键帧
			isKeyFrame := packet[0]&0x10 != 0
			b.Broadcast(packet, isKeyFrame)
		}

		c.Status(http.StatusOK)
	})

	// 播放接口：前端浏览器或 VLC 获取 FLV 流
	r.GET("/live/:stream", func(c *gin.Context) {
		streamName := c.Param("stream")
		b := getBroadcaster(streamName)

		c.Header("Content-Type", "video/x-flv")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError, "Streaming unsupported")
			return
		}

		ch := b.AddClient()
		defer b.RemoveClient(ch)

		for {
			select {
			case pkt, ok := <-ch:
				if !ok {
					return
				}
				_, err := c.Writer.Write(pkt)
				if err != nil {
					return
				}
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	})

	fmt.Println("Go FLV server started on :8080")
	r.Run(":8080")
}

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/ingest/test"
// http://127.0.0.1:8080/live/test
// http://127.0.0.1:8080/live/test.flv

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/ingest/test" -reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2
// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/ingest/test"
