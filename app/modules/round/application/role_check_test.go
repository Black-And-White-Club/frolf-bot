package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_CheckUserAuthorization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name              string
		args              args
		expectedEvent     string
		expectErr         bool
		mockExpects       func()
		creator           bool
		requestingDiscord string
	}{
		{
			name: "Successful authorization check for round creator",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        1,
						CreatedBy: "some-discord-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserUserID: "some-discord-id",
					},
				},
			},
			expectedEvent: roundevents.RoundDeleteAuthorized,
			expectErr:     false,
			creator:       true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteAuthorized), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundDeleteAuthorized {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundDeleteAuthorizedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Successful role check request for non-creator",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        1,
						CreatedBy: "creator-discord-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserUserID: "some-discord-id",
					},
				},
			},
			expectedEvent: roundevents.RoundUserRoleCheckRequest,
			expectErr:     false,
			creator:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUserRoleCheckRequest), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundUserRoleCheckRequest {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.UserRoleCheckRequestPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.UserID != "some-discord-id" {
						return fmt.Errorf("unexpected Discord ID: %s", payload.UserID)
					}

					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
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
			name: "Publish RoundDeleteAuthorized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        1,
						CreatedBy: "some-discord-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserUserID: "some-discord-id",
					},
				},
			},
			expectedEvent: "",
			expectErr:     true,
			creator:       true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteAuthorized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Publish RoundUserRoleCheckRequest event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        1,
						CreatedBy: "creator-discord-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserUserID: "some-discord-id",
					},
				},
			},
			expectedEvent: "",
			expectErr:     true,
			creator:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUserRoleCheckRequest), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.CheckUserAuthorization(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("CheckUserAuthorization() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("CheckUserAuthorization() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UserRoleCheckResult(t *testing.T) {
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
		hasRole       bool
	}{
		{
			name: "Successful authorization for user with role",
			args: args{
				ctx: context.Background(),
				payload: roundevents.UserRoleCheckResultPayload{
					UserID:  "some-discord-id",
					RoundID: 1,
					HasRole: true,
				},
			},
			expectedEvent: roundevents.RoundDeleteAuthorized,
			expectErr:     false,
			hasRole:       true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteAuthorized), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundDeleteAuthorized {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundDeleteAuthorizedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "User without role",
			args: args{
				ctx: context.Background(),
				payload: roundevents.UserRoleCheckResultPayload{
					UserID:  "some-discord-id",
					RoundID: 1,
					HasRole: false,
				},
			},
			expectedEvent: roundevents.RoundDeleteUnauthorized,
			expectErr:     false,
			hasRole:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteUnauthorized), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundDeleteUnauthorized {
						return fmt.Errorf("unexpected topic: %s", topic)
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
			name: "Publish RoundDeleteAuthorized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.UserRoleCheckResultPayload{
					UserID:  "some-discord-id",
					RoundID: 1,
					HasRole: true,
				},
			},
			expectedEvent: "",
			expectErr:     true,
			hasRole:       true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteAuthorized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Publish RoundDeleteUnauthorized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.UserRoleCheckResultPayload{
					UserID:  "some-discord-id",
					RoundID: 1,
					HasRole: false,
				},
			},
			expectedEvent: "",
			expectErr:     true,
			hasRole:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteUnauthorized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.UserRoleCheckResult(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("UserRoleCheckResult() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("UserRoleCheckResult() unexpected error: %v", err)
				}
			}
		})
	}
}
