package roundutil

import (
	"fmt"
	"log/slog"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
)

// RoundValidator defines the interface for round validation.
type RoundValidator interface {
	ValidateRoundInput(input roundtypes.CreateRoundInput) []error
}

// RoundValidationError defines a struct for collecting validation errors.
type RoundValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RoundValidatorImpl is the concrete implementation of the RoundValidator interface.
type RoundValidatorImpl struct {
	logger *slog.Logger
}

// NewRoundValidator creates a new instance of RoundValidatorImpl.
func NewRoundValidator() RoundValidator {
	return &RoundValidatorImpl{
		logger: slog.Default(), // Use your preferred logger
	}
}

// ValidateRoundInput validates the input for creating a new round.
func (v *RoundValidatorImpl) ValidateRoundInput(input roundtypes.CreateRoundInput) []error {
	var errs []error

	if input.Title == "" {
		errs = append(errs, fmt.Errorf("title cannot be empty"))
	}

	if input.StartTime.Date == "" || input.StartTime.Time == "" {
		errs = append(errs, fmt.Errorf("date/time input cannot be empty"))
	}

	// Example: Validate that the date is in the future
	if input.StartTime.Date < time.Now().Format("2006-01-02") {
		errs = append(errs, fmt.Errorf("start date must be in the future"))
	}

	// Example: Validate that the end time is after the start time
	if input.EndTime.Date < input.StartTime.Date || (input.EndTime.Date == input.StartTime.Date && input.EndTime.Time <= input.StartTime.Time) {
		errs = append(errs, fmt.Errorf("end time must be after start time"))
	}

	// Add more validation rules here as needed...

	return errs
}
