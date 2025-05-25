package roundutil

import (
	"log/slog"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// RoundValidator defines the interface for round validation.
type RoundValidator interface {
	ValidateBaseRoundPayload(input roundtypes.BaseRoundPayload) []string
	ValidateRoundInput(input roundtypes.CreateRoundInput) []string // Update to accept CreateRoundInput
}

// RoundValidatorImpl is the concrete implementation of the RoundValidator interface.
type RoundValidatorImpl struct {
	logger *slog.Logger
}

// NewRoundValidator creates a new instance of RoundValidatorImpl.
func NewRoundValidator() RoundValidator {
	return &RoundValidatorImpl{
		logger: slog.Default(),
	}
}

// ValidateBaseRoundPayload validates the base round payload.
func (v *RoundValidatorImpl) ValidateBaseRoundPayload(input roundtypes.BaseRoundPayload) []string {
	var errs []string

	if input.Title == "" {
		errs = append(errs, "title cannot be empty")
	}

	if input.StartTime == nil { // This check may need to be removed or adjusted
		errs = append(errs, "start time cannot be empty")
	}

	if input.Location == nil || *input.Location == "" {
		errs = append(errs, "location cannot be empty")
	}

	if input.Description == nil || *input.Description == "" {
		errs = append(errs, "description cannot be empty")
	}

	return errs
}

// ValidateRoundInput validates the input for creating a new round.
func (v *RoundValidatorImpl) ValidateRoundInput(input roundtypes.CreateRoundInput) []string {
	var errs []string

	if input.Title == "" {
		errs = append(errs, "title cannot be empty")
	}

	if input.StartTime == "" {
		errs = append(errs, "start time cannot be empty")
	}

	if input.Location == nil || *input.Location == "" {
		errs = append(errs, "location cannot be empty")
	}

	if input.Description == nil || *input.Description == "" {
		errs = append(errs, "description cannot be empty")
	}

	return errs
}
