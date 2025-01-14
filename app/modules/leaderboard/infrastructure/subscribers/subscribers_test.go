package leaderboardsubscribers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	handlermocks "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/handlers/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardSubscribers(t *testing.T) {
	tests := []struct {
		name     string
		eventBus *eventbusmock.MockEventBus
		handlers *handlermocks.MockHandlers
		logger   *slog.Logger
	}{
		{
			name:     "Create New Leaderboard Subscribers",
			eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
			handlers: handlermocks.NewMockHandlers(gomock.NewController(t)),
			logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLeaderboardSubscribers(tt.eventBus, tt.handlers, tt.logger)
			if got == nil {
				t.Fatal("NewLeaderboardSubscribers() returned nil") // Use t.Fatal to stop execution
			}
			if got.EventBus != tt.eventBus {
				t.Error("EventBus not set correctly")
			}
			if got.Handlers != tt.handlers {
				t.Error("Handlers not set correctly")
			}
			if got.logger != tt.logger {
				t.Error("Logger not set correctly")
			}
		})
	}
}

func TestLeaderboardSubscribers_SubscribeToLeaderboardEvents(t *testing.T) {
	tests := []struct {
		name       string
		eventBus   *eventbusmock.MockEventBus
		handlers   *handlermocks.MockHandlers
		logger     *slog.Logger
		wantErr    bool
		setupMocks func(handlers *handlermocks.MockHandlers, eventBus *eventbusmock.MockEventBus)
	}{
		{
			name:     "Successful Subscription",
			eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
			handlers: handlermocks.NewMockHandlers(gomock.NewController(t)),
			logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			wantErr:  false,
			setupMocks: func(handlers *handlermocks.MockHandlers, eventBus *eventbusmock.MockEventBus) {
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.LeaderboardUpdatedSubject, gomock.Any()).Return(nil)
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.TagAssignedSubject, gomock.Any()).Return(nil)
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.TagSwapRequestedSubject, gomock.Any()).Return(nil)
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.GetLeaderboardRequestSubject, gomock.Any()).Return(nil)
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.GetTagByDiscordIDRequestSubject, gomock.Any()).Return(nil)
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.CheckTagAvailabilityRequestSubject, gomock.Any()).Return(nil)
			},
		},
		{
			name:     "Subscription Error",
			eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
			handlers: handlermocks.NewMockHandlers(gomock.NewController(t)),
			logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			wantErr:  true,
			setupMocks: func(handlers *handlermocks.MockHandlers, eventBus *eventbusmock.MockEventBus) {
				eventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.LeaderboardUpdatedSubject, gomock.Any()).Return(errors.New("subscription error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardSubscribers{
				EventBus: tt.eventBus,
				Handlers: tt.handlers,
				logger:   tt.logger,
			}

			if tt.setupMocks != nil {
				tt.setupMocks(tt.handlers, tt.eventBus)
			}

			if err := s.SubscribeToLeaderboardEvents(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardSubscribers.SubscribeToLeaderboardEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
