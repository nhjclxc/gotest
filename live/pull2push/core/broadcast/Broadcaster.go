package broadcast

import "pull2push/core/broker"

// ====================== Broadcaster ======================

// Broadcaster 广播器接口
type Broadcaster interface {

	// AddBroker 添加 Broker
	AddBroker(brokerKey string, b broker.Broker)

	// RemoveBroker 移除 Broker
	RemoveBroker(brokerKey string)

	// FindBroker 查询 Broker
	FindBroker(brokerKey string) (broker.Broker, error)
}
