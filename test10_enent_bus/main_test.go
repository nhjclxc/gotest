package test10_enent_bus

import "testing"

func Test1(t *testing.T) {

	exitCh := make(chan bool, 1)

	e := MsgEvent{
		EventName: "msg1",
		Data:      "哈哈哈你好呀",
	}

	//  event_bus
	var meb EventBus = &MsgEventBus{
		subscribers: make(map[string][]Subscriber),
	}

	// sub
	ms1 := MsgSubscriber{SubName: "sub11"}
	meb.Subscribe("msg1", &ms1)

	ms2 := MsgSubscriber{SubName: "sub11111"}
	meb.Subscribe("msg1", &ms2)

	p := NewMsgPublisher(meb)
	p.Publish(e)

	meb.Unsubscribe("msg1", &ms1)

	p.Publish(e)

	<-exitCh

}
