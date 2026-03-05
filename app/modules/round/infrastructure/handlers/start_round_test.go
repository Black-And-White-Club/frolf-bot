package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundStartRequested_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testClubUUID := uuid.New()

	testPayload := &roundevents.RoundStartRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService, *FakeUserService)
		payload        *roundevents.RoundStartRequestedPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedTopics []string
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundStarted",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						Title:          "Test Round",
						Location:       "Test Location",
						EventMessageID: "msg-123",
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1")},
							{UserID: sharedtypes.DiscordID("user2")},
						},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 4,
			expectedTopics: []string{
				roundevents.RoundStartedV1,
				roundevents.RoundStartedV1 + "." + string(testGuildID),
				roundevents.RoundStartedV1 + "." + testClubUUID.String(),
				roundevents.RoundStartedDiscordV1,
			},
		},
		{
			name: "Successfully handle PWA-only start without discord message",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:       testRoundID,
						GuildID:  testGuildID,
						Title:    "Test Round",
						Location: "Test Location",
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 3,
			expectedTopics: []string{
				roundevents.RoundStartedV1,
				roundevents.RoundStartedV1 + "." + string(testGuildID),
				roundevents.RoundStartedV1 + "." + testClubUUID.String(),
			},
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return roundservice.StartRoundResult{}, errors.New("database connection failed")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection failed",
		},
		{
			name: "Service returns failure result",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("start failed")), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			expectedTopics: []string{
				roundevents.RoundStartFailedV1,
			},
		},
		{
			name: "Service returns empty result (unknown)",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return roundservice.StartRoundResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			fakeUserService := NewFakeUserService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService, fakeUserService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: fakeUserService,
				logger:      logger,
			}

			results, err := h.HandleRoundStartRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStartRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundStartRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundStartRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			for i, topic := range tt.expectedTopics {
				if i >= len(results) {
					t.Fatalf("expected topic at index %d = %q but got only %d results", i, topic, len(results))
				}
				if results[i].Topic != topic {
					t.Fatalf("result topic at index %d = %q, want %q", i, results[i].Topic, topic)
				}
			}
		})
	}
}
