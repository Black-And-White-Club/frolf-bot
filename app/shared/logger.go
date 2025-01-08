package shared

// LoggerAdapter interface for logging.
type LoggerAdapter interface {
	Error(msg string, err error, fields LogFields)
	Info(msg string, fields LogFields)
	Debug(msg string, fields LogFields)
	Trace(msg string, fields LogFields)
	With(fields LogFields) LoggerAdapter
}

// LogFields represents a map of log fields.
type LogFields map[string]interface{}
