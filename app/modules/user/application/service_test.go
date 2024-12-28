package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/events"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
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
				"tag_number": float64(123), // Ensure tag_number is a float64
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
				"tag_number": float64(456), // Ensure tag_number is a float64
			},
			responsePayload: nil, // No response payload expected in timeout scenario
		},
		{
			name:        "Tag Not Available",
			tagNumber:   789,
			isAvailable: false,
			timeout:     false,
			expectError: nil,
			expected:    false,
			publishPayload: map[string]interface{}{
				"tag_number": float64(789), // Ensure tag_number is a float64
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

			// Create mock dependencies
			mockUserDB := userdb.NewMockUserDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			// Set up expectations for the mock logger
			mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
			mockLogger.EXPECT().Debug("Generated replySubject", gomock.Any()).Times(1)
			if tt.timeout {
				mockLogger.EXPECT().Error("Response channel full", nil, gomock.Any()).Times(0)
			}

			// Set up expectations for the mock event bus
			mockEventBus.EXPECT().
				Publish(gomock.Any(), userevents.CheckTagAvailabilityRequest, gomock.Any()).
				DoAndReturn(func(_ context.Context, eventType events.EventType, msg types.Message) error {
					// Validate message payload and metadata using MessageMatcher
					if !testutils.NewMessageMatcher(t, tt.publishPayload).Matches(msg) {
						fmt.Println("Expected payload:", tt.publishPayload)
						fmt.Println("Actual message:", msg)
						return fmt.Errorf("message does not match expected")
					}
					return nil
				}).
				Times(1)

			mockEventBus.EXPECT().
				Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, types.Message) error) error {
					if tt.timeout {
						// Simulate a timeout by returning the custom timeout error
						return errTimeout
					} else {
						// Simulate receiving a response
						if tt.responsePayload != nil {
							mockMsg := testutils.NewMockMessage(ctrl)
							mockMsg.EXPECT().Metadata().Return(message.Metadata{}).AnyTimes()
							mockMsg.EXPECT().UUID().Return("").AnyTimes()

							mockMsg.EXPECT().Payload().Return(func() []byte {
								payloadBytes, err := json.Marshal(tt.responsePayload)
								if err != nil {
									t.Fatalf("Failed to marshal response payload: %v", err)
								}
								return payloadBytes
							}()).Times(1)

							// Call the handler with the mocked message
							if handler != nil {
								err := handler(ctx, mockMsg)
								if err != nil {
									t.Errorf("Handler returned an error: %v", err)
								}
							}
						}
						return nil
					}
				}).
				Times(1)

			// Create the service instance
			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   mockLogger,
			}

			// Call the method being tested with appropriate timeout context
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.timeout {
				ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			}
			defer cancel()

			got, err := service.checkTagAvailability(ctx, tt.tagNumber)

			// Assert the results
			if err != tt.expectError {
				t.Errorf("Expected error: %v, got %v", tt.expectError, err)
			}
			if got != tt.expected {
				t.Errorf("Expected result: %v, got %v", tt.expected, got)
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
		// Add more test cases as needed
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			// Set up logger expectations (if needed)
			mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1)

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus: mockEventBus,
				logger:   mockLogger,
			}
			if err := s.publishTagAssigned(context.Background(), tt.discordID, tt.tagNumber); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
