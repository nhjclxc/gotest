package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// StreamBroker 每个 upstream stream 用一个 Broker 管理：拉流 goroutine + 客户端集合
type StreamBroker struct {
	brokerKey   string
	upstreamURL string

	flvHeader []byte

	clientsMu sync.Mutex
	clients   map[*wsClient]struct{}

	// 控制
	stopCh chan struct{}
	once   sync.Once
}

type wsClient struct {
	brokerKey      string
	clientId       string
	httpRequest    *http.Request
	responseWriter *gin.ResponseWriter
	sendCh         chan []byte
	closeSig       chan struct{}
}

var (
	brokerMutex sync.Mutex
	brokerPool  = map[string]*StreamBroker{} // key: sha1(upstreamURL)
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// ==== Broker methods ====

func newBroker(brokerKey string, upstream string) *StreamBroker {
	b := &StreamBroker{
		brokerKey:   brokerKey,
		upstreamURL: upstream,
		clients:     make(map[*wsClient]struct{}),
		stopCh:      make(chan struct{}),
	}
	// start pulling loop
	go b.pullLoop()
	fmt.Printf("\n newBroker = %#v \n", b)
	return b
}

func getOrCreateBroker(brokerKey string, upstream string) *StreamBroker {
	if brokerKey == "" {
		brokerKey = sha1hex(upstream)
	}
	brokerMutex.Lock()
	defer brokerMutex.Unlock()
	if b, ok := brokerPool[brokerKey]; ok {
		return b
	}
	b := newBroker(brokerKey, upstream)
	brokerPool[brokerKey] = b
	return b
}

func sha1hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
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
		fmt.Printf("send non-blocking %#v \n\n", c.sendCh)
		select {
		case c.sendCh <- data:
			fmt.Println("发送数据成功")
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

// ==== wsClient ====

func newClient(brokerKey string, clientId string, httpRequest *http.Request, responseWriter *gin.ResponseWriter) *wsClient {

	return &wsClient{
		brokerKey:      brokerKey,
		clientId:       clientId,
		httpRequest:    httpRequest,
		responseWriter: responseWriter,
		sendCh:         make(chan []byte, 512),
		closeSig:       make(chan struct{}),
	}
}

func (c *wsClient) run(broker *StreamBroker) {
	defer func() {
		//brokerMutex.Lock()
		//delete(brokerPool, c.brokerKey)
		//brokerMutex.Unlock()
		//close(c.sendCh)
	}()

	flusher, ok := (*c.responseWriter).(http.Flusher)
	if !ok {
		fmt.Println("streaming unsupported")
		return
	}
	// 先写flv header给客户端
	_, err := (*c.responseWriter).Write(broker.flvHeader)
	if err != nil {
		log.Println("write header error:", err)
		return
	}
	flusher.Flush()

	// 持续写数据
	for {
		fmt.Printf("持续写数据 %#v \n\n", c.sendCh)
		fmt.Printf("持续写数据 %#v \n\n", <-c.sendCh)

		select {
		case data, ok := <-c.sendCh:
			if !ok {
				return
			}
			_, err := (*c.responseWriter).Write(data)
			if err != nil {
				log.Println("write data error:", err)
				return
			}
			flusher.Flush()
			//case <-c.httpRequest.Context().Done():
			//	return
		}
	}

}

/*
使用ffmpeg推流：ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream

拉流
● RTMP (by VLC): rtmp://192.168.203.182/live/livestream
● H5(HTTP-FLV): http://192.168.203.182:8080/live/livestream.flv
● H5(HLS): http://192.168.203.182:8080/live/livestream.m3u8

*/

// ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream
func main() {

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	upstreamURL := "http://192.168.203.182:8080/live/livestream.flv"
	getOrCreateBroker("test-room", upstreamURL)

	// health
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	r.GET("/live/:brokerKey/:clientId", func(c *gin.Context) {

		c.Writer.Header().Set("Content-Type", "video/x-flv")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.WriteHeader(http.StatusOK)

		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")

		client := newClient(brokerKey, clientId, c.Request, &c.Writer)
		borker := brokerPool[brokerKey]
		borker.addClient(client)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			fmt.Println("streaming unsupported")
			return
		}
		// 先写flv header给客户端
		_, err := c.Writer.Write(borker.flvHeader)
		if err != nil {
			log.Println("write header error:", err)
			return
		}
		flusher.Flush()

		// 持续写数据
		for {
			fmt.Printf("持续写数据 %#v \n\n", client.sendCh)
			fmt.Printf("持续写数据 %#v \n\n", <-client.sendCh)

			select {
			case data, ok := <-client.sendCh:
				if !ok {
					return
				}
				_, err := c.Writer.Write(data)
				if err != nil {
					log.Println("write data error:", err)
					return
				}
				flusher.Flush()
				//case <-c.httpRequest.Context().Done():
				//	return
			}
		}

		// wait until client closes
		//go client.run(borker)
	})

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
