package test10_enent_bus

import (
	"fmt"
	"time"
)

type Subscriber interface {
	Handle(event Event)
}

type MsgSubscriber struct {
	SubName string
}

func (m *MsgSubscriber) Handle(event Event) {
	fmt.Printf("【%s】开始处理：\n", m.SubName)
	for i := range 5 {
		fmt.Printf("\t【%s】正在处理 %d - %s ... \n", m.SubName, i, event.Name())
		time.Sleep(time.Second * 1)
	}
	fmt.Printf("【%s】处理完成！\n", m.SubName)

}
