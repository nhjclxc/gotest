package main

import (
	"github.com/gin-contrib/cors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Broadcaster interface {
	AddClient() chan []byte
	RemoveClient(chan []byte)
	BroadcastTag([]byte, bool)
}

type CameraBroadcaster struct {
	clients map[chan []byte]struct{}
	gop     [][]byte
	mu      sync.Mutex
	maxGOP  int
}

func NewCameraBroadcaster() *CameraBroadcaster {
	return &CameraBroadcaster{
		clients: make(map[chan []byte]struct{}),
		gop:     make([][]byte, 0),
		maxGOP:  30, // 保存最近 30 个 Tag（大约 1 秒 GOP）
	}
}

func (b *CameraBroadcaster) AddClient() chan []byte {
	ch := make(chan []byte, 1024)
	b.mu.Lock()
	// 先把 GOP 发送给新客户端
	for _, tag := range b.gop {
		ch <- tag
	}
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *CameraBroadcaster) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *CameraBroadcaster) BroadcastTag(tag []byte, isKeyFrame bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if isKeyFrame {
		b.gop = [][]byte{tag}
	} else {
		b.gop = append(b.gop, tag)
		if len(b.gop) > b.maxGOP {
			b.gop = b.gop[len(b.gop)-b.maxGOP:]
		}
	}

	for ch := range b.clients {
		select {
		case ch <- tag:
		default:
			// 客户端阻塞就丢帧
		}
	}
}

var broadcasters = map[string]*CameraBroadcaster{}
var bMu sync.Mutex

func getBroadcaster(name string) *CameraBroadcaster {
	bMu.Lock()
	defer bMu.Unlock()
	b, ok := broadcasters[name]
	if !ok {
		b = NewCameraBroadcaster()
		broadcasters[name] = b
	}
	return b
}

/*
1、先启动go服务器，再打开前端页面，最后使用ffmpeng推流。这时页面可以正常显示摄像头的推流数据
2、先使用ffmpeg推流，再启动go服务器，最后打开页面，无法显示摄像头的推流数据。

这个问题必须解决，因为实际使用过程中绝大部分是情况2
*/
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

	// 推流接口，FFmpeg 使用 POST 推 FLV 数据
	r.POST("/live/ingest/:stream", func(c *gin.Context) {
		stream := c.Param("stream")
		b := getBroadcaster(stream)

		//if !c.Request.Header.Get("Content-Type") == "video/x-flv" {
		//	c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		//	return
		//}

		buf := make([]byte, 4096)
		for {
			n, err := c.Request.Body.Read(buf)
			if err != nil {
				break
			}
			data := make([]byte, n)
			copy(data, buf[:n])

			isKeyFrame := parseFLVTagIsKeyFrame(data)
			b.BroadcastTag(data, isKeyFrame)
		}

		c.Status(http.StatusOK)
	})

	// 播放接口，前端用 flv.js 或 VLC
	r.GET("/live/:stream", func(c *gin.Context) {
		stream := c.Param("stream")
		b := getBroadcaster(stream)

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
			case data, ok := <-ch:
				if !ok {
					return
				}
				_, err := c.Writer.Write(data)
				if err != nil {
					return
				}
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	})

	r.Run(":8080")
}

// 模拟解析 FLV 是否关键帧
func parseFLVTagIsKeyFrame(data []byte) bool {
	// 这里只是示例，可用真正的 FLV 解析库
	if len(data) > 13 && data[0]>>4 == 1 {
		return true
	}
	return false
}

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/ingest/test"
// http://127.0.0.1:8080/live/test
// http://127.0.0.1:8080/live/test.flv

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/ingest/test" -reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2
