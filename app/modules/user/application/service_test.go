package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_checkTagAvailability(t *testing.T) {
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
		responsePayload *userevents.CheckTagAvailabilityResponsePayload
		timeout         bool
		expected        bool
		expectError     error
	}{
		{
			name:      "Tag Available",
			tagNumber: 123,
			responsePayload: &userevents.CheckTagAvailabilityResponsePayload{
				IsAvailable: true,
				Error:       "",
			},
			expected:    true,
			expectError: nil,
		},
		{
			name:      "Tag Not Available",
			tagNumber: 456,
			responsePayload: &userevents.CheckTagAvailabilityResponsePayload{
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
				Publish(gomock.Any(), userstream.LeaderboardStreamName, gomock.Any()). // Expect leaderboard stream name
				DoAndReturn(func(_ context.Context, streamName string, msg *message.Message) error {
					// Verify the stream name
					if streamName != userstream.LeaderboardStreamName {
						t.Errorf("Expected stream name: %s, got: %s", userstream.LeaderboardStreamName, streamName)
					}
					// Verify the subject from message metadata
					subject := msg.Metadata.Get("subject")
					if subject != userevents.CheckTagAvailabilityRequest {
						t.Errorf("Expected subject: %s, got: %s", userevents.CheckTagAvailabilityRequest, subject)
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
				Subscribe(gomock.Any(), userstream.LeaderboardStreamName, userevents.CheckTagAvailabilityResponse, gomock.Any()).
				DoAndReturn(func(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
					// Verify the stream name and subject
					if streamName != userstream.LeaderboardStreamName { // Expect LeaderboardStreamName
						t.Errorf("Expected stream name: %s, got: %s", userstream.LeaderboardStreamName, streamName)
					}
					if subject != userevents.CheckTagAvailabilityResponse {
						t.Errorf("Expected subject: %s, got: %s", userevents.CheckTagAvailabilityResponse, subject)
					}

					if !tt.timeout && tt.responsePayload != nil {
						payloadBytes, _ := json.Marshal(tt.responsePayload)
						responseMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
						responseMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityResponse)

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
					Publish(gomock.Any(), userstream.LeaderboardStreamName, gomock.Any()). // Expect LeaderboardStreamName
					DoAndReturn(func(_ context.Context, streamName string, msg *message.Message) error {
						if streamName != userstream.LeaderboardStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.LeaderboardStreamName, streamName)
						}
						subject := msg.Metadata.Get("subject")
						if subject != userevents.TagAssignedRequest {
							t.Errorf("Expected subject: %s, got: %s", userevents.TagAssignedRequest, subject)
						}
						return nil
					}).
					Times(1)
			},
			wantErr: false,
		},
		{
			name:      "Error Publishing Event",
			discordID: "9876543210",
			tagNumber: 456,
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userstream.LeaderboardStreamName, gomock.Any()). // Expect LeaderboardStreamName
					Return(fmt.Errorf("simulated publish error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus: mockEventBus,
				logger:   logger,
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
					Publish(gomock.Any(), userstream.UserRoleUpdateResponseStreamName, gomock.Any()). // Expect UserRoleUpdateResponseStreamName
					DoAndReturn(func(_ context.Context, streamName string, msg *message.Message) error {
						if streamName != userstream.UserRoleUpdateResponseStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.UserRoleUpdateResponseStreamName, streamName)
						}
						subject := msg.Metadata.Get("subject")
						if subject != userevents.UserRoleUpdated {
							t.Errorf("Expected subject: %s, got: %s", userevents.UserRoleUpdated, subject)
						}
						return nil
					}).
					Times(1)
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
					Publish(gomock.Any(), userstream.UserRoleUpdateResponseStreamName, gomock.Any()). // Expect UserRoleUpdateResponseStreamName
					Return(fmt.Errorf("failed to publish"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus: mockEventBus,
				logger:   logger,
			}

			ctx := context.Background()
			err := s.publishUserRoleUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishUserRoleUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
