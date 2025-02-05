package roundservice

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	rounddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ProcessRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddbmocks.NewMockRoundDB(ctrl)
	logger := slog.Default()

	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name        string
		args        args
		mockExpects func()
		wantErr     bool
	}{
		{
			name: "Successful round reminder processing",
			args: args{
				msg: createTestMessage(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq("some-round-id")).
					Return(&roundtypes.Round{
						ID: "some-round-id",
						Participants: []roundtypes.RoundParticipant{
							{DiscordID: "user1"},
							{DiscordID: "user2"},
						},
					}, nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "Invalid payload",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid json")),
			},
			mockExpects: func() {},
			wantErr:     true,
		},
		{
			name: "Database error",
			args: args{
				msg: createTestMessage(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq("some-round-id")).
					Return(nil, fmt.Errorf("database error")).
					Times(1)
			},
			wantErr: true,
		},
		{
			name: "No participants",
			args: args{
				msg: createTestMessage(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq("some-round-id")).
					Return(&roundtypes.Round{
						ID:           "some-round-id",
						Participants: []roundtypes.RoundParticipant{},
					}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "Failed to publish to Discord",
			args: args{
				msg: createTestMessage(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq("some-round-id")).
					Return(&roundtypes.Round{
						ID: "some-round-id",
						Participants: []roundtypes.RoundParticipant{
							{DiscordID: "user1"},
							{DiscordID: "user2"},
						},
					}, nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DiscordEventsSubject), gomock.Any()).
					Return(fmt.Errorf("publish error")).
					Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockExpects()

			s := &RoundService{
				RoundDB:  mockRoundDB,
				EventBus: mockEventBus,
				logger:   logger,
			}

			if err := s.ProcessRoundReminder(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ProcessRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createTestMessage(payload roundevents.RoundReminderPayload) *message.Message {
	payloadBytes, _ := json.Marshal(payload)
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlationID", payload.RoundID)
	return msg
}
