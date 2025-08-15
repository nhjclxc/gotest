package flv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	cameraBroadcast "pull2push/core/broadcast/camera"
	"pull2push/core/broker"
	cameraBroker "pull2push/core/broker/camera"
)

// ====================== CameraLiveClient ======================

// CameraLiveClient 每一个前端页面有持有一个客户端对象
type CameraLiveClient struct {
	BrokerKey string      // 这个客户端的直播房间的唯一编号
	ClientId  string      // 这个客户端的id
	dataCh    chan []byte // 这个客户端的一个只写通道

	// http连接相关
	httpCloseSig        <-chan struct{} // 当这个请求被客户端主动被关闭时触发
	httpRequestCloseSig <-chan struct{} // 当这个请求被客户端主动被关闭时触发

	// 父级 Broker相关的内容
	clientCloseSig chan<- string                   // broker通过该信道监听客户端离线 【仅发送】
	brokerCloseSig <-chan broker.BROKER_CLOSE_TYPE // broker被关闭时，同时通知客户端关闭 【仅接收】
}

func NewCameraLiveClient(c *gin.Context, brokerKey, clientId string, clientCloseSig chan<- string, brokerCloseSig <-chan broker.BROKER_CLOSE_TYPE) (*CameraLiveClient, error) {

	clc := CameraLiveClient{
		BrokerKey:           brokerKey,
		ClientId:            clientId,
		dataCh:              make(chan []byte, 1024),
		httpCloseSig:        c.Done(),
		httpRequestCloseSig: c.Request.Context().Done(),
		clientCloseSig:      clientCloseSig,
		brokerCloseSig:      brokerCloseSig,
	}

	fmt.Println("HLS 客户端连接成功 ClientId = ", clientId)

	// 开启状态监听
	go clc.Listen()

	return &clc, nil
}

// GetDataChan 获取当前客户端的写通道
func (clc *CameraLiveClient) GetDataChan() chan []byte {
	return clc.dataCh
}

func (clc *CameraLiveClient) Listen() {

	for {
		select {
		case <-clc.httpCloseSig:
			//// 收到关闭信号，退出循环
			//fmt.Println("clc.httpCloseSig 收到客户端关闭信号，退出循环 ", clc.ClientId)
			//
			//// when client closes, remove it
			//clc.clientCloseSig <- clc.ClientId

			return
		case <-clc.httpRequestCloseSig:
			//// 收到关闭信号，退出循环
			//fmt.Println("<-clc.httpRequestCloseSig 收到客户端关闭信号，退出循环 ", clc.ClientId)
			//
			//// when client closes, remove it
			//clc.clientCloseSig <- clc.ClientId

			return
		case <-clc.brokerCloseSig:
			return
		}
	}

}

func (clc *CameraLiveClient) Broadcast(data []byte) {

}

// ExecutePush ==================== HTTP ====================
// ExecutePush 处理摄像头推上来的流数据
func ExecutePush(cameraBroadcastPool *cameraBroadcast.CameraBroadcaster) func(c *gin.Context) {
	return func(c *gin.Context) {
		//if !strings.HasPrefix(c.GetHeader("Content-Type"), "video/x-flv") {
		//	c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		//	return
		//}
		brokerKey := c.Param("brokerKey")

		findBroker, err := cameraBroadcastPool.FindBroker(brokerKey)
		if err != nil {
			fmt.Printf("未找到对应的广播器 %s \n", brokerKey)
			return
		}

		// 开始不断接收推流
		findBroker.PullLoop(broker.BrokerOptional{GinContext: c})

		cameraBroadcastPool.RemoveBroker(brokerKey)

	}
}

// ExecutePull 处理每一个链接上来的客户端的推流
func ExecutePull(cameraBroadcastPool *cameraBroadcast.CameraBroadcaster) func(c *gin.Context) {
	return func(c *gin.Context) {

		c.Header("Content-Type", "video/x-flv")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)

		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")
		findBroker, err := cameraBroadcastPool.FindBroker(brokerKey)
		if err != nil {
			fmt.Printf("未找到对应的广播器 %s \n", brokerKey)
			return
		}

		cameraBrokerTemp, _ := findBroker.(*cameraBroker.CameraBroker)
		client, err := NewCameraLiveClient(c, brokerKey, clientId, cameraBrokerTemp.ClientCloseSig, cameraBrokerTemp.BrokerCloseSig)
		if err != nil {
			fmt.Println("NewCameraLiveClient 创建失败：", err)
			return
		}
		fmt.Println("NewCameraLiveClient 创建成功：clientId = ", clientId)
		cameraBrokerTemp.AddLiveClient(clientId, client)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError, "Streaming unsupported")
			return
		}

		for {
			select {
			case pkt, ok := <-client.GetDataChan():
				if !ok {
					return
				}
				_, err := c.Writer.Write(pkt)
				if err != nil {
					return
				}
				flusher.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}
