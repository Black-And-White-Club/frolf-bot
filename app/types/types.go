// app/types/topic.go

package types

import "github.com/ThreeDotsLabs/watermill/message"

type Topic string

type Handler struct {
	Topic         Topic
	Handler       message.HandlerFunc
	ResponseTopic string
}
