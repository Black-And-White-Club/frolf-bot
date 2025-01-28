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

func TestRoundService_GetRound(t *testing.T) {
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
			name: "Successful round retrieval",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectedEvent: roundevents.RoundFetched,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:    "some-round-id",
					Title: "Test Round",
					State: roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFetched), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundFetched {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundFetchedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.Round.ID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.Round.ID)
					}

					if payload.Round.Title != "Test Round" {
						return fmt.Errorf("unexpected round title: %s", payload.Round.Title)
					}

					if payload.RoundUpdateRequestPayload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundUpdateRequestPayload.RoundID)
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
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundFetched event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:    "some-round-id",
					Title: "Test Round",
					State: roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFetched), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.GetRound(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("GetRound() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("GetRound() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_CheckRoundExists(t *testing.T) {
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
			name: "Successful round existence check",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectedEvent: roundevents.RoundToDeleteFetched,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:    "some-round-id",
					Title: "Test Round",
					State: roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundToDeleteFetched), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundToDeleteFetched {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundToDeleteFetchedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.Round.ID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.Round.ID)
					}

					if payload.Round.Title != "Test Round" {
						return fmt.Errorf("unexpected round title: %s", payload.Round.Title)
					}

					if payload.RoundDeleteRequestPayload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundDeleteRequestPayload.RoundID)
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
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectedEvent: roundevents.RoundDeleteError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundToDeleteFetched event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID: "some-round-id",
					},
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), "some-round-id").Return(&roundtypes.Round{
					ID:    "some-round-id",
					Title: "Test Round",
					State: roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundToDeleteFetched), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.CheckRoundExists(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("CheckRoundExists() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("CheckRoundExists() unexpected error: %v", err)
				}
			}
		})
	}
}
