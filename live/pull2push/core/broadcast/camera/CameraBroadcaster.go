package camera

import (
	"errors"
	"fmt"
	"pull2push/core/broker"
	"sync"
)

// ====================== CameraBroadcaster ======================

// CameraBroadcaster.go 广播器
type CameraBroadcaster struct {
	mutex sync.Mutex

	//  负责协调所有的拉流链接 FLVStreamBroker 直接的工作
	brokerMap map[string]broker.Broker // key为直播房间号

	mu       sync.Mutex
	clients  map[chan []byte]struct{}
	cache    [][]byte
	maxCache int
}

func NewCameraBroadcaster() *CameraBroadcaster {
	return &CameraBroadcaster{
		brokerMap: make(map[string]broker.Broker),
	}
}

func (cb *CameraBroadcaster) AddBroker(brokerKey string, b broker.Broker) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.brokerMap[brokerKey] = b
}

func (cb *CameraBroadcaster) RemoveBroker(brokerKey string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if _, ok := cb.brokerMap[brokerKey]; !ok {
		return
	}

	delete(cb.brokerMap, brokerKey)
}

// FindBroker 查询 Broker
func (cb *CameraBroadcaster) FindBroker(brokerKey string) (broker.Broker, error) {
	if val, ok := cb.brokerMap[brokerKey]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("未找到 %s 对应的Broker", brokerKey))

}
