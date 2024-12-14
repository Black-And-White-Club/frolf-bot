package commands

// Command is the interface that all commands must implement.
type Command interface {
	CommandName() string
}
