package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ParticipantTagFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
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
			name: "Successful participant tag found",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					RoundID:   "some-round-id",
					DiscordID: "some-discord-id",
					TagNumber: 1234,
				},
			},
			expectErr: false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "some-round-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, roundID string, participant roundtypes.RoundParticipant) error {
						if participant.DiscordID != "some-discord-id" {
							return fmt.Errorf("unexpected Discord ID: %s", participant.DiscordID)
						}
						if participant.TagNumber != 1234 {
							return fmt.Errorf("unexpected Tag Number: %d", participant.TagNumber)
						}
						if participant.Response != roundtypes.ResponseAccept {
							return fmt.Errorf("unexpected Response: %s", participant.Response)
						}
						return nil
					},
				).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ParticipantJoined), gomock.Any()).Return(nil).Times(1)
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
			name: "Missing round ID in metadata",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					DiscordID: "some-discord-id",
					TagNumber: 1234,
				},
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					RoundID:   "some-round-id",
					DiscordID: "some-discord-id",
					TagNumber: 1234,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "some-round-id", gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set RoundID in metadata for relevant test cases
			if tt.name != "Invalid payload" {
				payload := tt.args.payload.(roundevents.RoundTagNumberFoundPayload)
				if payload.RoundID != "" {
					msg.Metadata.Set("RoundID", payload.RoundID)
				}
			}

			tt.mockExpects()

			s := &RoundService{
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.ParticipantTagFound(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ParticipantTagFound() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParticipantTagFound() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_ParticipantTagNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
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
			name: "Successful participant join without tag",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberNotFoundPayload{
					DiscordID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.ParticipantJoined,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "some-round-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, roundID string, participant roundtypes.RoundParticipant) error {
						if participant.DiscordID != "some-discord-id" {
							return fmt.Errorf("unexpected Discord ID: %s", participant.DiscordID)
						}
						if participant.TagNumber != 0 {
							return fmt.Errorf("unexpected Tag Number: %d", participant.TagNumber)
						}
						if participant.Response != roundtypes.ResponseAccept {
							return fmt.Errorf("unexpected Response: %s", participant.Response)
						}
						return nil
					},
				).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ParticipantJoined), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.ParticipantJoined {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.ParticipantJoinedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
					}

					if payload.Participant != "some-discord-id" {
						return fmt.Errorf("unexpected participant ID: %s", payload.Participant)
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
			name: "Missing round ID in metadata",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberNotFoundPayload{
					DiscordID: "some-discord-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberNotFoundPayload{
					DiscordID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundParticipantJoinError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "some-round-id", gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish ParticipantJoined event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberNotFoundPayload{
					DiscordID: "some-discord-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "some-round-id", gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ParticipantJoined), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set RoundID in metadata for relevant test cases
			if tt.name != "Invalid payload" && tt.name != "Missing round ID in metadata" {
				msg.Metadata.Set("RoundID", "some-round-id")
			}

			tt.mockExpects()

			s := &RoundService{
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.ParticipantTagNotFound(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ParticipantTagNotFound() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParticipantTagNotFound() unexpected error: %v", err)
				}
			}
		})
	}
}
