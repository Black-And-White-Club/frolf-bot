package roundutil

import (
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func descPtr(s string) *roundtypes.Description {
	d := roundtypes.Description(s)
	return &d
}

func TestRoundValidator_ValidateBaseRoundPayload(t *testing.T) {
	v := NewRoundValidator()

	t.Run("valid payload", func(t *testing.T) {
		now := sharedtypes.StartTime(time.Now())
		payload := roundtypes.BaseRoundPayload{
			Title:       "Test Round",
			StartTime:   &now,
			Location:    "Test Location",
			Description: "Test Description",
		}
		errs := v.ValidateBaseRoundPayload(payload)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		payload := roundtypes.BaseRoundPayload{}
		errs := v.ValidateBaseRoundPayload(payload)
		expectedErrors := 3
		if len(errs) != expectedErrors {
			t.Errorf("expected %d errors, got %d: %v", expectedErrors, len(errs), errs)
		}
	})
}

func TestRoundValidator_ValidateRoundInput(t *testing.T) {
	v := NewRoundValidator()

	t.Run("valid input", func(t *testing.T) {
		input := roundtypes.CreateRoundInput{
			Title:       "Test Round",
			StartTime:   "2023-10-27T10:00:00Z",
			Location:    "Test Location",
			Description: descPtr("Test Description"),
		}
		errs := v.ValidateRoundInput(input)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		input := roundtypes.CreateRoundInput{}
		errs := v.ValidateRoundInput(input)
		expectedErrors := 3
		if len(errs) != expectedErrors {
			t.Errorf("expected %d errors, got %d: %v", expectedErrors, len(errs), errs)
		}
	})
}
