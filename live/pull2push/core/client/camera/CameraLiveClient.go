package flv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ====================== CameraLiveClient ======================

// CameraLiveClient 每一个前端页面有持有一个客户端对象
type CameraLiveClient struct {
	BrokerKey string      // 这个客户端的直播房间的唯一编号
	ClientId  string      // 这个客户端的id
	DataCh    chan []byte // 这个客户端的一个只写通道

	// http连接相关
	ctx                 *gin.Context
	httpCloseSig        <-chan struct{} // 当这个请求被客户端主动被关闭时触发
	httpRequestCloseSig <-chan struct{} // 当这个请求被客户端主动被关闭时触发

	// 父级 Broker相关的内容
	clientCloseSig chan<- string   // broker通过该信道监听客户端离线 【仅发送】
	brokerCloseSig <-chan struct{} // broker被关闭时，同时通知客户端关闭 【仅接收】
}

func NewCameraLiveClient(c *gin.Context, brokerKey, clientId string, clientCloseSig chan<- string, brokerCloseSig <-chan struct{}) (*CameraLiveClient, error) {

	clc := CameraLiveClient{
		BrokerKey:           brokerKey,
		ClientId:            clientId,
		ctx:                 c,
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

	flusher, ok := clc.ctx.Writer.(http.Flusher)
	if !ok {
		clc.ctx.String(http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	for data := range clc.DataCh {
		_, err := clc.ctx.Writer.Write(data)
		if err != nil {
			break
		}
		flusher.Flush()
	}
}
