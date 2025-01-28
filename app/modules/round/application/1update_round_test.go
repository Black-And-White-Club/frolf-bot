package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
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
						StartTime: time.Now(),
						State:     roundtypes.RoundStateUpcoming,
					},
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID:   "some-round-id",
						Title:     func() *string { s := "New Title"; return &s }(),
						Location:  func() *string { s := "New Location"; return &s }(),
						EventType: func() *string { s := "New Type"; return &s }(),
						Date:      func() *time.Time { t := time.Date(2024, time.January, 27, 0, 0, 0, 0, time.UTC); return &t }(),
						Time:      func() *time.Time { t := time.Date(0, time.January, 1, 10, 30, 0, 0, time.UTC); return &t }(),
					},
				},
			},
			expectedEvent: roundevents.RoundEntityUpdated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundEntityUpdated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundEntityUpdated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundEntityUpdatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.Round.ID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.Round.ID)
					}

					if payload.Round.Title != "New Title" {
						return fmt.Errorf("unexpected title: %s", payload.Round.Title)
					}

					if payload.Round.Location != "New Location" {
						return fmt.Errorf("unexpected location: %s", payload.Round.Location)
					}

					if *payload.Round.EventType != "New Type" {
						return fmt.Errorf("unexpected event type: %s", *payload.Round.EventType)
					}

					expectedTime := time.Date(2024, time.January, 27, 10, 30, 0, 0, time.UTC)
					if !payload.Round.StartTime.Equal(expectedTime) {
						return fmt.Errorf("unexpected start time: %v", payload.Round.StartTime)
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
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundEntityUpdated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Old Title",
						Location:  "Old Location",
						EventType: func() *string { s := "Old Type"; return &s }(),
						StartTime: time.Now(),
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
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundEntityUpdated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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

func TestRoundService_PublishRoundUpdated(t *testing.T) {
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
			name: "Successful round updated event publishing",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdatedPayload{
					RoundID: "some-round-id",
				},
			},
			expectedEvent: roundevents.RoundUpdateSuccess,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateSuccess), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundUpdateSuccess {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundUpdateSuccessPayload
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
			expectErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundUpdateSuccess event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdatedPayload{
					RoundID: "some-round-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateSuccess), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.PublishRoundUpdated(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("PublishRoundUpdated() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("PublishRoundUpdated() unexpected error: %v", err)
				}
			}
		})
	}
}
