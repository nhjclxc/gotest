package main

import (
	"fmt"
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

	gopMU  sync.RWMutex
	header []byte
	gop    [][]byte
}

func NewBroadcaster(maxCache int) *Broadcaster {
	return &Broadcaster{
		clients:  make(map[chan []byte]struct{}),
		cache:    make([][]byte, 0, maxCache),
		maxCache: maxCache,
		header:   buildFLVHeader(), // 生成 FLV 文件头
	}
}
func buildFLVHeader() []byte {
	header := []byte{
		'F', 'L', 'V', // Signature
		0x01,                   // Version
		0x05,                   // Flags: audio+video
		0x00, 0x00, 0x00, 0x09, // Header length
		0x00, 0x00, 0x00, 0x00, // PreviousTagSize0
	}
	return header
}
func (b *Broadcaster) BroadcastTag(tag []byte, isKeyFrame bool) {
	b.mu.Lock()
	if isKeyFrame {
		b.gop = [][]byte{tag} // 新关键帧，清空 GOP
	} else {
		b.gop = append(b.gop, tag)
	}
	for ch := range b.clients {
		select {
		case ch <- tag:
		default:
			// 客户端阻塞就丢帧
		}
	}
	b.mu.Unlock()
}

func (b *Broadcaster) GetGOP() [][]byte {
	b.gopMU.RLock()
	defer b.gopMU.RUnlock()
	gopCopy := make([][]byte, len(b.gop))
	copy(gopCopy, b.gop)
	return gopCopy
}

func (b *Broadcaster) GetHeader() []byte {
	return b.header
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
			fmt.Println("广播数据")
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
	fmt.Println("信道关闭")
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
	r.GET("/live/:stream", func(c *gin.Context) {
		streamName := c.Param("stream")
		b := getBroadcaster(streamName)

		// 提前拿到 ResponseWriter 和 Flusher；不要把 gin.Context 存到别处
		w := c.Writer
		flusher, ok := w.(http.Flusher)
		if !ok || flusher == nil {
			c.String(http.StatusInternalServerError, "Streaming unsupported")
			return
		}

		// HTTP 头 & 立即刷一次，建立分块传输
		w.Header().Set("Content-Type", "video/x-flv")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		c.Status(http.StatusOK)
		flusher.Flush()

		// 订阅本流（每个客户端一个独立通道）
		ch := b.AddClient()
		defer b.RemoveClient(ch)

		// 1. 先发 FLV 文件头
		header := b.GetHeader() // []byte，包含 FLV + metadata
		c.Writer.Write(header)
		flusher.Flush()

		// 2. 再发缓存的 GOP（关键帧）
		gop := b.GetGOP() // [][]byte，多个 tag
		for _, tag := range gop {
			c.Writer.Write(tag)
		}
		flusher.Flush()

		// 只用这个协程写回给客户端；不要从其它协程写 w
		for {
			select {
			// 监听客户端断开
			case <-c.Request.Context().Done():
				// 客户端断开或服务器关闭连接
				fmt.Println("客户端断开或服务器关闭连接")
				return

			case data, ok := <-ch:
				if !ok {
					fmt.Println("广播器关闭")
					return // 广播器关闭
				}
				// 分块写，防止大包越界或 short write
				if err := writeChunked(w, data, 32*1024); err != nil {
					// 写失败，结束本客户端
					fmt.Println("写失败，结束本客户端")
					return
				}
				// Flush 一次，把缓冲区推给浏览器
				// flusher 是从 w 断言出来的 flusher；此处不再访问 c.Writer
				safeFlush(flusher)
				fmt.Println(" Flush 一次，把缓冲区推给浏览器")
			}
		}

		// 阻塞客户端
		<-c.Request.Context().Done()
	})

	//r.GET("/live/:stream.flv", func(c *gin.Context) {
	r.GET("/live22/:stream111", func(c *gin.Context) {
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

		go func() {
			for {
				select {
				case data := <-ch:
					_, err := c.Writer.Write(data)
					if err != nil {
						break
					}
					flusher.Flush()
				}
			}
		}()

		// 阻塞客户端
		<-c.Request.Context().Done()

	})

	r.Run(":8080")
}

// 分块循环写，避免一次写太大导致越界或短写
func writeChunked(w io.Writer, data []byte, chunkSize int) error {
	if chunkSize <= 0 {
		chunkSize = 32 * 1024
	}
	total := len(data)
	offset := 0
	for offset < total {
		remain := total - offset
		size := chunkSize
		if remain < size {
			size = remain
		}
		n, err := w.Write(data[offset : offset+size])
		if err != nil {
			return fmt.Errorf("write failed: %w", err)
		}
		if n != size {
			return fmt.Errorf("short write: expect=%d got=%d", size, n)
		}
		offset += size
	}
	return nil
}

// 防御性 Flush：即使底层实现为 nil 指针也不崩
func safeFlush(f http.Flusher) {
	defer func() {
		_ = recover() // 防止底层连接突然失效导致 panic
	}()
	if f != nil {
		f.Flush()
	}
}

// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/ingest/test"
// http://127.0.0.1:8080/live/test
// http://127.0.0.1:8080/live/test.flv
