package roundhandlers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func scorePointer(s sharedtypes.Score) *sharedtypes.Score {
	return &s
}

func TestRoundHandlers_HandleAllScoresSubmitted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now().UTC())
	testGuildID := sharedtypes.GuildID("guild-123")
	testClubUUID := uuid.New()
	testEventMessageID := "msg-123"

	testParticipants := []roundtypes.Participant{
		{
			UserID:   sharedtypes.DiscordID("user1"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(60)),
		},
		{
			UserID:   sharedtypes.DiscordID("user2"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(65)),
		},
	}

	testPayload := &roundevents.AllScoresSubmittedPayloadV1{
		GuildID:        testGuildID,
		RoundID:        testRoundID,
		EventMessageID: testEventMessageID,
		RoundData: roundtypes.Round{
			ID:             testRoundID,
			Title:          testTitle,
			Location:       testLocation,
			StartTime:      &testStartTime,
			EventMessageID: testEventMessageID,
			Participants:   testParticipants,
		},
		Participants: testParticipants,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService, *FakeUserService)
		payload        *roundevents.AllScoresSubmittedPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedErrMsg string
	}{
		{
			name: "Successfully handle AllScoresSubmitted",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					return results.SuccessResult[*roundtypes.FinalizeRoundResult, error](&roundtypes.FinalizeRoundResult{
						Round: &roundtypes.Round{
							ID:             testRoundID,
							Title:          testTitle,
							Location:       testLocation,
							StartTime:      &testStartTime,
							EventMessageID: testEventMessageID,
						},
						Participants: testParticipants,
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 4, // Discord + Backend finalization + Guild-scoped + Club-scoped
		},
		{
			name: "Service returns finalization failure",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					return results.FailureResult[*roundtypes.FinalizeRoundResult, error](errors.New("finalization failed")), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1, // Error event
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					return roundservice.FinalizeRoundResult{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					return roundservice.FinalizeRoundResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Payload with no GuildID",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					return results.SuccessResult[*roundtypes.FinalizeRoundResult, error](&roundtypes.FinalizeRoundResult{
						Round: &roundtypes.Round{
							ID:             testRoundID,
							Title:          testTitle,
							Location:       testLocation,
							StartTime:      &testStartTime,
							EventMessageID: testEventMessageID,
						},
						Participants: testParticipants,
					}), nil
				}
			},
			payload: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        "",
				RoundID:        testRoundID,
				EventMessageID: testEventMessageID,
				RoundData: roundtypes.Round{
					ID:             testRoundID,
					Title:          testTitle,
					Location:       testLocation,
					StartTime:      &testStartTime,
					EventMessageID: testEventMessageID,
					Participants:   testParticipants,
				},
				Participants: testParticipants,
			},
			wantErr:       false,
			wantResultLen: 2, // No guild-scoped event when GuildID is empty
		},
		{
			name: "Payload with multiple participants",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.FinalizeRoundFunc = func(ctx context.Context, req *roundtypes.FinalizeRoundInput) (roundservice.FinalizeRoundResult, error) {
					manyParticipants := []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
						{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
						{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
					}
					return results.SuccessResult[*roundtypes.FinalizeRoundResult, error](&roundtypes.FinalizeRoundResult{
						Round: &roundtypes.Round{
							ID:             testRoundID,
							Title:          testTitle,
							EventMessageID: testEventMessageID,
						},
						Participants: manyParticipants,
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				EventMessageID: testEventMessageID,
				RoundData: roundtypes.Round{
					ID:             testRoundID,
					Title:          testTitle,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
						{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
						{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
					},
				},
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
					{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
					{UserID: sharedtypes.DiscordID("user4"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(70))},
				},
			},
			wantErr:       false,
			wantResultLen: 4,
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
				helpers:     utils.NewHelper(logger),
			}

			ctx := context.Background()
			results, err := h.HandleAllScoresSubmitted(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAllScoresSubmitted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleAllScoresSubmitted() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleAllScoresSubmitted() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			// Verify metadata for Discord message
			if tt.wantResultLen == 4 {
				if results[0].Topic != roundevents.RoundFinalizedDiscordV1 {
					t.Errorf("First result should be Discord finalization, got %v", results[0].Topic)
				}
				if results[0].Metadata["discord_message_id"] != testEventMessageID {
					t.Errorf("Discord message ID metadata not set correctly")
				}
				if results[1].Topic != roundevents.RoundFinalizedV1 {
					t.Errorf("Second result should be backend finalization, got %v", results[1].Topic)
				}
				// Third result should be guild-scoped backend finalization
				if results[2].Topic != "round.finalized.v1.guild-123" {
					t.Errorf("Third result should be guild-scoped finalization, got %v", results[2].Topic)
				}
				// Fourth result should be club-scoped backend finalization
				if results[3].Topic != fmt.Sprintf("%s.%s", roundevents.RoundFinalizedV1, testClubUUID.String()) {
					t.Errorf("Fourth result should be club-scoped finalization, got %v", results[3].Topic)
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-456")

	testParticipants := []roundtypes.Participant{
		{
			UserID:   sharedtypes.DiscordID("user1"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(60)),
		},
		{
			UserID:   sharedtypes.DiscordID("user2"),
			Response: roundtypes.ResponseAccept,
			Score:    scorePointer(sharedtypes.Score(65)),
		},
	}

	testPayload := &roundevents.RoundFinalizedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		RoundData: roundtypes.Round{
			ID:           testRoundID,
			Participants: testParticipants,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundFinalizedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundFinalized with score processing",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID: testRoundID,
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1"), Score: scorePointer(sharedtypes.Score(60))},
							{UserID: sharedtypes.DiscordID("user2"), Score: scorePointer(sharedtypes.Score(65))},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: sharedevents.ProcessRoundScoresRequestedV1,
		},
		{
			name: "Service returns finalization error",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("no participants with scores")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundFinalizationErrorV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("service unavailable")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service unavailable",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Empty participants list",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("no participants")), nil
				}
			},
			payload: &roundevents.RoundFinalizedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: []roundtypes.Participant{},
				},
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundFinalizationErrorV1,
		},
		{
			name: "Multiple scores for processing",
			fakeSetup: func(fake *FakeService) {
				fake.NotifyScoreModuleFunc = func(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID: testRoundID,
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1"), Score: scorePointer(sharedtypes.Score(60))},
							{UserID: sharedtypes.DiscordID("user2"), Score: scorePointer(sharedtypes.Score(65))},
							{UserID: sharedtypes.DiscordID("user3"), Score: scorePointer(sharedtypes.Score(55))},
						},
					}), nil
				}
			},
			payload: &roundevents.RoundFinalizedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID: testRoundID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(60))},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(65))},
						{UserID: sharedtypes.DiscordID("user3"), Response: roundtypes.ResponseAccept, Score: scorePointer(sharedtypes.Score(55))},
					},
				},
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: sharedevents.ProcessRoundScoresRequestedV1,
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
				helpers:     utils.NewHelper(logger),
			}

			ctx := context.Background()
			results, err := h.HandleRoundFinalized(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundFinalized() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundFinalized() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundFinalized() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
