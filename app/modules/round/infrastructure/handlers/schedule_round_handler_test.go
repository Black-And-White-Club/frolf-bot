package roundhandlers

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleDiscordMessageIDUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Test Round")
	testStartTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
	testUserID := sharedtypes.DiscordID("user-123")
	testEventMessageID := "discord-msg-123"

	testPayload := &roundevents.RoundScheduledPayloadV1{
		GuildID: testGuildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:   testRoundID,
			Title:     testTitle,
			StartTime: &testStartTime,
			UserID:    testUserID,
		},
		EventMessageID: testEventMessageID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService)
		payload        *roundevents.RoundScheduledPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedErrMsg string
	}{
		{
			name: "Successfully schedule round events",
			fakeSetup: func(fake *FakeService) {
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					return results.SuccessResult[*roundtypes.ScheduleRoundEventsResult, error](&roundtypes.ScheduleRoundEventsResult{}), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0, // No downstream events for scheduling
		},
		{
			name: "Service returns failure",
			fakeSetup: func(fake *FakeService) {
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					return results.FailureResult[*roundtypes.ScheduleRoundEventsResult, error](errors.New("scheduling failed")), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					return roundservice.ScheduleRoundEventsResult{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					return roundservice.ScheduleRoundEventsResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Successfully schedule with description and location",
			fakeSetup: func(fake *FakeService) {
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					return results.SuccessResult[*roundtypes.ScheduleRoundEventsResult, error](&roundtypes.ScheduleRoundEventsResult{}), nil
				}
			},
			payload: &roundevents.RoundScheduledPayloadV1{
				GuildID: testGuildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       testTitle,
					Description: func() roundtypes.Description { d := roundtypes.Description("Test Description"); return d }(),
					Location:    func() roundtypes.Location { l := roundtypes.Location("Test Location"); return l }(),
					StartTime:   &testStartTime,
					UserID:      testUserID,
				},
				EventMessageID: testEventMessageID,
			},
			wantErr:       false,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
				logger:      logger,
			}

			ctx := context.Background()
			results, err := h.HandleDiscordMessageIDUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleDiscordMessageIDUpdated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
		})
	}
}
