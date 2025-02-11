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
	if input.StartTime == nil {
		errs = append(errs, fmt.Errorf("start time cannot be empty"))
	}

	// Example: Validate that the date is in the future
	if input.StartTime != nil && input.StartTime.Before(time.Now()) {
		errs = append(errs, fmt.Errorf("start date must be in the future"))
	}

	// Example: Validate that the end time is after the start time
	if input.EndTime != nil && input.StartTime != nil && (input.EndTime.Before(*input.StartTime) || input.EndTime.Equal(*input.StartTime)) {
		errs = append(errs, fmt.Errorf("end time must be after start time"))
	}
	return errs
}
