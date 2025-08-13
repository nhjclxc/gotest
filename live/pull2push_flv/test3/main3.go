package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ---------- 配置 ----------
const (
	bufferDuration     = 3 * time.Second // 缓存时长（约 3s）
	readChunkSize      = 32 * 1024       // 从 ffmpeg stdout 读取块大小（32KB）
	clientChanBuf      = 128             // 每个 client channel 缓冲大小
	restartDelay       = 2 * time.Second // ffmpeg 异常退出后重启等待时间
	ffmpegStartTimeout = 5 * time.Second
	wsWriteTimeout     = 10 * time.Second
)

// ---------- timeBuffer（同前） ----------
type chunk struct {
	timestamp time.Time
	data      []byte
}

type timeBuffer struct {
	mu     sync.Mutex
	chunks []chunk
	size   int
}

func (tb *timeBuffer) append(data []byte) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tb.chunks = append(tb.chunks, chunk{timestamp: now, data: append([]byte(nil), data...)})
	tb.size += len(data)
	tb.trimLocked()
}

func (tb *timeBuffer) readAll() []byte {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.chunks) == 0 {
		return nil
	}
	total := 0
	for _, c := range tb.chunks {
		total += len(c.data)
	}
	out := make([]byte, 0, total)
	for _, c := range tb.chunks {
		out = append(out, c.data...)
	}
	return out
}

func (tb *timeBuffer) trimLocked() {
	cut := time.Now().Add(-bufferDuration)
	i := 0
	for ; i < len(tb.chunks); i++ {
		if tb.chunks[i].timestamp.After(cut) || tb.chunks[i].timestamp.Equal(cut) {
			break
		}
		tb.size -= len(tb.chunks[i].data)
	}
	if i > 0 {
		tb.chunks = tb.chunks[i:]
	}
}

// ---------- Broadcaster（改造以支持 websocket 客户端） ----------
type Broadcaster struct {
	src       string
	ctx       context.Context
	cancel    context.CancelFunc
	buf       *timeBuffer
	clients   map[int]chan []byte
	mu        sync.Mutex
	nextID    int
	wg        sync.WaitGroup
	startOnce sync.Once
}

func NewBroadcaster(src string) *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broadcaster{
		src:     src,
		ctx:     ctx,
		cancel:  cancel,
		buf:     &timeBuffer{},
		clients: make(map[int]chan []byte),
	}
}

func (b *Broadcaster) Start() {
	b.startOnce.Do(func() {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.ffmpegLoop()
		}()
	})
}

func (b *Broadcaster) Stop() {
	b.cancel()
	b.mu.Lock()
	for id, ch := range b.clients {
		close(ch)
		delete(b.clients, id)
	}
	b.mu.Unlock()
	b.wg.Wait()
}

func (b *Broadcaster) RegisterClient() (id int, ch <-chan []byte, init []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id = b.nextID
	b.nextID++
	c := make(chan []byte, clientChanBuf)
	b.clients[id] = c
	init = b.buf.readAll()
	return id, c, init
}

func (b *Broadcaster) UnregisterClient(id int) {
	b.mu.Lock()
	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
	b.mu.Unlock()
}

func (b *Broadcaster) broadcast(p []byte) {
	b.buf.append(p)

	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.clients {
		select {
		case ch <- append([]byte(nil), p...):
		default:
			// 慢客户端，丢帧
			log.Printf("client %d slow — drop packet", id)
		}
	}
}

func (b *Broadcaster) ffmpegLoop() {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		log.Printf("starting ffmpeg pull from %s", b.src)
		cmdCtx, cmdCancel := context.WithCancel(b.ctx)
		cmd := exec.CommandContext(cmdCtx, "ffmpeg",
			"-i", b.src,
			"-c", "copy",
			"-f", "flv",
			"-loglevel", "error",
			"-")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("ffmpeg StdoutPipe error: %v", err)
			cmdCancel()
			time.Sleep(restartDelay)
			continue
		}
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			log.Printf("ffmpeg start error: %v", err)
			cmdCancel()
			time.Sleep(restartDelay)
			continue
		}

		go func() {
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				log.Printf("[ffmpeg] %s", sc.Text())
			}
		}()

		reader := bufio.NewReaderSize(stdout, readChunkSize)
		readErr := make(chan error, 1)
		go func() {
			for {
				select {
				case <-cmdCtx.Done():
					readErr <- io.EOF
					return
				default:
				}
				buf := make([]byte, readChunkSize)
				n, err := reader.Read(buf)
				if n > 0 {
					b.broadcast(buf[:n])
				}
				if err != nil {
					if err == io.EOF {
						readErr <- io.EOF
						return
					}
					readErr <- err
					return
				}
			}
		}()

		// 等待或重启判定
		started := false
		select {
		case <-time.After(ffmpegStartTimeout):
			if len(b.buf.readAll()) > 0 {
				started = true
			}
		case err := <-readErr:
			log.Printf("ffmpeg read immediate error: %v", err)
		case <-time.After(10 * time.Millisecond):
			if len(b.buf.readAll()) > 0 {
				started = true
			}
		}

		if !started {
			log.Printf("ffmpeg seems not producing data yet; will wait and restart")
		}

		err = <-readErr
		if err != nil && err != io.EOF {
			log.Printf("ffmpeg read error: %v", err)
		} else {
			log.Printf("ffmpeg stdout closed")
		}

		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		cmdCancel()

		select {
		case <-b.ctx.Done():
			return
		default:
			time.Sleep(restartDelay)
		}
	}
}

// ---------- WebSocket handler ----------
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域（按需修改）
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(b *Broadcaster) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}
		defer ws.Close()

		// 注册客户端
		id, ch, init := b.RegisterClient()
		defer b.UnregisterClient(id)
		log.Printf("ws client %d connected", id)

		// 发送 init buffer（作为 binary frame）
		if len(init) > 0 {
			ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ws.WriteMessage(websocket.BinaryMessage, init); err != nil {
				log.Printf("write init to client %d err: %v", id, err)
				return
			}
		}

		// 启动读协程，仅用于接收客户端关闭（websocket 需要读以检测断开）
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_, _, err := ws.ReadMessage()
				if err != nil {
					// 通常会在客户端关闭时触发
					return
				}
			}
		}()

		// 主写循环：把后续 ch 中的数据写入 websocket（二进制帧）
		for {
			select {
			case <-done:
				log.Printf("ws client %d disconnected (read loop)", id)
				return
			case p, ok := <-ch:
				if !ok {
					log.Printf("client %d channel closed", id)
					return
				}
				ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
					log.Printf("write to ws client %d error: %v", id, err)
					return
				}
			case <-time.After(30 * time.Second):
				// 心跳以防长时间无数据时连接被中间网络设备关闭
				ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("ping to client %d failed: %v", id, err)
					return
				}
			}
		}
	}
}

func main() {
	src := os.Getenv("PULL_SRC")
	if src == "" {
		src = "http://192.168.203.182:8080/live/livestream.m3u8"
	}

	b := NewBroadcaster(src)
	b.Start()
	defer b.Stop()

	router := gin.Default()
	router.GET("/ws/flv", wsHandler(b))
	router.GET("/", func(c *gin.Context) {
		c.String(200, fmt.Sprintf("WebSocket-FLV proxy running. Pulling: %s\nWS endpoint: /ws/flv", src))
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("gin run error: %v", err)
	}
}
