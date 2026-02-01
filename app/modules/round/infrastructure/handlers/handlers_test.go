package roundhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
)

func TestNewRoundHandlers(t *testing.T) {
	// Define test cases
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				// Create fake dependencies
				fakeRoundService := NewFakeService()
				fakeUserService := NewFakeUserService()
				fakeHelpers := &FakeHelpers{}
				logger := loggerfrolfbot.NoOpLogger

				// Call the function being tested
				handlers := NewRoundHandlers(fakeRoundService, fakeUserService, logger, fakeHelpers)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				// Access roundHandlers directly from the RoundHandlers struct
				roundHandlers := handlers.(*RoundHandlers)

				// Check that all dependencies were correctly assigned
				if roundHandlers.service != fakeRoundService {
					t.Errorf("service not correctly assigned")
				}
				if roundHandlers.userService != fakeUserService {
					t.Errorf("userService not correctly assigned")
				}
				if roundHandlers.helpers != fakeHelpers {
					t.Errorf("helpers not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				// Call with nil dependencies
				handlers := NewRoundHandlers(nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				// Check nil fields
				if roundHandlers, ok := handlers.(*RoundHandlers); ok {
					if roundHandlers.service != nil {
						t.Errorf("service should be nil")
					}
					if roundHandlers.userService != nil {
						t.Errorf("userService should be nil")
					}
					if roundHandlers.helpers != nil {
						t.Errorf("helpers should be nil")
					}
				} else {
					t.Errorf("handlers is not of type *RoundHandlers")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
