package test10_enent_bus

type Event interface {
	Name() string // 返回事件名称，用于路由
}

type MsgEvent struct {
	EventName string
	Data      string
}

func (e MsgEvent) Name() string {
	return e.EventName
}
