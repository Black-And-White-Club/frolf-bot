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

func TestRoundHandlers_HandleScoreUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(42)

	testPayload := &roundevents.ScoreUpdateRequestPayloadV1{
		GuildID:   sharedtypes.GuildID("test-guild"),
		RoundID:   testRoundID,
		UserID:    testParticipant,
		Score:     &testScore,
		ChannelID: "test-channel",
		MessageID: "test-message",
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScoreUpdateRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScoreUpdateRequest with validation success",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateScoreUpdateRequestFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
					return results.SuccessResult[*roundtypes.ScoreUpdateRequest, error](&roundtypes.ScoreUpdateRequest{
						GuildID: sharedtypes.GuildID("test-guild"),
						RoundID: testRoundID,
						UserID:  testParticipant,
						Score:   &testScore,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateValidatedV1,
		},
		{
			name: "Service returns validation failure",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateScoreUpdateRequestFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
					return results.FailureResult[*roundtypes.ScoreUpdateRequest, error](errors.New("validation failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateErrorV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateScoreUpdateRequestFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
					return results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{}, errors.New("internal service error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "internal service error",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateScoreUpdateRequestFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
					return results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{}, nil
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
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
				logger:      logger,
			}

			ctx := context.Background()
			results, err := h.HandleScoreUpdateRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScoreUpdateRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScoreUpdateRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(50)

	testPayload := &roundevents.ScoreUpdateValidatedPayloadV1{
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:   sharedtypes.GuildID("test-guild"),
			RoundID:   testRoundID,
			UserID:    testParticipant,
			Score:     &testScore,
			ChannelID: "test-channel",
			MessageID: "test-message",
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScoreUpdateValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScoreUpdateValidated",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateParticipantScoreFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error) {
					return results.SuccessResult[*roundtypes.ScoreUpdateResult, error](&roundtypes.ScoreUpdateResult{
						RoundID:        testRoundID,
						GuildID:        sharedtypes.GuildID("test-guild"),
						EventMessageID: "msg-12345",
						UpdatedParticipants: []roundtypes.Participant{
							{UserID: testParticipant, Score: &testScore},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundParticipantScoreUpdatedV1,
		},
		{
			name: "Service returns failure",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateParticipantScoreFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error) {
					return results.FailureResult[*roundtypes.ScoreUpdateResult, error](errors.New("database error")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoreUpdateErrorV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateParticipantScoreFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error) {
					return roundservice.ScoreUpdateResult{}, errors.New("connection failed")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "connection failed",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateParticipantScoreFunc = func(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (roundservice.ScoreUpdateResult, error) {
					return roundservice.ScoreUpdateResult{}, nil
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
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
				logger:      logger,
			}

			ctx := context.Background()
			results, err := h.HandleScoreUpdateValidated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateValidated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScoreUpdateValidated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScoreUpdateValidated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(45)
	testEventMessageID := "msg-12345"

	testPayload := &roundevents.ParticipantScoreUpdatedPayloadV1{
		RoundID:        testRoundID,
		UserID:         testParticipant,
		Score:          testScore,
		ChannelID:      "test-channel",
		EventMessageID: testEventMessageID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name             string
		fakeSetup        func(*FakeService)
		payload          *roundevents.ParticipantScoreUpdatedPayloadV1
		wantErr          bool
		wantResultLen    int
		wantResultTopic  string
		expectedErrMsg   string
		wantResultTopics []string
	}{
		{
			name: "All scores submitted - success path",
			fakeSetup: func(fake *FakeService) {
				fake.CheckAllScoresSubmittedFunc = func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
					return results.SuccessResult[*roundtypes.AllScoresSubmittedResult, error](&roundtypes.AllScoresSubmittedResult{
						IsComplete: true,
						Round:      &roundtypes.Round{ID: testRoundID},
					}), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantResultTopics: []string{
				roundevents.RoundAllScoresSubmittedV1,
			},
		},
		{
			name: "Not all scores submitted yet - partial path",
			fakeSetup: func(fake *FakeService) {
				fake.CheckAllScoresSubmittedFunc = func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
					return results.SuccessResult[*roundtypes.AllScoresSubmittedResult, error](&roundtypes.AllScoresSubmittedResult{
						IsComplete: false,
						Participants: []roundtypes.Participant{
							{UserID: testParticipant, Score: &testScore},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundScoresPartiallySubmittedV1,
		},
		{
			name: "Service returns failure",
			fakeSetup: func(fake *FakeService) {
				fake.CheckAllScoresSubmittedFunc = func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
					return results.FailureResult[*roundtypes.AllScoresSubmittedResult, error](errors.New("round not found")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundFinalizationFailedV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.CheckAllScoresSubmittedFunc = func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
					return roundservice.AllScoresSubmittedResult{}, errors.New("database connection lost")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection lost",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.CheckAllScoresSubmittedFunc = func(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (roundservice.AllScoresSubmittedResult, error) {
					return roundservice.AllScoresSubmittedResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       true,
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
			results, err := h.HandleParticipantScoreUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleParticipantScoreUpdated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 {
				if len(tt.wantResultTopics) > 0 {
					for i, topic := range tt.wantResultTopics {
						if results[i].Topic != topic {
							t.Errorf("HandleParticipantScoreUpdated() result topic[%d] = %v, want %v", i, results[i].Topic, topic)
						}
					}
				} else if tt.wantResultTopic != "" && results[0].Topic != tt.wantResultTopic {
					t.Errorf("HandleParticipantScoreUpdated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
				}
			}
		})
	}
}

func TestRoundHandlers_HandleScoreBulkUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("test-guild")
	testUserID1 := sharedtypes.DiscordID("user-1")
	testUserID2 := sharedtypes.DiscordID("user-2")
	testScore := sharedtypes.Score(60)

	testPayload := &roundevents.ScoreBulkUpdateRequestPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Updates: []roundevents.ScoreUpdateRequestPayloadV1{
			{UserID: testUserID1, Score: &testScore},
			{UserID: testUserID2, Score: &testScore},
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScoreBulkUpdateRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
	}{
		{
			name: "Successfully handle ScoreBulkUpdateRequest",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateParticipantScoresBulkFunc = func(ctx context.Context, req *roundtypes.BulkScoreUpdateRequest) (roundservice.BulkScoreUpdateResult, error) {
					return results.SuccessResult[*roundtypes.BulkScoreUpdateResult, error](&roundtypes.BulkScoreUpdateResult{
						GuildID: testGuildID,
						RoundID: testRoundID,
						Updates: []roundtypes.ScoreUpdateRequest{
							{UserID: testUserID1, Score: &testScore},
							{UserID: testUserID2, Score: &testScore},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   2, // RoundScoresBulkUpdatedV1 + ScoreBulkUpdatedV1
			wantResultTopic: roundevents.RoundScoresBulkUpdatedV1,
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
			results, err := h.HandleScoreBulkUpdateRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreBulkUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScoreBulkUpdateRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScoreBulkUpdateRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			if tt.wantResultLen > 0 {
				_, ok := results[0].Payload.(*roundevents.RoundScoresBulkUpdatedPayloadV1)
				if !ok {
					t.Errorf("HandleScoreBulkUpdateRequest() payload type mismatch, got %T, want *roundevents.RoundScoresBulkUpdatedPayloadV1", results[0].Payload)
				}
			}
		})
	}
}
