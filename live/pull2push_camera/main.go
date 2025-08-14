package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ------------------------------
// 设计说明（摘要）
// - 手机端（或任意推流端）将 FLV 流推到 Go 服务器：
//   * 方式 A：HTTP 推流  POST /ingest/:stream   (Content-Type: video/x-flv)
//   * 方式 B：WebSocket 推流 WS   /publish/:stream
//     两种方式都只要把"原始 FLV 字节"按顺序写入 body/WS 即可。
// - 服务器维护一个带短时缓存（默认 3s 或最大 4MB）的环形缓冲区，并对每个流进行多路分发。
// - 网页端拉流：HTTP-FLV  GET /live/:stream.flv   可直接用 flv.js 播放。
//
// 重要约定：推流端必须从 FLV Header 开始推送（以字节串 "FLV" 开头）。
// ------------------------------

/*
摄像头推流

ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/ingest/test"
ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv -headers "Content-Type: video/x-flv" http://127.0.0.1:8080/ingest/test
ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv - | curl -X POST -H "Content-Type: video/x-flv" --data-binary @- http://127.0.0.1:8080/ingest/test


ffmpeg -loglevel debug -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" \
  -vcodec libx264 -preset veryfast -tune zerolatency -g 30 \
  -acodec aac -ar 44100 -ac 2 \
  -f flv -headers "Content-Type: video/x-flv\r\n" \
  http://127.0.0.1:8080/ingest/test



ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" \
       -vcodec libx264 -preset veryfast -tune zerolatency \
       -acodec aac -ar 44100 -ac 2 \
       -f flv - | curl -X POST -H "Content-Type: video/x-flv" \
       --data-binary @- http://127.0.0.1:8080/ingest/test

拉流地址：http://127.0.0.1:8080/live/test.flv


*/

// 配置
const (
	RecentWindow   = 3 * time.Second // 最近窗口时长（用于新观众快速起播）
	RecentMaxBytes = 4 << 20         // 最近缓冲最大字节数（4MB 上限）
	WriteTimeout   = 10 * time.Second
	PingInterval   = 10 * time.Second
)

// 一个字节块，带时间戳，便于按窗口裁剪
type chunk struct {
	b  []byte
	ts time.Time
}

// Broadcaster 负责一个流的分发与缓存
// - subscribers: 订阅者集合（每个订阅者是一个 chan []byte）
// - recent: 最近一段时间/字节的环形缓冲
// - header: 记住最新的 FLV Header（便于新订阅者立刻播放）

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
	recent      []chunk
	recentBytes int
	header      []byte
	closing     bool
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan []byte]struct{}),
		recent:      make([]chunk, 0, 256),
	}
}

// Publish 接收推流端写入的原始 FLV 字节
func (b *Broadcaster) Publish(data []byte) {
	if len(data) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		return
	}

	// 识别并保存 FLV Header（只要分片以 "FLV" 开头就视为 header）
	if b.header == nil && len(data) >= 3 && string(data[:3]) == "FLV" {
		b.header = append([]byte(nil), data...)
	}

	// 维护短时缓存（时间窗 + 字节上限）
	now := time.Now()
	b.recent = append(b.recent, chunk{b: append([]byte(nil), data...), ts: now})
	b.recentBytes += len(data)

	// 剪掉过期（时间窗）
	cutoff := now.Add(-RecentWindow)
	i := 0
	for ; i < len(b.recent); i++ {
		if b.recent[i].ts.After(cutoff) {
			break
		}
		b.recentBytes -= len(b.recent[i].b)
	}
	if i > 0 {
		b.recent = append([]chunk(nil), b.recent[i:]...)
	}
	// 若超过字节上限，再从最早处裁剪
	for b.recentBytes > RecentMaxBytes && len(b.recent) > 0 {
		b.recentBytes -= len(b.recent[0].b)
		b.recent = b.recent[1:]
	}

	// 扇出分发（非阻塞：写不进去就丢）
	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
			// 慎重：如果观众读取太慢，我们直接丢帧，避免阻塞整个流
		}
	}
}

// Subscribe 新观众订阅，返回一个只读通道与一个函数用于取消订阅
func (b *Broadcaster) Subscribe() (<-chan []byte, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		ch := make(chan []byte)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan []byte, 128) // 给一定缓冲，抗网络抖动
	b.subscribers[ch] = struct{}{}
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
	}
	return ch, cancel
}

// Snapshot 供新观众快速起播：优先返回 header + 最近窗口数据
func (b *Broadcaster) Snapshot() (header []byte, rec [][]byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.header != nil {
		header = append([]byte(nil), b.header...)
	}
	rec = make([][]byte, len(b.recent))
	for i := range b.recent {
		rec[i] = append([]byte(nil), b.recent[i].b...)
	}
	return
}

// Close 停止该路流（例如推流端断开）
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		return
	}
	b.closing = true
	for ch := range b.subscribers {
		close(ch)
	}
	b.subscribers = map[chan []byte]struct{}{}
	b.recent = nil
	b.recentBytes = 0
}

// StreamHub 管理多路流

type StreamHub struct {
	mu  sync.RWMutex
	hub map[string]*Broadcaster
}

func NewStreamHub() *StreamHub {
	return &StreamHub{hub: map[string]*Broadcaster{}}
}

func (h *StreamHub) Get(stream string) *Broadcaster {
	h.mu.RLock()
	b := h.hub[stream]
	h.mu.RUnlock()
	return b
}

func (h *StreamHub) GetOrCreate(stream string) *Broadcaster {
	h.mu.Lock()
	defer h.mu.Unlock()
	if b, ok := h.hub[stream]; ok {
		return b
	}
	b := NewBroadcaster()
	h.hub[stream] = b
	return b
}

func (h *StreamHub) Remove(stream string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if b, ok := h.hub[stream]; ok {
		b.Close()
		delete(h.hub, stream)
	}
}

var (
	streams  = NewStreamHub()
	upgrader = websocket.Upgrader{
		ReadBufferSize:  64 << 10,
		WriteBufferSize: 64 << 10,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

// ------------------------------
// 推流入口 A：HTTP POST /ingest/:stream
//
//	Header: Content-Type: video/x-flv
//	Body:   原始 FLV 字节（建议按块写入，使用 chunked 传输）
//
// ------------------------------
func handleHTTPIngest(c *gin.Context) {
	stream := c.Param("stream")
	if stream == "" {
		c.String(http.StatusBadRequest, "missing stream key")
		return
	}
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "video/x-flv") {
		c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		return
	}
	b := streams.GetOrCreate(stream)

	fmt.Printf("%s 广播器创建成功，开始推流！！！\n", stream)

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.(http.Flusher).Flush()

	//reader := bufio.NewReader(c.Request.Body)
	buf := make([]byte, 64<<10)
	for {
		//n, err := reader.Read(buf)
		n, err := c.Request.Body.Read(buf)
		if n > 0 {
			b.Publish(buf[:n])
		}
		if err != nil {
			fmt.Printf("handleHTTPIngest.err = %s \n", err)
			if err != io.EOF {
				fmt.Println("读取错误:", err)
			}
			break
		}
	}
	// 推流结束
	// 不立刻 Close 路流，允许观众继续看最近缓存（也可选择立即关闭）
	c.Status(http.StatusNoContent)
}

// ------------------------------
// 推流入口 B：WebSocket /publish/:stream
//
//	BinaryMessage: 原始 FLV 字节块
//	心跳：可选 Ping/Pong
//
// ------------------------------
func handleWSPublish(c *gin.Context) {
	stream := c.Param("stream")
	if stream == "" {
		c.String(http.StatusBadRequest, "missing stream key")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}
	defer conn.Close()

	b := streams.GetOrCreate(stream)
	_ = conn.SetReadDeadline(time.Now().Add(3 * PingInterval))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(3 * PingInterval))
	})

	for {
		typ, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("ws read error:", err)
			}
			return
		}
		if typ != websocket.BinaryMessage {
			continue
		}
		b.Publish(data)
	}
}

// ------------------------------
// 拉流：HTTP-FLV  GET /live/:stream.flv
//
//	响应头：Content-Type: video/x-flv
//	先返回 header + 最近窗口，再进入实时订阅。
//
// ------------------------------
func handleHTTPFLV(c *gin.Context) {
	stream := strings.TrimSuffix(c.Param("stream"), ".flv")
	if stream == "" {
		c.String(http.StatusBadRequest, "missing stream key")
		return
	}
	b := streams.Get(stream)
	if b == nil {
		c.String(http.StatusNotFound, "stream not found")
		return
	}

	w := c.Writer
	c.Header("Content-Type", "video/x-flv")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "streaming unsupported")
		return
	}

	// 先发 header + recent
	head, rec := b.Snapshot()
	if len(head) > 0 {
		_, _ = w.Write(head)
	}
	for _, ch := range rec {
		_, _ = w.Write(ch)
	}
	flusher.Flush()

	// 订阅实时
	realtime, cancel := b.Subscribe()
	defer cancel()

	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()

	notify := w.CloseNotify()
	for {
		select {
		case <-notify:
			return
		case data, ok := <-realtime:
			if !ok {
				return
			}
			//_ = w.SetWriteDeadline(time.Now().Add(WriteTimeout))
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
			if !deadline.Stop() {
				<-deadline.C
			}
			deadline.Reset(30 * time.Second)
		case <-deadline.C:
			// 30s 没有任何数据，断开以避免挂死连接
			return
		}
	}
}

// ------------------------------
// 辅助：一个简单的 flv.js HTML 播放页（便于手工测试）
// 打开 /player/:stream  即可在浏览器中播放 /live/:stream.flv
// ------------------------------
func handlePlayer(c *gin.Context) {
	stream := c.Param("stream")
	html := fmt.Sprintf(`<!doctype html>
<html>
<head>
	<meta charset="utf-8" />
	<title>FLV Player - %s</title>
	<script src="https://cdn.jsdelivr.net/npm/flv.js@latest/dist/flv.min.js"></script>
</head>
<body>
	<h3>Stream: %s</h3>
	<video id="video" controls autoplay muted style="width: 80%%; max-width: 1280px;"></video>
	<script>
	if (flvjs.isSupported()) {
		var videoElement = document.getElementById('video');
		var flvPlayer = flvjs.createPlayer({
			type: 'flv',
			url: location.origin + '/live/%s.flv',
			isLive: true,
			cors: true,
			hasAudio: true,
			hasVideo: true
		});
		flvPlayer.attachMediaElement(videoElement);
		flvPlayer.load();
		flvPlayer.play();
	}
	</script>
</body>
</html>`, stream, stream, stream)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// ------------------------------
// 健康检查 & 列表
// ------------------------------
func handleHealth(c *gin.Context) { c.String(200, "ok") }

func handleList(c *gin.Context) {
	var names []string
	streams.mu.RLock()
	for k := range streams.hub {
		names = append(names, k)
	}
	streams.mu.RUnlock()
	c.JSON(200, gin.H{"streams": names})
}

// ------------------------------
// 中间件 & 工具
// ------------------------------
func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		lat := time.Since(start)
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), lat)
	}
}

// ------------------------------
// main
// ------------------------------
func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(logger(), gin.Recovery())

	r.GET("/healthz", handleHealth)
	r.GET("/streams", handleList)

	// 推流入口
	r.Any("/ingest/:stream", handleHTTPIngest) // HTTP 推流
	r.GET("/publish/:stream", handleWSPublish) // WS 推流（使用 ws:// 连接）
	// 某些代理只转发 WS 到 GET，所以这里用 GET，真实是 WS 升级

	// 拉流入口（网页使用 flv.js 播放）
	r.GET("/live/:stream", func(c *gin.Context) { // 兼容不带后缀
		c.Params = append(c.Params, gin.Param{Key: "stream", Value: c.Param("stream") + ".flv"})
		handleHTTPFLV(c)
	})
	//r.GET("/live2/:stream.flv", handleHTTPFLV)

	// 简易播放页
	r.GET("/player/:stream", handlePlayer)

	addr := ":8080"
	log.Println("server listening on", addr)
	if err := r.Run(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
