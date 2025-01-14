package usersubscribers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	handlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewSubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := handlers.NewMockHandlers(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Creates a new UserSubscribers instance", func(t *testing.T) {
		got := NewSubscribers(mockEventBus, mockHandlers, logger)
		if got == nil {
			t.Errorf("NewSubscribers() returned nil")
		}
	})
}

func TestUserSubscribers_SubscribeToUserEvents(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(mockEventBus *eventbusmock.MockEventBus, mockHandlers *handlers.MockHandlers)
		wantErr bool
	}{
		{
			name: "Successful subscription",
			setup: func(mockEventBus *eventbusmock.MockEventBus, mockHandlers *handlers.MockHandlers) {
				// Expect Subscribe calls for each event type with correct stream names
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupRequest, gomock.Any()).
					Return(nil).Times(1)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserRoleUpdateRequest, gomock.Any()).
					Return(nil).Times(1)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupResponse, gomock.Any()).
					Return(nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "Error subscribing to UserSignupRequest",
			setup: func(mockEventBus *eventbusmock.MockEventBus, mockHandlers *handlers.MockHandlers) {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupRequest, gomock.Any()).
					Return(fmt.Errorf("subscription error")).Times(1)
			},
			wantErr: true,
		},
		{
			name: "Error subscribing to UserRoleUpdateRequest",
			setup: func(mockEventBus *eventbusmock.MockEventBus, mockHandlers *handlers.MockHandlers) {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupRequest, gomock.Any()).
					Return(nil).Times(1)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserRoleUpdateRequest, gomock.Any()).
					Return(fmt.Errorf("subscription error")).Times(1)
			},
			wantErr: true,
		},
		{
			name: "Error subscribing to UserSignupResponse",
			setup: func(mockEventBus *eventbusmock.MockEventBus, mockHandlers *handlers.MockHandlers) {
				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupRequest, gomock.Any()).
					Return(nil).Times(1)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserRoleUpdateRequest, gomock.Any()).
					Return(nil).Times(1)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.UserStreamName, events.UserSignupResponse, gomock.Any()).
					Return(fmt.Errorf("subscription error")).Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Initialize mocks
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockHandlers := handlers.NewMockHandlers(ctrl)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			// Call the setup function to configure mocks
			tt.setup(mockEventBus, mockHandlers)

			// Initialize UserSubscribers instance
			s := &UserSubscribers{
				eventBus: mockEventBus,
				handlers: mockHandlers,
				logger:   logger,
			}

			// Invoke the SubscribeToUserEvents method
			err := s.SubscribeToUserEvents(context.Background(), mockEventBus, mockHandlers, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubscribeToUserEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
