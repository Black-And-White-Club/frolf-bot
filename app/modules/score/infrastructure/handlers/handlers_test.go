package scorehandlers

import (
	"testing"

	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockScoreService := scoreservice.NewMockService(ctrl)
				handlers := NewScoreHandlers(mockScoreService, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				scoreHandlers := handlers.(*ScoreHandlers)

				if scoreHandlers.service != mockScoreService {
					t.Errorf("service not correctly assigned")
				}
				if scoreHandlers.helpers != nil {
					t.Errorf("helpers should be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				handlers := NewScoreHandlers(nil, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				scoreHandlers, ok := handlers.(*ScoreHandlers)
				if !ok {
					t.Fatalf("handlers is not of type *ScoreHandlers")
				}
				if scoreHandlers.service != nil {
					t.Errorf("service should be nil")
				}
				if scoreHandlers.helpers != nil {
					t.Errorf("helpers should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
