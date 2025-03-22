package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_RequestTagNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Successful tag number request",
			args: args{
				ctx: context.Background(),
				payload: roundevents.TagNumberRequestPayload{
					UserID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundTagNumberRequest,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberRequest), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundTagNumberRequest {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.TagNumberRequestPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Publish event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.TagNumberRequestPayload{
					UserID: "some-discord-id",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberRequest), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects()

			s := &RoundService{
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.RequestTagNumber(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("RequestTagNumber() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("RequestTagNumber() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_TagNumberRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Successful tag number request to leaderboard",
			args: args{
				ctx: context.Background(),
				payload: roundevents.TagNumberRequestPayload{
					UserID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.LeaderboardGetTagNumberRequest,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.LeaderboardGetTagNumberRequest), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.LeaderboardGetTagNumberRequest {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.TagNumberRequestPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Publish event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.TagNumberRequestPayload{
					UserID: "some-discord-id",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.LeaderboardGetTagNumberRequest), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects()

			s := &RoundService{
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.TagNumberRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("TagNumberRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("TagNumberRequest() unexpected error: %v", err)
				}
			}
		})
	}
}
func TestRoundService_TagNumberResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Successful tag number found",
			args: args{
				ctx: context.Background(),
				payload: roundevents.GetTagNumberResponsePayload{
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
					Error:     "",
				},
			},
			expectedEvent: roundevents.RoundTagNumberFound,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberFound), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundTagNumberFound {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundTagNumberFoundPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					if payload.TagNumber == nil || *payload.TagNumber != 1234 { // Corrected assertion
						return fmt.Errorf("unexpected tag number: %v", payload.TagNumber)
					}
					if payload.RoundID != 1 { //Added this assertion
						return fmt.Errorf("unexpected round ID: %d", payload.RoundID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Tag number not found",
			args: args{
				ctx: context.Background(),
				payload: roundevents.GetTagNumberResponsePayload{
					UserID:    "some-discord-id",
					TagNumber: nil, // Changed to nil
					Error:     "",
				},
			},
			expectedEvent: roundevents.RoundTagNumberNotFound,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberNotFound), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundTagNumberNotFound {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundTagNumberNotFoundPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Error from leaderboard",
			args: args{
				ctx: context.Background(),
				payload: roundevents.GetTagNumberResponsePayload{
					UserID:    "some-discord-id",
					TagNumber: nil,
					Error:     "some error",
				},
			},
			expectErr: false, // Expecting no error, the function handles the error.
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberNotFound), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundTagNumberNotFound {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundTagNumberNotFoundPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Publish RoundTagNumberFound event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.GetTagNumberResponsePayload{
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
					Error:     "",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberFound), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Publish RoundTagNumberNotFound event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.GetTagNumberResponsePayload{
					UserID:    "some-discord-id",
					TagNumber: nil,
					Error:     "",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundTagNumberNotFound), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())
			msg.Metadata.Set("RoundID", "1") // Corrected RoundID to a valid integer string

			tt.mockExpects()

			s := &RoundService{
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.TagNumberResponse(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("TagNumberResponse() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("TagNumberResponse() unexpected error: %v", err)
				}
			}
		})
	}
}
