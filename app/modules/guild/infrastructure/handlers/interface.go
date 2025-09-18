package guildhandlers

import "github.com/ThreeDotsLabs/watermill/message"

// Handlers defines the contract for guild event handlers.
type Handlers interface {
	HandleCreateGuildConfig(msg *message.Message) ([]*message.Message, error)
	HandleRetrieveGuildConfig(msg *message.Message) ([]*message.Message, error)
	HandleUpdateGuildConfig(msg *message.Message) ([]*message.Message, error)
	HandleDeleteGuildConfig(msg *message.Message) ([]*message.Message, error)
	HandleGuildSetup(msg *message.Message) ([]*message.Message, error)
}
