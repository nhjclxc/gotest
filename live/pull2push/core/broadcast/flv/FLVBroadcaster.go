package broadcast

import (
	"errors"
	"fmt"
	"pull2push/core/broker"
	"sync"
)

// ====================== Broadcaster ======================

// FLVBroadcaster 广播器
type FLVBroadcaster struct {
	mutex sync.Mutex

	//  负责协调所有的拉流链接 FLVStreamBroker 直接的工作
	brokerMap map[string]broker.Broker // key为直播房间号

}

func NewFLVBroadcaster() *FLVBroadcaster {
	return &FLVBroadcaster{
		brokerMap: make(map[string]broker.Broker),
	}
}

func (fb *FLVBroadcaster) AddBroker(brokerKey string, b broker.Broker) {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	fb.brokerMap[brokerKey] = b
}

func (fb *FLVBroadcaster) RemoveBroker(brokerKey string) {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	if _, ok := fb.brokerMap[brokerKey]; !ok {
		return
	}

	delete(fb.brokerMap, brokerKey)
}

// FindBroker 查询 Broker
func (fb *FLVBroadcaster) FindBroker(brokerKey string) (broker.Broker, error) {
	if val, ok := fb.brokerMap[brokerKey]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("未找到 %s 对应的Broker", brokerKey))

}
