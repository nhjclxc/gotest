package flv

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"pull2push/core/broker/flv"
	"runtime/debug"
)

// ====================== FLVLiveClient ======================

// FLVLiveClient 每一个前端页面有持有一个客户端对象
type FLVLiveClient struct {
	BrokerKey string        // 这个客户端的直播房间的唯一编号
	ClientId  string        // 这个客户端的id
	DataCh    chan []byte   // 这个客户端的一个只写通道
	CloseSig  chan struct{} // broker被关闭时，同时通知客户端关闭

	// http连接相关
	httpRequest         *http.Request
	responseWriter      io.Writer
	flusher             http.Flusher
	httpCloseSig        <-chan struct{} // 当这个请求被客户端主动被关闭时触发
	httpRequestCloseSig <-chan struct{} // 当这个请求被客户端主动被关闭时触发

	flvStreamBroker *flv.FLVStreamBroker
}

func NewFLVLiveClient(c *gin.Context, brokerKey, clientId string, dataCh1 chan []byte, gOPCache *flv.GOPCache, flvStreamBroker *flv.FLVStreamBroker) (*FLVLiveClient, error) {
	// 创建一个带缓冲的双向通道，缓冲大小根据需求调节
	dataCh := make(chan []byte, 4096)

	// gin.ResponseWriter 是接口，不能用指针
	var writer io.Writer = c.Writer

	// 断言出 http.Flusher 接口，方便主动刷新数据
	flusher, ok := writer.(http.Flusher)
	if !ok {
		// 不支持刷新，可能无法做到流式推送
		log.Println("ResponseWriter does not support Flusher interface")
	}

	hc := FLVLiveClient{
		BrokerKey:           brokerKey,
		ClientId:            clientId,
		DataCh:              dataCh,
		CloseSig:            make(chan struct{}),
		httpRequest:         c.Request,
		responseWriter:      writer,
		flusher:             flusher,
		httpCloseSig:        c.Done(),
		httpRequestCloseSig: c.Request.Context().Done(),
		flvStreamBroker:     flvStreamBroker,
	}

	fmt.Println("客户端连接成功 ClientId = ", clientId)

	//// 确保头部信息已解析
	//flvStreamBroker.HeaderMutex.RLock()
	//headerParsed := flvStreamBroker.HeaderParsed
	//headerBytes := flvStreamBroker.HeaderBytes
	//flvStreamBroker.HeaderMutex.RUnlock()
	//
	//if !headerParsed {
	//	return nil, fmt.Errorf("FLV头部尚未解析完成，请先调用Start()")
	//}
	//
	//// 一次性写入头部数据
	//if _, err := writer.Write(headerBytes); err != nil {
	//	return nil, fmt.Errorf("写入FLV头部失败: %v", err)
	//}

	// 持续监控是否一些控制通道的消息
	go hc.Listen()

	return &hc, nil
}

// Listen 客户端监听器
func (hc *FLVLiveClient) Listen() {

	for {
		select {
		case data, ok := <-hc.DataCh:
			//fmt.Printf("接收到数据 len = %d \n", len(data))
			if ok {

				if hc.responseWriter == nil || hc.flusher == nil {
					log.Println("responseWriter 或 flusher 已经无效，退出")
					return
				}

				_, err := hc.responseWriter.Write(data)
				if err != nil {
					// 写出错，关闭连接
					return
				}
				hc.flusher.Flush()
			}
		case <-hc.httpCloseSig:
			hc.CloseSig <- struct{}{}
			// 收到关闭信号，退出循环
			fmt.Println("hc.httpCloseSig 收到客户端关闭信号，退出循环 ", hc.ClientId)

			// when client closes, remove it
			hc.flvStreamBroker.RemoveLiveClient(hc.ClientId)

			close(hc.CloseSig)

			return
		case <-hc.httpRequestCloseSig:
			hc.CloseSig <- struct{}{}
			// 收到关闭信号，退出循环
			fmt.Println("<-hc.httpRequestCloseSig 收到客户端关闭信号，退出循环 ", hc.ClientId)

			// when client closes, remove it
			hc.flvStreamBroker.RemoveLiveClient(hc.ClientId)

			close(hc.CloseSig)

			return
		}
	}
}

func (hc *FLVLiveClient) Broadcast(data []byte) {

	defer func() {
		if err := recover(); err != nil {
			// 这里写自定义日志处理  打印堆栈
			log.Printf("panic recovered: %v: %v\n stack trace:\n%s", err, debug.Stack())
		}
	}()

	if hc.responseWriter == nil || hc.flusher == nil {
		log.Println("responseWriter 或 flusher 已经无效，退出")
		return
	}

	_, err := hc.responseWriter.Write(data)
	if err != nil {
		// 写出错，关闭连接
		return
	}
	hc.flusher.Flush()

}
