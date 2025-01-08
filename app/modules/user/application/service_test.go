package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	adaptermock "github.com/Black-And-White-Club/tcr-bot/app/adapters/mocks"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"

	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_checkTagAvailability(t *testing.T) {
	tests := []struct {
		name            string
		tagNumber       int
		isAvailable     bool
		timeout         bool
		expectError     error
		expected        bool
		publishPayload  map[string]interface{}
		responsePayload map[string]interface{}
	}{
		{
			name:        "Happy Path",
			tagNumber:   123,
			isAvailable: true,
			timeout:     false,
			expectError: nil,
			expected:    true,
			publishPayload: map[string]interface{}{
				"tag_number": 123,
			},
			responsePayload: map[string]interface{}{
				"is_available": true,
			},
		},
		{
			name:        "Timeout",
			tagNumber:   456,
			isAvailable: false,
			timeout:     true,
			expectError: errTimeout,
			expected:    false,
			publishPayload: map[string]interface{}{
				"tag_number": 456,
			},
			responsePayload: nil,
		},
		{
			name:        "Tag Not Available",
			tagNumber:   789,
			isAvailable: false,
			timeout:     false,
			expectError: nil,
			expected:    false,
			publishPayload: map[string]interface{}{
				"tag_number": 789,
			},
			responsePayload: map[string]interface{}{
				"is_available": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)
			mockEventAdapter := adaptermock.NewMockEventAdapterInterface(ctrl)

			mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)

			mockEventBus.EXPECT().
				Publish(gomock.Any(), userevents.CheckTagAvailabilityRequest, gomock.Any()).
				DoAndReturn(func(_ context.Context, eventType shared.EventType, msg shared.Message) error {
					if !testutils.NewMessageMatcher(t, tt.publishPayload).Matches(msg) {
						t.Errorf("message payload does not match expected: %v", tt.publishPayload)
						return fmt.Errorf("message does not match expected")
					}
					return nil
				}).
				Times(1)

			// Use a very short timeout duration for the test
			const shortTimeout = 1 * time.Nanosecond
			var ctxWithTimeout context.Context
			var cancel context.CancelFunc
			if tt.timeout {
				ctxWithTimeout, cancel = context.WithTimeout(context.Background(), shortTimeout)
				cancel() // Simulate immediate timeout
			} else {
				ctxWithTimeout, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			// Expect Subscribe to be called
			mockEventBus.EXPECT().
				Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, shared.Message) error) error {
					if !tt.timeout && tt.responsePayload != nil {
						payloadBytes, _ := json.Marshal(tt.responsePayload)
						responseMsg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadBytes) // Use the function from the adapters package

						// Directly call the handler function to simulate the response
						go func() {
							handler(ctx, responseMsg)
						}()
					}
					return nil
				}).
				AnyTimes()

			service := &UserServiceImpl{
				eventBus:     mockEventBus,
				logger:       mockLogger,
				eventAdapter: mockEventAdapter,
			}

			got, err := service.checkTagAvailability(ctxWithTimeout, tt.tagNumber)

			if tt.expectError != nil {
				if err == nil || err.Error() != tt.expectError.Error() {
					t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if got != tt.expected {
				t.Errorf("Expected: %v, got: %v", tt.expected, got)
			}
		})
	}
}

func TestUserServiceImpl_publishTagAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		discordID    usertypes.DiscordID
		tagNumber    int
		mockEventBus func(mockEventBus *eventbusmock.MockEventBus)
		wantErr      bool
	}{
		{
			name:      "Happy Path",
			discordID: "1234567890",
			tagNumber: 123,
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.TagAssignedRequest, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "Error Publishing Event",
			discordID: "9876543210",
			tagNumber: 456,
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.TagAssignedRequest, gomock.Any()).
					Return(fmt.Errorf("simulated publish error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)
			mockEventAdapter := new(adaptermock.MockEventAdapterInterface)

			// Set up logger expectations
			mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1)

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus:     mockEventBus,
				logger:       mockLogger,
				eventAdapter: mockEventAdapter,
			}

			// Call publishTagAssigned with a context
			ctx := context.Background()
			err := s.publishTagAssigned(ctx, tt.discordID, tt.tagNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_publishUserRoleUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		payload      userevents.UserRoleUpdatedPayload
		mockEventBus func(mockEventBus *eventbusmock.MockEventBus)
		wantErr      bool
	}{
		{
			name: "Happy Path",
			payload: userevents.UserRoleUpdatedPayload{
				DiscordID: "1234567890",
				NewRole:   usertypes.UserRoleAdmin.String(),
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.UserRoleUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error Publishing Event",
			payload: userevents.UserRoleUpdatedPayload{
				DiscordID: "1234567890",
				NewRole:   usertypes.UserRoleAdmin.String(),
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.UserRoleUpdated, gomock.Any()).
					Return(fmt.Errorf("failed to publish"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)
			mockEventAdapter := new(adaptermock.MockEventAdapterInterface)

			// Set up logger expectations
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes() // Allow any number of calls to Info

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus:     mockEventBus,
				logger:       mockLogger,
				eventAdapter: mockEventAdapter,
			}

			ctx := context.Background()
			err := s.publishUserRoleUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishUserRoleUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
