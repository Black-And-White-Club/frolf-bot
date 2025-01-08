package usersubscribers

import (
	"context"
	"errors"
	"reflect"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	handlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestNewSubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockHandlers := handlers.NewMockHandlers(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)

	t.Run("Creates a new UserSubscribers instance", func(t *testing.T) {
		want := &UserSubscribers{
			eventBus: mockEventBus,
			handlers: mockHandlers,
			logger:   mockLogger,
		}
		got := NewSubscribers(mockEventBus, mockHandlers, mockLogger)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("NewSubscribers() = %v, want %v", got, want)
		}
	})
}

func TestUserSubscribers_SubscribeToUserEvents(t *testing.T) {
	type fields struct {
		eventBus *eventbusmocks.MockEventBus
		handlers *handlers.MockHandlers
		logger   *testutils.MockLoggerAdapter
		args     struct {
			ctx context.Context
		}
	}
	tests := []struct {
		name    string
		fields  fields
		setup   func(f fields)
		wantErr bool
	}{
		{
			name: "Successful subscription",
			fields: fields{
				eventBus: eventbusmocks.NewMockEventBus(gomock.NewController(t)),  // New controller for each test
				handlers: handlers.NewMockHandlers(gomock.NewController(t)),       // New controller for each test
				logger:   testutils.NewMockLoggerAdapter(gomock.NewController(t)), // New controller for each test
				args: struct {
					ctx context.Context
				}{
					ctx: context.Background(),
				},
			},
			setup: func(f fields) {
				f.eventBus.EXPECT().
					Subscribe(f.args.ctx, userevents.UserSignupRequest.String(), gomock.Any()).
					Return(nil). // Correct Return
					Times(1)
				f.eventBus.EXPECT().
					Subscribe(f.args.ctx, userevents.UserRoleUpdateRequest.String(), gomock.Any()).
					Return(nil). // Correct Return
					Times(1)
				f.logger.EXPECT().Info("Subscribing to event", gomock.Any()).AnyTimes()
				f.logger.EXPECT().Info("Successfully subscribed to event", gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Error subscribing to UserSignupRequest",
			fields: fields{
				eventBus: eventbusmocks.NewMockEventBus(gomock.NewController(t)),  // New controller for each test
				handlers: handlers.NewMockHandlers(gomock.NewController(t)),       // New controller for each test
				logger:   testutils.NewMockLoggerAdapter(gomock.NewController(t)), // New controller for each test
				args: struct {
					ctx context.Context
				}{
					ctx: context.Background(),
				},
			},
			setup: func(f fields) {
				f.eventBus.EXPECT().
					Subscribe(f.args.ctx, userevents.UserSignupRequest.String(), gomock.Any()).
					Return(errors.New("subscription error")). // Correct Return
					Times(1)
			},
			wantErr: true,
		},
		{
			name: "Error subscribing to UserRoleUpdateRequest",
			fields: fields{
				eventBus: eventbusmocks.NewMockEventBus(gomock.NewController(t)),  // New controller for each test
				handlers: handlers.NewMockHandlers(gomock.NewController(t)),       // New controller for each test
				logger:   testutils.NewMockLoggerAdapter(gomock.NewController(t)), // New controller for each test
				args: struct {
					ctx context.Context
				}{
					ctx: context.Background(),
				},
			},
			setup: func(f fields) {
				f.eventBus.EXPECT().
					Subscribe(f.args.ctx, userevents.UserSignupRequest.String(), gomock.Any()).
					Return(nil).
					Times(1)
				f.eventBus.EXPECT().
					Subscribe(f.args.ctx, userevents.UserRoleUpdateRequest.String(), gomock.Any()).
					Return(errors.New("subscription error")). // Correct Return
					Times(1)
				f.logger.EXPECT().Info("Subscribing to event", gomock.Any()).AnyTimes()
				f.logger.EXPECT().Error("Failed to subscribe to event", gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new mock controllers *inside* each test run
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			tt.fields.eventBus = eventbusmocks.NewMockEventBus(ctrl)
			tt.fields.handlers = handlers.NewMockHandlers(ctrl)
			tt.fields.logger = testutils.NewMockLoggerAdapter(ctrl)

			s := &UserSubscribers{
				eventBus: tt.fields.eventBus,
				handlers: tt.fields.handlers,
				logger:   tt.fields.logger,
			}
			tt.setup(tt.fields) // Call setup to configure mocks

			if err := s.SubscribeToUserEvents(tt.fields.args.ctx, tt.fields.eventBus, tt.fields.handlers, tt.fields.logger); (err != nil) != tt.wantErr {
				t.Errorf("UserSubscribers.SubscribeToUserEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
