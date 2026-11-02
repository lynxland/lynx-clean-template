package events

import "github.com/lynxland/lynx-clean-template/pkg/pubsub"

const (
	EventAccountCreated       pubsub.EventName = "account:created"
	EventAccountAuthenticated pubsub.EventName = "account:authenticated"
)

const (
	TopicUserEvents    pubsub.TopicName = "user_events"
	TopicHello         pubsub.TopicName = "hello"
	TopicWebhooks      pubsub.TopicName = "webhooks"
	TopicNotifications pubsub.TopicName = "notifications"
)
