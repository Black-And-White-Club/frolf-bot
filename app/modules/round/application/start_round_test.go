package roundservice

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---
const (
	startRoundID       roundtypes.ID = 1
	startCorrelationID               = "some-correlation-id"
	startRoundTitle                  = "Test Round"
	startDBError                     = "database error"
	startUpdateError                 = "update error"
	startPublishError                = "publish error"
	startDiscordUser1                = "user1"
	startDiscordUser2                = "user2"
)

var (
	startLocation = roundtypes.Location("Test Location")
	startNow      = time.Now().UTC().Truncate(time.Second)
	startTime     = roundtypes.StartTime(startNow)

	validStartPayload = roundevents.RoundStartedPayload{
		RoundID:   startRoundID,
		Title:     startRoundTitle,
		Location:  &startLocation,
		StartTime: &startTime,
	}
	validRoundStart = &roundtypes.Round{
		ID: startRoundID,
		Participants: []roundtypes.Participant{
			{UserID: startDiscordUser1},
			{UserID: startDiscordUser2},
		},
		Location: &startLocation,
	}
)

func TestRoundService_ProcessRoundStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddbmocks.NewMockRoundDBInterface(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")

	logger := slog.Default()

	s := &RoundService{
		RoundDB:       mockRoundDB,
		EventBus:      mockEventBus,
		logger:        logger,
		ErrorReporter: mockErrorReporter,
	}

	tests := []struct {
		name          string
		payload       interface{}
		mockDBSetup   func()
		expectedEvent string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "Successful round start processing",
			payload:       validStartPayload,
			expectedEvent: roundevents.DiscordRoundStarted, // Expect initial publish
			wantErr:       false,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(validRoundStart, nil). // Return valid round with participants
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(startRoundID), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundStarted), gomock.Any()).
					Times(1).Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Times(1).Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStateUpdated), gomock.Any()).
					Times(1).Return(nil)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to unmarshal payload",
		},
		{
			name:          "Database error",
			payload:       validStartPayload,
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to get round from database: " + startDBError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(nil, fmt.Errorf("failed to get round from database: %s", startDBError)). // Simulate DB error
					Times(1)
			},
		},
		{
			name:          "Failed to update round",
			payload:       validStartPayload,
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to update round: " + startUpdateError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(validRoundStart, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(startRoundID), gomock.Any()).
					Return(fmt.Errorf("failed to update round: %s", startUpdateError)). // Simulate update error
					Times(1)
			},
		},
		{
			name:          "Failed to publish round.started event",
			payload:       validStartPayload,
			expectedEvent: roundevents.DiscordRoundStarted, // Event should *attempt* to be published
			wantErr:       true,
			errMsg:        "failed to publish round.started event: " + startPublishError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(validRoundStart, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(startRoundID), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect the first publish call to fail with an error
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundStarted), gomock.Any()).
					Return(fmt.Errorf("failed to publish round.started event: %s", startPublishError)).
					Times(1)

				// These shouldn't be called if the first publish fails
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Times(0)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStateUpdated), gomock.Any()).
					Times(0)

			},
		},
		{
			name:          "Failed to publish to Discord",
			payload:       validStartPayload,
			expectedEvent: roundevents.DiscordEventsSubject,
			wantErr:       true,
			errMsg:        "failed to publish to discord.round.event: " + startPublishError,
			// For the "Failed to publish to Discord" test case:
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(validRoundStart, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(startRoundID), gomock.Any()).
					Return(nil).
					Times(1)

				// First publish succeeds
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundStarted), gomock.Any()).
					Return(nil).
					Times(1)

				// Second publish fails with error
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Return(fmt.Errorf("failed to publish to discord.round.event: %s", startPublishError)).
					Times(1)

				// This shouldn't be called if the second publish fails
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStateUpdated), gomock.Any()).
					Times(0)
			},
		},
		{
			name:          "Failed to publish round.state.updated event",
			payload:       validStartPayload,
			expectedEvent: roundevents.RoundStateUpdated,
			wantErr:       true,
			errMsg:        "failed to publish round.state.updated event: " + startPublishError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(startRoundID)).
					Return(validRoundStart, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(startRoundID), gomock.Any()).
					Return(nil).
					Times(1)
				// Expect the first publish to succeed
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundStarted), gomock.Any()).
					Return(nil).
					Times(1)
				// Expect the second publish to succeed
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Return(nil).
					Times(1)
				// Expect the third publish to *fail*
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStateUpdated), gomock.Any()).
					Return(fmt.Errorf("failed to publish round.state.updated event: %s", startPublishError)).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, startCorrelationID)

			if tt.mockDBSetup != nil {
				tt.mockDBSetup()
			}

			err := s.ProcessRoundStart(msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ProcessRoundStart() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ProcessRoundStart() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ProcessRoundStart() unexpected error: %v", err)
				}
			}
		})
	}
}
