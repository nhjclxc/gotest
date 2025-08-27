package test10_enent_bus

import "fmt"

type EventBus interface {
	Subscribe(eventName string, subscriber Subscriber)   // 订阅事件
	Unsubscribe(eventName string, subscriber Subscriber) // 取消订阅
	Publish(event Event)                                 // 发布事件
	AsyncPublish(event Event)
}

type MsgEventBus struct {
	subscribers map[string][]Subscriber
}

func (m *MsgEventBus) Subscribe(eventName string, subscriber Subscriber) {
	m.subscribers[eventName] = append(m.subscribers[eventName], subscriber)
}

func (m *MsgEventBus) Unsubscribe(eventName string, subscriber Subscriber) {
	list := m.subscribers[eventName]
	if len(list) > 0 {
		for i, s := range list {
			if s == subscriber {
				m.subscribers[eventName] = append(list[0:i], list[i+1:]...)
			}
		}
	}
}

func (m *MsgEventBus) Publish(event Event) {
	m.doPublish(event, false)
}

func (m *MsgEventBus) AsyncPublish(event Event) {
	m.doPublish(event, true)
}

func (m *MsgEventBus) doPublish(event Event, async bool) {
	fmt.Println("消息投递开始。")
	for _, subscriber := range m.subscribers[event.Name()] {
		if async {
			go subscriber.Handle(event)
		} else {
			subscriber.Handle(event)
		}
	}
	fmt.Println("消息投递结束。")
}
