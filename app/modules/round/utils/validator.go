package roundutil

import (
	"fmt"
	"log/slog"

	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
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

	// Add more validation rules here as needed...

	return errs
}
