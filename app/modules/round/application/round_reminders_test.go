package roundservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	errormocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
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
	reminderRoundID       roundtypes.ID = 1
	reminderCorrelationID               = "some-correlation-id"
	reminderType                        = "1h"
	reminderRoundTitle                  = "Test Round"
	reminderUser1                       = "user1"
	reminderUser2                       = "user2"
	reminderDBError                     = "database error"
	reminderPubError                    = "publish error"
)

var (
	reminderLocation roundtypes.Location = "Test Location"
	//valid reminder
	validReminderPayload = roundevents.DiscordReminderPayload{
		RoundID:      1,
		RoundTitle:   reminderRoundTitle,
		ReminderType: reminderType,
	}
	//Valid Round
	validReminderRound = roundtypes.Round{
		ID: 1,
		Participants: []roundtypes.Participant{
			{UserID: reminderUser1},
			{UserID: reminderUser2},
		},
		Location: &reminderLocation,
	}
)

func TestRoundService_ProcessRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddbmocks.NewMockRoundDBInterface(ctrl)
	mockErrorReporter := errormocks.NewMockErrorReporterInterface(ctrl)
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
			name:          "Successful round reminder processing",
			payload:       validReminderPayload,             // Use pre-built payload
			expectedEvent: roundevents.DiscordRoundReminder, // Expect publish to Discord
			wantErr:       false,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(reminderRoundID)).
					Return(&validReminderRound, nil). // Return valid round with participants
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundReminder), gomock.Any()).
					Times(1).
					Return(nil)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: "", // No events expected
			wantErr:       true,
			errMsg:        "failed to unmarshal RoundReminderPayload",
			mockDBSetup:   func() {}, // No DB interactions expected
		},
		{
			name:          "Database error",
			payload:       validReminderPayload,
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to get round from database: " + reminderDBError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(reminderRoundID)).
					Return(nil, fmt.Errorf("failed to get round from database: %s", reminderDBError)).
					Times(1)
			},
		},
		{
			name:          "No participants",
			payload:       validReminderPayload,
			expectedEvent: "",    // No event published to Discord
			wantErr:       false, // Not an error, just a skipped publish
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(reminderRoundID)).
					Return(&roundtypes.Round{ // Return a round with NO participants
						ID:           1,
						Participants: []roundtypes.Participant{},
					}, nil).
					Times(1)
			},
		},
		{
			name:          "Failed to publish to Discord",
			payload:       validReminderPayload,
			expectedEvent: roundevents.DiscordRoundReminder, // Corrected expected event
			wantErr:       true,
			errMsg:        "failed to publish Discord notification: " + reminderPubError,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(reminderRoundID)).
					Return(&validReminderRound, nil). // Valid round with participants
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordRoundReminder), gomock.Any()).
					Return(errors.New(reminderPubError)).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, reminderCorrelationID)

			if tt.mockDBSetup != nil {
				tt.mockDBSetup()
			}

			err := s.ProcessRoundReminder(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ProcessRoundReminder() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ProcessRoundReminder() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ProcessRoundReminder() unexpected error: %v", err)
				}
			}
		})
	}
}
