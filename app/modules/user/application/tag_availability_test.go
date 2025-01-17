package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_checkTagAvailability(t *testing.T) {
	errTimeout := errors.New("timeout waiting for tag availability response") // Declare errTimeout here
	type fields struct {
		_ *eventbusmock.MockEventBus
		_ *slog.Logger
	}
	type args struct {
		_ context.Context
		_ int
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		tagNumber       int
		responsePayload *events.CheckTagAvailabilityResponsePayload
		timeout         bool
		expected        bool
		expectError     error
	}{
		{
			name:      "Tag Available",
			tagNumber: 123,
			responsePayload: &events.CheckTagAvailabilityResponsePayload{
				IsAvailable: true,
				Error:       "",
			},
			expected:    true,
			expectError: nil,
		},
		{
			name:      "Tag Not Available",
			tagNumber: 456,
			responsePayload: &events.CheckTagAvailabilityResponsePayload{
				IsAvailable: false,
				Error:       "",
			},
			expected:    false,
			expectError: nil,
		},
		{
			name:        "Timeout",
			tagNumber:   789,
			timeout:     true,
			expected:    false,
			expectError: errTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Set up the expected call to Publish for CheckTagAvailabilityRequest
			mockEventBus.EXPECT().
				Publish(gomock.Any(), events.LeaderboardStreamName, gomock.Any()). // Expect leaderboard stream name
				DoAndReturn(func(_ context.Context, streamName string, msg *message.Message) error {
					// Verify the stream name
					if streamName != events.LeaderboardStreamName {
						t.Errorf("Expected stream name: %s, got: %s", events.LeaderboardStreamName, streamName)
					}
					// Verify the subject from message metadata
					subject := msg.Metadata.Get("subject")
					if subject != events.CheckTagAvailabilityRequest {
						t.Errorf("Expected subject: %s, got: %s", events.CheckTagAvailabilityRequest, subject)
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
				defer cancel() // Cancel immediately if timeout is expected
			} else {
				ctxWithTimeout, cancel = context.WithCancel(context.Background())
				defer cancel() // Ensure cancel is called even when no timeout is expected
			}

			// Set up the expected call to Subscribe for CheckTagAvailabilityResponse
			mockEventBus.EXPECT().
				Subscribe(gomock.Any(), events.LeaderboardStreamName, events.CheckTagAvailabilityResponse, gomock.Any()).
				DoAndReturn(func(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
					// Verify the stream name and subject
					if streamName != events.LeaderboardStreamName { // Expect LeaderboardCheckTagAvailabilityStreamName
						t.Errorf("Expected stream name: %s, got: %s", events.LeaderboardStreamName, streamName)
					}
					if subject != events.CheckTagAvailabilityResponse {
						t.Errorf("Expected subject: %s, got: %s", events.CheckTagAvailabilityResponse, subject)
					}

					if !tt.timeout && tt.responsePayload != nil {
						payloadBytes, _ := json.Marshal(tt.responsePayload)
						responseMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
						responseMsg.Metadata.Set("subject", events.CheckTagAvailabilityResponse)

						// Call the handler function to simulate the response
						go handler(ctx, responseMsg)
					}
					return nil
				}).
				AnyTimes()

			service := &UserServiceImpl{
				eventBus: mockEventBus,
				logger:   logger,
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
		name          string
		discordID     usertypes.DiscordID
		tagNumber     int
		mockEventBus  func(mockEventBus *eventbusmock.MockEventBus)
		wantErr       bool
		expectedError string
	}{
		{
			name:      "Happy Path",
			discordID: "1234567890",
			tagNumber: 123,
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						// Assertions for stream name
						if streamName != events.LeaderboardStreamName {
							t.Errorf("Expected stream name: %s, got: %s", events.LeaderboardStreamName, streamName)
						}

						// Assertions for message metadata
						if msg.Metadata.Get("subject") != events.TagAssignedRequest {
							t.Errorf("Expected subject: %s, got: %s", events.TagAssignedRequest, msg.Metadata.Get("subject"))
						}

						// Assertions for payload
						var payload events.TagAssignedRequestPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							t.Errorf("Failed to unmarshal message payload: %v", err)
						}
						if string(payload.DiscordID) != "1234567890" {
							t.Errorf("Expected Discord ID: %s, got: %s", "1234567890", string(payload.DiscordID))
						}
						if payload.TagNumber != 123 {
							t.Errorf("Expected Tag Number: %d, got: %d", 123, payload.TagNumber)
						}
						return nil
					}).
					Times(1)
			},
			wantErr:       false,
			expectedError: "",
		},
		{
			name:      "Error Publishing Event",
			discordID: "9876543210",
			tagNumber: 456,
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).
					Return(fmt.Errorf("simulated publish error")).
					Times(1)
			},
			wantErr:       true,
			expectedError: "simulated publish error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockUserDB := userdb.NewMockUserDB(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			service := NewUserService(mockUserDB, mockEventBus, logger)

			// Invoke the function
			ctx := context.Background()
			err := service.publishTagAssigned(ctx, tt.discordID, tt.tagNumber)

			// Validate errors
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error message to contain: %q, but got: %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}
