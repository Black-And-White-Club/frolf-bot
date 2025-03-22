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
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// Helper function to create a pointer to an int
func intPtr(i int) *int {
	return &i
}
func TestRoundService_ParticipantTagFound(t *testing.T) {
	// Helper function to create a pointer to an int
	intPtr := func(i int) *int {
		return &i
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
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
					RoundID:   roundtypes.ID(1),
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				},
			},
			expectErr: false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoined), gomock.Any()).Return(nil).Times(1)
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
			name: "Missing round ID in payload", // Changed test name
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				},
			},
			expectErr: true,
			mockExpects: func() {
				// We do not expect any interaction with the database or event bus in this case.
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					RoundID:   roundtypes.ID(1),
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "UpdateParticipant returns error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberFoundPayload{
					RoundID:   roundtypes.ID(1),
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(fmt.Errorf("unexpected Tag Number: 1234")).Times(1)
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
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
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
					UserID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundParticipantJoined,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoined), gomock.Any()).Return(nil).Times(1)
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
					UserID: "some-discord-id",
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
					UserID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundParticipantJoinError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish ParticipantJoined event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundTagNumberNotFoundPayload{
					UserID: "some-discord-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), roundtypes.ID(1), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoined), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set RoundID in metadata for relevant test cases.  Crucially, we set it *before* calling tt.mockExpects
			if tt.name != "Invalid payload" && tt.name != "Missing round ID in metadata" {
				msg.Metadata.Set("RoundID", fmt.Sprintf("%d", 1))
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
