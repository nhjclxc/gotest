package camera

import (
	"errors"
	"fmt"
	"log"
	"pull2push/core/broker"
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
	BrokerKey string // 直播房间的唯一编号
	//cache        [][]byte // 内存缓存最近的若干个 FLV 包，方便新客户端秒开  缓存最近一个 GOP（关键帧 + 后续帧）
	gop [][]byte // 内存缓存最近的若干个 FLV 包，方便新客户端秒开缓存最近一个 GOP（关键帧 + 后续帧）

	// 状态控制相关
	BrokerCloseSig chan broker.BROKER_CLOSE_TYPE // 控制当前这个直播是否被关闭
	once           sync.Once

	// 客户端相关
	clientMutex    sync.Mutex                   // 客户端的异步操作控制器
	clientMap      map[string]client.LiveClient // map[clientId]LiveClient 存储这个broker里面所有的客户端
	ClientCloseSig chan string                  // 客户端关闭信号，当客户端主动关闭通知时，该信道被触发，输出的字符串为关闭的客户端编号clientId

}

func NewCameraBroker(brokerKey string, maxCache int) *CameraBroker {
	if maxCache == 0 {
		maxCache = 150
	}
	cb := CameraBroker{
		BrokerKey: brokerKey,
		//cache:          make([][]byte, 0),
		clientMap:      make(map[string]client.LiveClient),
		BrokerCloseSig: make(chan broker.BROKER_CLOSE_TYPE),
		ClientCloseSig: make(chan string),
		gop:            make([][]byte, 0),
	}

	// 开启必要的状态监听
	go cb.ListenStatus()

	log.Println("开始推流:", brokerKey)

	return &cb
}

// AddLiveClient 添加客户端
func (cb *CameraBroker) AddLiveClient(clientId string, client client.LiveClient) {
	cb.clientMutex.Lock()
	cb.clientMap[clientId] = client
	cb.clientMutex.Unlock()

	// 先把缓存的 GOP 发送给新客户端
	for _, pkt := range cb.gop {
		client.GetDataChan() <- pkt
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

// PullLoop 持续去直播原地址拉流/数据
func (cb *CameraBroker) PullLoop(bo broker.BrokerOptional) {

	data := make([]byte, 4096)
	for {
		n, err := bo.GinContext.Request.Body.Read(data)
		if err != nil {
			fmt.Println("推流断开:", err)
			break
		}
		packet := make([]byte, n)
		copy(packet, data[:n])
		cb.Broadcast2LiveClient(packet)
	}

}

// Broadcast2LiveClient 原地址拉取到数据之后广播给客户端
func (cb *CameraBroker) Broadcast2LiveClient(data []byte) {
	cb.clientMutex.Lock()
	clientMap := cb.clientMap
	cb.clientMutex.Unlock()

	// 简化：这里假设每个 packet 都是关键帧或帧数据
	// 实际项目中需要解析 FLV tag 判断关键帧
	isKeyFrame := data[0]&0x10 != 0
	// 如果是关键帧，重置 GOP 缓存
	if isKeyFrame {
		cb.gop = cb.gop[:0]
	}
	// 缓存
	cb.gop = append(cb.gop, data)

	// 广播给所有客户端
	for _, c := range clientMap {
		c.GetDataChan() <- data
	}

}
