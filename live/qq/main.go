package main

//
//import (
//	"sync"
//)
//
//// Broadcaster 广播器接口
//type Broadcaster interface {
//
//	// AddBroker 添加 Broker
//	AddBroker(brokerKey string, b Broker)
//
//	// RemoveBroker 移除 Broker
//	RemoveBroker(brokerKey string)
//}
//
//// FLVBroadcaster 广播器
//type FLVBroadcaster struct {
//	mutex sync.Mutex
//
//	//  负责协调所有的拉流链接 FLVStreamBroker 直接的工作
//	flvBrokerMap map[string]*Broker // key为直播房间号
//
//}
//
//func NewFLVBroadcaster() *FLVBroadcaster {
//	return &FLVBroadcaster{
//		flvBrokerMap: make(map[string]*Broker),
//	}
//}
//
//func (fb *FLVBroadcaster) AddBroker(brokerKey string, b Broker) {
//}
//
//func (fb *FLVBroadcaster) RemoveBroker(brokerKey string) {
//}
//
//// Broker 每一个直播地址都持有一个 Broker 对象，用于保存当前这个直播的信息
//type Broker interface {
//}
//
//// FLVStreamBroker 每个 直播地址 用一个 Broker 管理，里面管理了多个当前直播链接的客户端
//type FLVStreamBroker struct {
//	BrokerKey   string // 直播房间的唯一编号
//	UpstreamURL string // 直播房间的上游拉流地址
//
//	clientMutex sync.Mutex // 客户端的异步操作控制器
//}
//
//func NewBroker(brokerKey, upstreamURL string) *FLVStreamBroker {
//	return &FLVStreamBroker{
//		BrokerKey:   brokerKey,
//		UpstreamURL: upstreamURL,
//	}
//}
//
//func main() {
//
//	brokerKey := "test1"
//	flvUpstreamURL := "http://192.168.203.182:8080/live/livestream.flv"
//
//	var flvBroadcast *FLVBroadcaster = NewFLVBroadcaster()
//
//	var flvStreamBroker *FLVStreamBroker = NewBroker(brokerKey, flvUpstreamURL)
//	flvBroadcast.AddBroker(brokerKey, flvStreamBroker) // 无法将 'flvStreamBroker' (类型 *FLVStreamBroker) 用作类型 *Broker
//
//}
