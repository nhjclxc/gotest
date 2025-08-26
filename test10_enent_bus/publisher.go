package test10_enent_bus

type Publish interface {
	Publish(event Event)
}

type MsgPublisher struct {
	bus     EventBus
	PubName string
}

func NewMsgPublisher(bus EventBus) *MsgPublisher {
	return &MsgPublisher{
		bus: bus,
	}
}

func (p *MsgPublisher) Publish(event Event) {
	p.bus.Publish(event)
}
