package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateRoundUpdateRequest(t *testing.T) {
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
			name: "Successful round update request validation",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateRequestPayload{
					RoundID: "some-round-id",
					Title:   func() *string { s := "New Title"; return &s }(),
				},
			},
			expectedEvent: roundevents.RoundUpdateValidated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateValidated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundUpdateValidated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundUpdateValidatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundUpdateRequestPayload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundUpdateRequestPayload.RoundID)
					}

					if *payload.RoundUpdateRequestPayload.Title != "New Title" {
						return fmt.Errorf("unexpected title: %s", *payload.RoundUpdateRequestPayload.Title)
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
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty round ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateRequestPayload{
					Title: func() *string { s := "New Title"; return &s }(),
				},
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "No fields to update",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateRequestPayload{
					RoundID: "some-round-id",
				},
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundUpdateValidated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateRequestPayload{
					RoundID: "some-round-id",
					Title:   func() *string { s := "New Title"; return &s }(),
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateValidated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.ValidateRoundUpdateRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ValidateRoundUpdateRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRoundUpdateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateRoundEntity(t *testing.T) {
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
			name: "Successful round entity update",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Old Title",
						Location:  "Old Location",
						EventType: func() *string { s := "Old Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID:   "some-round-id",
						Title:     func() *string { s := "New Title"; return &s }(),
						Location:  func() *string { s := "New Location"; return &s }(),
						EventType: func() *string { s := "New Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Date(2024, time.January, 27, 0, 0, 0, 0, time.UTC); return &t }(),
						EndTime:   func() *time.Time { t := time.Date(0, time.January, 1, 10, 30, 0, 0, time.UTC); return &t }(),
					},
				},
			},
			expectedEvent: roundevents.RoundUpdated,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:        "some-round-id",
					Title:     "Old Title",
					Location:  "Old Location",
					EventType: func() *string { s := "Old Type"; return &s }(),
					StartTime: func() *time.Time { t := time.Now(); return &t }(),
					State:     roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, roundID string, round *roundtypes.Round) error {
						if round.Title != "New Title" {
							return fmt.Errorf("unexpected title: %s", round.Title)
						}
						if round.Location != "New Location" {
							return fmt.Errorf("unexpected location: %s", round.Location)
						}
						if *round.EventType != "New Type" {
							return fmt.Errorf("unexpected event type: %s", *round.EventType)
						}
						return nil
					},
				).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundUpdated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundUpdatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
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
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Old Title",
						Location:  "Old Location",
						EventType: func() *string { s := "Old Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: "some-round-id",
						Title:   func() *string { s := "New Title"; return &s }(),
					},
				},
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:        "some-round-id",
					Title:     "Old Title",
					Location:  "Old Location",
					EventType: func() *string { s := "Old Type"; return &s }(),
					StartTime: func() *time.Time { t := time.Now(); return &t }(),
					State:     roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundUpdated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Old Title",
						Location:  "Old Location",
						EventType: func() *string { s := "Old Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: "some-round-id",
						Title:   func() *string { s := "New Title"; return &s }(),
					},
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:        "some-round-id",
					Title:     "Old Title",
					Location:  "Old Location",
					EventType: func() *string { s := "Old Type"; return &s }(),
					StartTime: func() *time.Time { t := time.Now(); return &t }(),
					State:     roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
				RoundDB:  mockRoundDB,
				EventBus: mockEventBus,
				logger:   logger,
			}

			// Call the service function
			err := s.UpdateRoundEntity(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("UpdateRoundEntity() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateRoundEntity() unexpected error: %v", err)
				}
			}
		})
	}
}
