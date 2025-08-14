package broker

import "pull2push/core/client"

// Broker 每一个直播地址都持有一个 Broker 对象，用于保存当前这个直播的信息
type Broker interface {

	// AddLiveClient 新增客户端
	AddLiveClient(clientId string, liveClient client.LiveClient)

	// RemoveLiveClient 移除客户端
	RemoveLiveClient(clientId string)

	// FindLiveClient 查询 LiveClient
	FindLiveClient(clientId string) (client.LiveClient, error)

	// ListenStatus 监听当前直播的必要状态
	ListenStatus()

	// PullLoop 持续去直播原地址拉流/数据
	PullLoop()

	// Broadcast 原地址拉取到数据之后广播给客户端
	Broadcast2LiveClient(data []byte)

	// UpdateSourceURL 支持切换直播原地址
	UpdateSourceURL(newSourceURL string)
}
