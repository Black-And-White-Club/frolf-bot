package roundsubscribers

import (
	"context"
	"errors"
	"os"
	"testing"

	"log/slog"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/handlers/mocks"
	"go.uber.org/mock/gomock"
)

func TestRoundEventSubscribers_SubscribeToRoundManagementEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := mocks.NewMockHandlers(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundCreated, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundUpdated, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundDeleted, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "SubscribeCreateRoundError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundCreated, gomock.Any()).
					Return(errors.New("subscribe error"))
			},
			wantErr: true,
		},
		{
			name: "SubscribeUpdateRoundError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundCreated, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundUpdated, gomock.Any()).
					Return(errors.New("subscribe error"))
			},
			wantErr: true,
		},
		{
			name: "SubscribeDeleteRoundError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundCreated, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundUpdated, gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundDeleted, gomock.Any()).
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
			if err := s.SubscribeToRoundManagementEvents(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("RoundEventSubscribers.SubscribeToRoundManagementEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundEventSubscribers_SubscribeToRoundStartedEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := mocks.NewMockHandlers(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundStarted, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "SubscribeRoundStartedError",
			setup: func() {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.RoundStreamName, events.RoundStarted, gomock.Any()).
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
			if err := s.SubscribeToRoundStartedEvents(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("RoundEventSubscribers.SubscribeToRoundStartedEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
