package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 每个 upstream stream 用一个 Broker 管理：拉流 goroutine + 客户端集合
type StreamBroker struct {
	upstreamURL string

	clientsMu sync.Mutex
	clients   map[*wsClient]struct{}

	// 控制
	stopCh chan struct{}
	once   sync.Once
}

type wsClient struct {
	conn     *websocket.Conn
	sendCh   chan []byte
	closeSig chan struct{}
}

var (
	brokersMu sync.Mutex
	brokers   = map[string]*StreamBroker{} // key: sha1(upstreamURL)
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 静态页面
	r.Static("/static", "./static")

	// WebSocket endpoint: ?url=<upstreamFlvUrl>
	r.GET("/ws", func(c *gin.Context) {
		upstream := c.Query("url")
		if upstream == "" {
			c.String(http.StatusBadRequest, "missing url")
			return
		}
		// validate upstream URL
		u, err := url.Parse(upstream)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			c.String(http.StatusBadRequest, "invalid upstream url")
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("ws upgrade:", err)
			return
		}

		b := getOrCreateBroker(upstream)
		client := newWSClient(conn)
		b.addClient(client)

		// wait until client closes
		client.run()
	})

	// health
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// ==== Broker methods ====

func newBroker(upstream string) *StreamBroker {
	b := &StreamBroker{
		upstreamURL: upstream,
		clients:     make(map[*wsClient]struct{}),
		stopCh:      make(chan struct{}),
	}
	// start pulling loop
	go b.pullLoop()
	return b
}

func getOrCreateBroker(upstream string) *StreamBroker {
	key := sha1hex(upstream)
	brokersMu.Lock()
	defer brokersMu.Unlock()
	if b, ok := brokers[key]; ok {
		return b
	}
	b := newBroker(upstream)
	brokers[key] = b
	return b
}

func (b *StreamBroker) addClient(c *wsClient) {
	b.clientsMu.Lock()
	b.clients[c] = struct{}{}
	b.clientsMu.Unlock()

	// when client closes, remove it
	go func() {
		<-c.closeSig
		b.removeClient(c)
	}()
}

func (b *StreamBroker) removeClient(c *wsClient) {
	b.clientsMu.Lock()
	delete(b.clients, c)
	remaining := len(b.clients)
	b.clientsMu.Unlock()

	_ = c.conn.Close()
	close(c.sendCh)

	// 如果没有客户端并且想释放 broker，可关闭 stopCh 让 pullLoop 停止（本示例保留 broker，防止频繁断开上游）
	if remaining == 0 {
		// optionally stop pulling after idle timeout. For simplicity we keep running.
	}
}

func (b *StreamBroker) broadcast(data []byte) {
	b.clientsMu.Lock()
	// 复制 client list to avoid holding lock during send
	clients := make([]*wsClient, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.clientsMu.Unlock()

	// send non-blocking (drop if client's channel full)
	for _, c := range clients {
		select {
		case c.sendCh <- data:
		default:
			// 客户端慢，丢包并继续
		}
	}
}

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
				DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("upstream connect error:", err)
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

		// read and broadcast in chunks
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				// copy bytes to avoid race
				cp := make([]byte, n)
				copy(cp, buf[:n])
				b.broadcast(cp)
			}
			if err != nil {
				if err == io.EOF {
					log.Println("upstream EOF, reconnecting")
				} else {
					log.Println("upstream read error:", err)
				}
				resp.Body.Close()
				break
			}
		}

		// 如果 stop 信号被触发，可以退出（此实现未触发 stop）
		select {
		case <-b.stopCh:
			return
		default:
		}

		// small backoff before reconnect
		time.Sleep(500 * time.Millisecond)
	}
}

func sha1hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// ==== wsClient ====

func newWSClient(conn *websocket.Conn) *wsClient {
	// binary messages only
	conn.SetReadLimit(1 << 20)
	return &wsClient{
		conn:     conn,
		sendCh:   make(chan []byte, 512),
		closeSig: make(chan struct{}),
	}
}

func (c *wsClient) run() {
	// writer
	go func() {
		// set write deadline per message
		for b := range c.sendCh {
			if err := c.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
				// write error -> close
				log.Println("ws write error:", err)
				break
			}
		}
		// ensure connection closed
		_ = c.conn.Close()
		// signal close
		select {
		case <-c.closeSig:
		default:
			close(c.closeSig)
		}
	}()

	// read pump (we don't expect meaningful messages; just detect close/ping)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// read to detect client close; we ignore payloads
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// close
			break
		}
	}
	// close sendCh to stop writer
	select {
	case <-c.closeSig:
	default:
		close(c.closeSig)
	}
}
