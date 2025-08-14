package main

import (
	"github.com/gin-contrib/cors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 每个流对应一个 broadcaster
type Broadcaster struct {
	mu       sync.Mutex
	clients  map[chan []byte]struct{}
	cache    [][]byte
	maxCache int
}

func NewBroadcaster(maxCache int) *Broadcaster {
	return &Broadcaster{
		clients:  make(map[chan []byte]struct{}),
		cache:    make([][]byte, 0, maxCache),
		maxCache: maxCache,
	}
}

// 添加数据到缓存并广播
func (b *Broadcaster) Broadcast(data []byte) {
	b.mu.Lock()
	// 添加到缓存
	b.cache = append(b.cache, data)
	if len(b.cache) > b.maxCache {
		b.cache = b.cache[1:]
	}
	// 广播给所有客户端
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// 客户端太慢，丢帧
		}
	}
	b.mu.Unlock()
}

// 新客户端订阅
func (b *Broadcaster) AddClient() chan []byte {
	ch := make(chan []byte, 1024)
	b.mu.Lock()
	// 先发送缓存数据
	for _, data := range b.cache {
		ch <- data
	}
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// 移除客户端
func (b *Broadcaster) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

// 流集合
var streamMap = struct {
	mu     sync.Mutex
	stream map[string]*Broadcaster
}{
	stream: make(map[string]*Broadcaster),
}

func getBroadcaster(name string) *Broadcaster {
	streamMap.mu.Lock()
	defer streamMap.mu.Unlock()
	b, ok := streamMap.stream[name]
	if !ok {
		b = NewBroadcaster(150) // 缓存150包，可调
		streamMap.stream[name] = b
	}
	return b
}

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

	// http://127.0.0.1:8080/ingest/test
	// ffmpeg 推流接口
	r.POST("/ingest/:stream", func(c *gin.Context) {
		//if !strings.HasPrefix(c.GetHeader("Content-Type"), "video/x-flv") {
		//	c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		//	return
		//}
		streamName := c.Param("stream")
		log.Println("开始推流:", streamName)
		b := getBroadcaster(streamName)

		buf := make([]byte, 32*1024)
		for {
			n, err := c.Request.Body.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				b.Broadcast(data)
			}
			if err != nil {
				if err != io.EOF {
					log.Println("读取错误:", err)
				}
				break
			}
		}
		log.Println("推流结束:", streamName)
	})

	// http://127.0.0.1:8080/live/test.flv
	// HTTP-FLV 拉流接口
	//r.GET("/live/:stream.flv", func(c *gin.Context) {
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

		for data := range ch {
			_, err := c.Writer.Write(data)
			if err != nil {
				break
			}
			flusher.Flush()
		}
	})

	r.Run(":8080")
}

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/ingest/test"
// http://127.0.0.1:8080/live/test
// http://127.0.0.1:8080/live/test.flv
