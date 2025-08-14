package camera

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"pull2push/core/client"
	"sync"
)

/*
1️⃣ 核心思路
目标是实现：
	摄像头或其他来源推流 → HTTP POST 到 Go 服务
	Go 服务缓存最近若干视频数据 → 用于新客户端秒开
	多客户端拉流 → 每个客户端可以同时观看实时流
	保证推流不卡死 → 慢客户端不阻塞推流

要解决的关键问题：
	流是长连接，不能一次性读完，否则 ffmpeg 推送就会阻塞
	多客户端同时读取，要避免互相阻塞
	新观众加入时希望立即看到最近几秒画面


*/

// CameraBroker 每个 直播地址 用一个 Broker 管理，里面管理了多个当前直播链接的客户端
type CameraBroker struct {
	// 直播数据相关
	BrokerKey    string   // 直播房间的唯一编号
	cache        [][]byte // 内存缓存最近的若干个 FLV 包，方便新客户端秒开
	cacheRWMutex sync.RWMutex
	maxCache     int          // 缓存最大长度，防止内存无限增长
	ctx          *gin.Context // 推流原请求对象

	// 状态控制相关
	BrokerCloseSig chan struct{} // 控制当前这个直播是否被关闭
	once           sync.Once

	// 客户端相关
	clientMutex    sync.Mutex                   // 客户端的异步操作控制器
	clientMap      map[string]client.LiveClient // map[clientId]LiveClient 存储这个broker里面所有的客户端
	ClientCloseSig chan string                  // 客户端关闭信号，当客户端主动关闭通知时，该信道被触发，输出的字符串为关闭的客户端编号clientId

}

func NewCameraBroker(ctx *gin.Context, maxCache int) *CameraBroker {
	if maxCache == 0 {
		maxCache = 150
	}
	streamName := ctx.Param("stream")
	cb := CameraBroker{
		BrokerKey:      streamName,
		ctx:            ctx,
		cache:          make([][]byte, 0),
		clientMap:      make(map[string]client.LiveClient),
		BrokerCloseSig: make(chan struct{}),
		ClientCloseSig: make(chan string),
		maxCache:       maxCache,
	}

	// 开始不断接收推流
	go cb.PullLoop()

	// 开启必要的状态监听
	go cb.ListenStatus()

	log.Println("开始推流:", streamName)

	return &cb
}

// AddLiveClient 添加客户端
func (cb *CameraBroker) AddLiveClient(clientId string, client client.LiveClient) {
	cb.clientMutex.Lock()
	cb.clientMap[clientId] = client
	cb.clientMutex.Unlock()

	// 先发送缓存数据
	for _, data := range cb.cache {
		client.Broadcast(data)
	}

}

// RemoveLiveClient 移除客户端
func (cb *CameraBroker) RemoveLiveClient(clientId string) {
	cb.clientMutex.Lock()
	defer cb.clientMutex.Unlock()
	liveClient := cb.clientMap[clientId]
	if liveClient == nil {
		return
	}
	delete(cb.clientMap, clientId)

	//// 如果没有客户端并且想释放 broker，可关闭 stopCh 让 PullLoop 停止（本示例保留 broker，防止频繁断开上游）
	//if remaining == 0 {
	//	// optionally stop pulling after idle timeout. For simplicity we keep running.
	//}
}

// FindLiveClient 查询 LiveClient
func (cb *CameraBroker) FindLiveClient(clientId string) (client.LiveClient, error) {
	if val, ok := cb.clientMap[clientId]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("未找到 %s 对应的 LiveClient", clientId))
}

// UpdateSourceURL 支持切换直播原地址
func (cb *CameraBroker) UpdateSourceURL(newSourceURL string) {}

// PullLoop 持续去直播原地址拉流/数据
func (cb *CameraBroker) PullLoop() {

	buf := make([]byte, 32*1024)
	for {
		n, err := cb.ctx.Request.Body.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			// 广播给所有的客户端
			cb.Broadcast2LiveClient(data)
		}
		if err != nil {
			if err != io.EOF {
				log.Println("读取错误:", err)
			}
			break
		}
	}
	log.Println("推流结束:", cb.BrokerKey)
}

// ListenStatus 监听当前直播的必要状态
func (cb *CameraBroker) ListenStatus() {
	for {
		select {
		case clientId := <-cb.ClientCloseSig:
			// 监听客户端离开消息
			cb.RemoveLiveClient(clientId)
			fmt.Printf("CameraBroker.ListenStatus.RemoveLiveClient.clientId %s successful.", clientId)
		case <-cb.BrokerCloseSig:
			// 直播被关闭
			close(cb.BrokerCloseSig)
		}

	}
}

// Broadcast2LiveClient 原地址拉取到数据之后广播给客户端
func (cb *CameraBroker) Broadcast2LiveClient(data []byte) {

	cb.cacheRWMutex.Lock()
	// 添加到缓存
	cb.cache = append(cb.cache, data)
	if len(cb.cache) > cb.maxCache {
		cb.cache = cb.cache[1:]
	}
	cb.cacheRWMutex.Unlock()

	// 广播给所有客户端
	for _, liveClient := range cb.clientMap {
		go liveClient.Broadcast(data)
	}
}
