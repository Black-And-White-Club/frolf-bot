package roundsubscribers

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/handlers/mocks"
	"go.uber.org/mock/gomock"
)

func TestRoundEventSubscribers_SubscribeToParticipantManagementEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := mocks.NewMockHandlers(ctrl) // Create the mock handler
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				// Expect calls to Subscribe without calling the handlers directly
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.ParticipantResponse, gomock.Any()).
					Return(nil)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.ScoreUpdated, gomock.Any()).
					Return(nil)

				// No need to call the handlers within DoAndReturn
			},
			wantErr: false,
		},
		{
			name: "SubscribeParticipantResponseError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.ParticipantResponse, gomock.Any()).
					Return(errors.New("subscribe error"))
			},
			wantErr: true,
		},
		{
			name: "SubscribeScoreUpdatedError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.ParticipantResponse, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.ScoreUpdated, gomock.Any()).
					Return(errors.New("subscribe error"))
			},
			wantErr: true,
		},
		// Add more test cases for other scenarios...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			s := &RoundEventSubscribers{
				eventBus: mockEventBus,
				logger:   logger,
				handlers: mockHandlers,
			}

			if err := s.SubscribeToParticipantManagementEvents(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("RoundEventSubscribers.SubscribeToParticipantManagementEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
