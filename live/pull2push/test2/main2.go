package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------- 配置 ----------
const (
	// 使用缓存时间3s的方法会导致后加入的客户端会慢3s左右，客户端会优先区缓存里面的字节数据，导致其区的数据一直比正常数据慢3s
	bufferDuration     = 3 * time.Second // 缓存时长（约 3s）
	readChunkSize      = 32 * 1024       // 从 ffmpeg stdout 读取块大小（32KB）
	clientChanBuf      = 64              // 每个 client channel 缓冲大小（避免短时阻塞）
	restartDelay       = 2 * time.Second // ffmpeg 异常退出后重启等待时间
	ffmpegStartTimeout = 5 * time.Second // 启动 ffmpeg 的短超时用于监测是否启动成功
)

// ---------- 内存缓冲区（基于时间窗口） ----------
type chunk struct {
	timestamp time.Time
	data      []byte
}

type timeBuffer struct {
	mu     sync.Mutex
	chunks []chunk
	// 总字节数用于快速修剪（可选）
	size int
}

// append 添加一个数据块并修剪旧数据（保持最近 bufferDuration）
func (tb *timeBuffer) append(data []byte) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tb.chunks = append(tb.chunks, chunk{timestamp: now, data: append([]byte(nil), data...)})
	tb.size += len(data)
	tb.trimLocked()
}

// readAll returns concatenated bytes of current buffer (用于新客户端的“预热”)
func (tb *timeBuffer) readAll() []byte {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.chunks) == 0 {
		return nil
	}
	// 计算总长度
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

// trimLocked 裁剪比 bufferDuration 更早的 chunk（内部使用，需持锁），丢弃bufferDuration时间段以前字节数据
func (tb *timeBuffer) trimLocked() {
	// 假设bufferDuration=3s，now为10:10:10
	// 那么cut=10:10:07
	// 即下面的操作就是保存timeBuffer里面的时间在10:10:07到10:10:10这个时间范围内的数据，其他的使用tb.chunks[i:]进行裁剪

	cut := time.Now().Add(-bufferDuration)
	i := 0
	for ; i < len(tb.chunks); i++ {
		// 第一个数据片在cut点以后 或者 第一个时间片对于当前的切点，则没有需要裁剪的数据
		if tb.chunks[i].timestamp.After(cut) || tb.chunks[i].timestamp.Equal(cut) {
			break
		}
		// 如果有数据超出了上面的时间段则裁剪数据，这里先把总数减下去，下面的if i>0进行数据切片
		tb.size -= len(tb.chunks[i].data)
	}
	if i > 0 {
		// 去除i以前的数据
		tb.chunks = tb.chunks[i:]
	}
	// 如果缓冲块过大（防止内存暴涨），可额外限制最大块数（这里不强制）
}

// Broadcaster ---------- 广播器 ----------
type Broadcaster struct {
	src       string
	ctx       context.Context
	cancel    context.CancelFunc
	buf       *timeBuffer         // 拉流转推的中间数据存储器
	clients   map[int]chan []byte // 客户端与通道的映射关系
	mu        sync.Mutex
	nextID    int // 用于标识链接的客户端，类似数据库的自增id
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

// Stop 关闭广播器并停止 ffmpeg
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

// RegisterClient 注册客户端并返回一个 channel 来接收后续数据；同时返回当前 buffer 初始内容
func (b *Broadcaster) RegisterClient() (id int, ch <-chan []byte, init []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id = b.nextID
	b.nextID++
	c := make(chan []byte, clientChanBuf)
	b.clients[id] = c

	// 这里会导致新加的客户端会慢3s左右
	init = b.buf.readAll() // 先把缓存复制出来（线程安全）
	return id, c, init
}

// UnregisterClient 移除客户端
func (b *Broadcaster) UnregisterClient(id int) {
	b.mu.Lock()
	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
	b.mu.Unlock()
}

// broadcast 将数据写入内存缓冲并向所有客户端分发（非阻塞：无法写入的客户端会丢帧）
//
// 参数：
//
//	p 当前读取到的拉流数据。
func (b *Broadcaster) broadcast(p []byte) {
	// append to time buffer
	b.buf.append(p)

	b.mu.Lock()
	defer b.mu.Unlock()
	// 将最新的数据推送给所有客户端
	for id, ch := range b.clients {
		select {
		case ch <- append([]byte(nil), p...): // 复制数据到客户端 channel（避免后续覆盖）
		default:
			// 如果客户端接收太慢（channel 满），直接丢弃这次帧，避免阻塞广播器
			log.Printf("client %d slow — drop packet", id)
		}
	}
}

// ffmpegLoop 启动 ffmpeg，读取 stdout 并广播；若 ffmpeg 退出则重启（除非上下文取消）
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
			"-loglevel", "error", // 减少噪音，可改为 info/debug
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

		// 从 stderr 打印少量日志（异步）
		go func() {
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				log.Printf("[ffmpeg] %s", sc.Text())
			}
		}()

		// 读取 stdout 并 broadcast
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

		// 短时间内若能读到数据，说明启动成功；否则认为启动失败并重试
		started := false
		select {
		case <-time.After(ffmpegStartTimeout):
			// 如果 buffer 里已有内容（表示 ffmpeg 已经生产），则 ok
			if len(b.buf.readAll()) > 0 {
				started = true
			}
		case err := <-readErr:
			// 读取立刻返回错误 => 启动失败
			log.Printf("ffmpeg read immediate error: %v", err)
		case <-time.After(10 * time.Millisecond):
			// tiny sleep to let data arrive
			if len(b.buf.readAll()) > 0 {
				started = true
			}
		}

		if !started {
			log.Printf("ffmpeg seems not producing data yet; will wait and restart")
		}

		// 等待读取或命令退出
		err = <-readErr
		// 如果读到 EOF，说明 ffmpeg 退出或被 cancel
		if err != nil && err != io.EOF {
			log.Printf("ffmpeg read error: %v", err)
		} else {
			log.Printf("ffmpeg stdout closed")
		}

		// 等待 cmd 结束并确保进程退出
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		cmdCancel()

		// 如果用户取消整个广播器上下文，则退出循环
		select {
		case <-b.ctx.Done():
			return
		default:
			// 等待一段时间重启 ffmpeg
			time.Sleep(restartDelay)
			// 清空 buffer? 一般保留最近 bufferDuration 的内容
			// 继续循环以重启 ffmpeg
		}
	}
}

// ---------- HTTP Handler（Gin） ----------
func flvHandler(b *Broadcaster) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置头
		c.Writer.Header().Set("Content-Type", "video/x-flv")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		// 注册客户端
		id, ch, init := b.RegisterClient()
		defer b.UnregisterClient(id)
		log.Printf("client %d connected", id)

		// 使用 http.Flusher 强制分块输出
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError, "streaming unsupported")
			return
		}

		// 先把 buffer 的内容发给客户端（快速预热）
		if len(init) > 0 {
			if _, err := c.Writer.Write(init); err != nil {
				log.Printf("write init to client %d error: %v", id, err)
				return
			}
			flusher.Flush()
		}

		// 然后把后续数据从 channel 发送给客户端，直到断开
		notify := c.Request.Context().Done()
		for {
			select {
			case <-notify:
				log.Printf("client %d disconnected (context done)", id)
				return
			case p, ok := <-ch:
				if !ok {
					log.Printf("client %d channel closed", id)
					return
				}
				_, err := c.Writer.Write(p)
				if err != nil {
					log.Printf("write to client %d error: %v", id, err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

func main() {
	src := "http://192.168.203.182:8080/live/livestream.m3u8"

	b := NewBroadcaster(src)
	b.Start()
	defer b.Stop()

	router := gin.Default()
	// http://localhost:8080/live.flv
	// http://192.168.203.182:8080/live/livestream.m3u8
	router.GET("/live.flv", flvHandler(b))
	router.GET("/", func(c *gin.Context) {
		c.String(200, fmt.Sprintf("HTTP-FLV proxy running. Pulling: %s\nEndpoint: /live.flv", src))
	})

	srvAddr := ":8080"
	log.Printf("listening on %s", srvAddr)
	if err := router.Run(srvAddr); err != nil {
		log.Fatalf("gin run error: %v", err)
	}
}
