package main

import (
	"bufio"
	"context"
	"encoding/binary"
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

// ========== 配置 ==========
const (
	bufferDuration     = 3 * time.Second // 缓存时长（3s）
	readChunkSize      = 32 * 1024       // 读取缓冲
	clientChanBuf      = 128             // 每客户端 channel 缓冲
	restartDelay       = 2 * time.Second // ffmpeg 重启等待
	ffmpegStartTimeout = 5 * time.Second
	wsWriteTimeout     = 10 * time.Second
)

// ========== FLV 基本解析结构 ==========
type flvTag struct {
	timestamp uint32 // ms
	raw       []byte // 完整的 tag bytes：11 header + data + 4 prevTagSize
	isKey     bool   // 是否关键帧（IDR 或 FrameType==1）
}

type flvHeader struct {
	raw []byte // FLV header + first previousTagSize（通常是 9 + 4 = 13 bytes）
}

// ========== 基于时间窗口的 tag 缓存（关键帧对齐） ==========
type tagBuffer struct {
	mu     sync.Mutex
	tags   []flvTag
	size   int
	header flvHeader
	// 记录是否已见到第一个关键帧（只有在见到关键帧后，才开始缓存并允许下发）
	seenKey bool
}

func (tb *tagBuffer) setHeader(h flvHeader) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.header = h
}

func (tb *tagBuffer) append(t flvTag) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 如果尚未见到关键帧：只有当 t.isKey==true 时，才开始缓存并将 seenKey=true，
	// 并把该关键帧加入缓存（即缓存从关键帧开始）
	if !tb.seenKey {
		if !t.isKey {
			// 丢弃非关键帧，直到碰到关键帧
			return
		}
		tb.seenKey = true
	}

	tb.tags = append(tb.tags, t)
	tb.size += len(t.raw)
	tb.trimLocked()
}

// 返回按序拼接的缓存字节（包含 header），供新客户端初始化使用
func (tb *tagBuffer) getInitBytes() []byte {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if !tb.seenKey || len(tb.tags) == 0 {
		// 没有可用缓存
		if len(tb.header.raw) > 0 {
			return tb.header.raw
		}
		return nil
	}
	total := len(tb.header.raw)
	for _, t := range tb.tags {
		total += len(t.raw)
	}
	out := make([]byte, 0, total)
	if len(tb.header.raw) > 0 {
		out = append(out, tb.header.raw...)
	}
	for _, t := range tb.tags {
		out = append(out, t.raw...)
	}
	return out
}

// 保持 buffer 中只保留最近 bufferDuration 的 tags。
// 这里我们使用系统时间和 tag.timestamp（ms）比较来裁剪
func (tb *tagBuffer) trimLocked() {
	if len(tb.tags) == 0 {
		return
	}
	cut := uint32(time.Now().Add(-bufferDuration).UnixNano() / 1e6) // ms
	i := 0
	for ; i < len(tb.tags); i++ {
		if tb.tags[i].timestamp >= cut {
			break
		}
		tb.size -= len(tb.tags[i].raw)
	}
	if i > 0 {
		tb.tags = tb.tags[i:]
	}
	// 如果我们因为长时间无关键帧导致 tags 为空，则重置 seenKey = false（等待下一关键帧）
	if len(tb.tags) == 0 {
		tb.seenKey = false
	}
}

// ========== Broadcaster（解析 FLV 并广播 tags） ==========
type Broadcaster struct {
	src    string
	ctx    context.Context
	cancel context.CancelFunc
	buf    *tagBuffer

	clients map[int]chan []byte
	mu      sync.Mutex
	nextID  int
	wg      sync.WaitGroup
	started sync.Once
}

func NewBroadcaster(src string) *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broadcaster{
		src:     src,
		ctx:     ctx,
		cancel:  cancel,
		buf:     &tagBuffer{},
		clients: make(map[int]chan []byte),
	}
}

func (b *Broadcaster) Start() {
	b.started.Do(func() {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.loopFFmpeg()
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

func (b *Broadcaster) RegisterClient() (int, <-chan []byte, []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++
	ch := make(chan []byte, clientChanBuf)
	b.clients[id] = ch
	init := b.buf.getInitBytes()
	return id, ch, init
}

func (b *Broadcaster) UnregisterClient(id int) {
	b.mu.Lock()
	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
	b.mu.Unlock()
}

func (b *Broadcaster) broadcastTagBytes(p []byte) {
	// forward to all clients (non阻塞)
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.clients {
		select {
		case ch <- append([]byte(nil), p...):
		default:
			// 慢客户端直接丢包
			log.Printf("client %d slow, drop packet", id)
		}
	}
}

// ========== FLV 解析与 ffmpeg 读流逻辑 ==========
func (b *Broadcaster) loopFFmpeg() {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}
		log.Printf("starting ffmpeg pulling %s", b.src)
		cmdCtx, cmdCancel := context.WithCancel(b.ctx)
		cmd := exec.CommandContext(cmdCtx, "ffmpeg", "-i", b.src, "-c", "copy", "-f", "flv", "-")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("StdoutPipe err: %v", err)
			cmdCancel()
			time.Sleep(restartDelay)
			continue
		}
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			log.Printf("ffmpeg start err: %v", err)
			cmdCancel()
			time.Sleep(restartDelay)
			continue
		}

		// log stderr
		go func() {
			s := bufio.NewScanner(stderr)
			for s.Scan() {
				log.Printf("[ffmpeg] %s", s.Text())
			}
		}()

		reader := bufio.NewReaderSize(stdout, readChunkSize)
		// 先读取并保存 FLV header（9 bytes） + first prevTagSize（4 bytes）
		headerBytes := make([]byte, 9)
		if _, err := io.ReadFull(reader, headerBytes); err != nil {
			log.Printf("read flv header err: %v", err)
			_ = cmd.Process.Kill()
			cmdCancel()
			_, _ = cmd.Process.Wait()
			time.Sleep(restartDelay)
			continue
		}
		// read first previousTagSize (4 bytes)
		prevTag := make([]byte, 4)
		if _, err := io.ReadFull(reader, prevTag); err != nil {
			log.Printf("read flv prevTag err: %v", err)
			_ = cmd.Process.Kill()
			cmdCancel()
			_, _ = cmd.Process.Wait()
			time.Sleep(restartDelay)
			continue
		}
		hdr := flvHeader{raw: append(headerBytes, prevTag...)}
		b.buf.setHeader(hdr)

		// 启动循环读取 tag
		readErr := make(chan error, 1)
		go func() {
			for {
				// read tag header 11 bytes
				head := make([]byte, 11)
				if _, err := io.ReadFull(reader, head); err != nil {
					readErr <- err
					return
				}
				tagType := head[0]
				dataSize := uint32(head[1])<<16 | uint32(head[2])<<8 | uint32(head[3])
				timestamp := uint32(head[7])<<24 | uint32(head[4])<<16 | uint32(head[5])<<8 | uint32(head[6])
				// read data
				data := make([]byte, dataSize)
				if dataSize > 0 {
					if _, err := io.ReadFull(reader, data); err != nil {
						readErr <- err
						return
					}
				}
				// read PreviousTagSize
				prev := make([]byte, 4)
				if _, err := io.ReadFull(reader, prev); err != nil {
					readErr <- err
					return
				}

				// compose raw tag bytes = header(11) + data + prevTagSize(4)
				raw := make([]byte, 0, 11+int(dataSize)+4)
				raw = append(raw, head...)
				if dataSize > 0 {
					raw = append(raw, data...)
				}
				raw = append(raw, prev...)

				// 判定是否关键帧（默认 false）
				isKey := false
				if tagType == 9 { // video
					// video tag: first data byte: FrameType(4) | CodecID(4)
					if len(data) > 0 {
						frameByte := data[0]
						frameType := (frameByte & 0xF0) >> 4
						codecID := frameByte & 0x0F
						// 如果是 AVC（7），则需要查看 avc packet type & nalus
						if codecID == 7 {
							// avcPacketType is data[1] if exists
							if len(data) >= 2 {
								avcPacketType := data[1]
								if avcPacketType == 1 {
									// NALU： data[5..] 中含有 NALUs，每个 NALU 前有 4字节 length
									// FLV video data for AVC: 0:frame+codec,1:avcPacketType,2:compositionTime(3)
									offset := 5
									for offset+4 <= len(data) {
										naluLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
										offset += 4
										if offset+naluLen > len(data) {
											break
										}
										if naluLen > 0 {
											nalUnitHeader := data[offset]
											nalType := nalUnitHeader & 0x1F
											// nal_type 5 = IDR (关键帧)
											if nalType == 5 {
												isKey = true
												break
											}
										}
										offset += naluLen
									}
								} else if avcPacketType == 0 {
									// AVC sequence header (SPS/PPS) —— not a frame, skip
								}
							}
						} else {
							// 非 AVC codec: 仍可用 frameType == 1 表示 keyframe（FLV 定义：1:keyframe）
							if frameType == 1 {
								isKey = true
							}
						}
					}
				}

				// 创建 flvTag 并 append 到 buffer（关键帧逻辑在 buffer 内）
				tag := flvTag{
					timestamp: timestamp,
					raw:       raw,
					isKey:     isKey,
				}
				b.buf.append(tag)

				// 同时广播该 raw bytes 给所有在线客户端
				b.broadcastTagBytes(raw)
			}
		}()

		// 等待读 goroutine 报错或 ctx cancel
		select {
		case err := <-readErr:
			if err != nil && err != io.EOF {
				log.Printf("readErr: %v", err)
			} else {
				log.Printf("ffmpeg stdout closed")
			}
		case <-b.ctx.Done():
			// 取消
		}

		// 杀死子进程并等待退出
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		cmdCancel()

		// 若整体上下文未取消，则短等后重启
		select {
		case <-b.ctx.Done():
			return
		default:
			time.Sleep(restartDelay)
			// 继续循环以重连 ffmpeg
		}
	}
}

// ========== WebSocket handler ==========
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(b *Broadcaster) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("upgrade err: %v", err)
			return
		}
		defer ws.Close()

		id, ch, init := b.RegisterClient()
		defer b.UnregisterClient(id)
		log.Printf("ws client %d connected", id)

		// 先发 header + buffer 初始化数据（如果有）
		if len(init) > 0 {
			ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ws.WriteMessage(websocket.BinaryMessage, init); err != nil {
				log.Printf("write init err: %v", err)
				return
			}
		}

		// 启动读协程以探测断开
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_, _, err := ws.ReadMessage()
				if err != nil {
					return
				}
			}
		}()

		for {
			select {
			case <-done:
				log.Printf("ws client %d read loop done", id)
				return
			case p, ok := <-ch:
				if !ok {
					log.Printf("client %d channel closed", id)
					return
				}
				ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
					log.Printf("write to client %d err: %v", id, err)
					return
				}
			case <-time.After(30 * time.Second):
				// 心跳 ping
				ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("ping err for client %d: %v", id, err)
					return
				}
			}
		}
	}
}

// ========== main ==========
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
		c.String(200, fmt.Sprintf("WebSocket-FLV (keyframe aligned) proxy running. Pulling: %s\nWS endpoint: /ws/flv", src))
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("gin run error: %v", err)
	}
}
