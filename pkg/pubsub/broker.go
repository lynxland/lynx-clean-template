package pubsub

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/lynxland/lynx/eventbus"
	"github.com/lynxland/x/log"
)

// Publisher publishes domain events to a logical topic.
type Publisher interface {
	Publish(ctx context.Context, topicName TopicName, eventName EventName, data any, opts ...eventbus.PublishOption) error
}

// Broker wraps the Lynx event bus and uses CloudEvents as the standard event format.
type Broker struct {
	bus eventbus.Bus
	id  string
}

// NewPubSub creates a Broker backed by the given event bus.
func NewPubSub(bus eventbus.Bus) *Broker {
	id := "lynx-broker"
	if bus != nil {
		id = bus.Name()
	}
	return &Broker{bus: bus, id: id}
}

// ID returns the broker identifier used as the CloudEvents source.
func (b *Broker) ID() string {
	return b.id
}

func (b *Broker) Publish(ctx context.Context, topicName TopicName, eventName EventName, data any, opts ...eventbus.PublishOption) error {
	log.InfoContext(ctx, "publishing event", "topicName", topicName, "eventName", eventName)
	payload, err := NewEventBytes(b.id, eventName.String(), data)
	if err != nil {
		return err
	}
	// v1.17.0 起 Publish 返回消息 ID（与消费端 event.id 对账用）；Broker
	// 的门面签名保持 error，ID 当前无消费方，不外露。
	_, err = b.bus.Publish(ctx, topicName.String(), payload, opts...)
	return err
}

// HandlerFunc handles a decoded CloudEvent.
type HandlerFunc func(ctx context.Context, e *cloudevents.Event) error

func (b *Broker) Subscribe(topicName TopicName, eventName EventName, handlerName string, h HandlerFunc, opts ...eventbus.SubscribeOption) error {
	_ = handlerName // eventbus v1.5.2 no longer requires a unique handler name
	rawHandler := func(ctx context.Context, e *eventbus.RawEvent) error {
		event := cloudevents.NewEvent()
		if err := event.UnmarshalJSON(e.Payload); err != nil {
			return err
		}
		if event.Type() == eventName.String() {
			return h(ctx, &event)
		}
		return nil
	}
	return b.bus.Subscribe(context.WithoutCancel(context.Background()), topicName.String(), rawHandler, opts...)
}

// Handler is implemented by event handlers that bind to the router.
type Handler interface {
	EventName() EventName
	TopicName() TopicName
	HandlerName() string
	HandlerFunc() HandlerFunc
}

// HandlerOptions allows a handler to provide extra subscribe options.
type HandlerOptions interface {
	Options() []eventbus.SubscribeOption
}

// EventName identifies an event type.
type EventName string

func (e EventName) String() string {
	return string(e)
}

// TopicName identifies a logical topic.
type TopicName string

func (t TopicName) String() string {
	return string(t)
}
