package client

type LiveClient interface {

	// Broadcast 服务端给客户端推流
	// data 当前接收到直播上游的新数据
	Broadcast(data []byte)

	// Listen 客户端监听器
	Listen()
}
