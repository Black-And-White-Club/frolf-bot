package usersubscribers

import (
	"context"
	"errors"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	handlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestSubscribeToUserEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockHandlers := handlers.NewMockHandlers(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)

	ctx := context.Background()

	t.Run("Successful Subscription", func(t *testing.T) {
		// Expect a successful subscription to UserSignupRequest
		mockEventBus.EXPECT().
			Subscribe(ctx, userevents.UserSignupRequest.String(), gomock.Any()).
			Return(nil).
			Times(1)

		// Expect successful subscriptions for other events
		mockEventBus.EXPECT().
			Subscribe(ctx, userevents.UserRoleUpdateRequest.String(), gomock.Any()).
			Return(nil).
			Times(1)

		// Expect log messages for subscribing and successful subscription
		mockLogger.EXPECT().Info("Subscribing to event", gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Info("Successfully subscribed to event", gomock.Any()).AnyTimes()

		err := SubscribeToUserEvents(ctx, mockEventBus, mockHandlers, mockLogger)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Failed Subscription to UserSignupRequest", func(t *testing.T) {
		// Expect a failed subscription to UserSignupRequest
		mockEventBus.EXPECT().
			Subscribe(ctx, userevents.UserSignupRequest.String(), gomock.Any()).
			Return(errors.New("subscription error")).
			Times(1)

		err := SubscribeToUserEvents(ctx, mockEventBus, mockHandlers, mockLogger)
		if err == nil || err.Error() != "failed to subscribe to UserSignupRequest: subscription error" {
			t.Errorf("Expected error 'failed to subscribe to UserSignupRequest: subscription error', got: %v", err)
		}
	})

	t.Run("Failed Subscription to Other Event", func(t *testing.T) {
		// Expect a successful subscription to UserSignupRequest
		mockEventBus.EXPECT().
			Subscribe(ctx, userevents.UserSignupRequest.String(), gomock.Any()).
			Return(nil).
			Times(1)

		// Expect a failed subscription to UserRoleUpdateRequest
		mockEventBus.EXPECT().
			Subscribe(ctx, userevents.UserRoleUpdateRequest.String(), gomock.Any()).
			Return(errors.New("subscription error")).
			Times(1)

		// Expect log messages for subscribing and error
		mockLogger.EXPECT().Info("Subscribing to event", gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Error("Failed to subscribe to event", gomock.Any(), gomock.Any()).AnyTimes()

		err := SubscribeToUserEvents(ctx, mockEventBus, mockHandlers, mockLogger)
		if err == nil || err.Error() != "failed to subscribe to event user.role.update.request: subscription error" {
			t.Errorf("Expected error 'failed to subscribe to event user.role.update.request: subscription error', got: %v", err)
		}
	})
}
