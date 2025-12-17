package roundservice

import "fmt"

// ImportError is a structured error used internally by import helpers.
type ImportError struct {
	Code    string
	Message string
	Err     error
}

func (e *ImportError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ImportError) Unwrap() error { return e.Err }
