package broadcast

import (
	"errors"
	"fmt"
	"pull2push/core/broker"
	"sync"
)

// ====================== HLSBroadcaster ======================

// HLSBroadcaster 广播器
type HLSBroadcaster struct {
	mutex sync.Mutex

	//  负责协调所有的 HLSM3U8Broker 直接的工作
	brokerMap map[string]broker.Broker // key为直播房间号

}

func NewBroadcaster() *HLSBroadcaster {
	return &HLSBroadcaster{
		brokerMap: make(map[string]broker.Broker),
	}
}

func (hb *HLSBroadcaster) AddBroker(brokerKey string, b broker.Broker) {
	hb.mutex.Lock()
	defer hb.mutex.Unlock()

	hb.brokerMap[brokerKey] = b
}

func (hb *HLSBroadcaster) RemoveBroker(brokerKey string) {
	hb.mutex.Lock()
	defer hb.mutex.Unlock()

	if _, ok := hb.brokerMap[brokerKey]; !ok {
		return
	}

	delete(hb.brokerMap, brokerKey)
}

// FindBroker 查询 Broker
func (fb *HLSBroadcaster) FindBroker(brokerKey string) (broker.Broker, error) {
	if val, ok := fb.brokerMap[brokerKey]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("未找到 %s 对应的Broker", brokerKey))
}
