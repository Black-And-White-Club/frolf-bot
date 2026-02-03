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

func TestRoundHandlers_HandleScorecardUploaded(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	payload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID: "imp-123",
		RoundID:  roundID,
	}

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScorecardUploadedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Success - Routes to Parse Requested",
			fakeSetup: func(fake *FakeService) {
				fake.CreateImportJobFunc = func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
					return results.SuccessResult[roundtypes.CreateImportJobResult, error](roundtypes.CreateImportJobResult{
						Job: &roundtypes.ImportCreateJobInput{
							ImportID: "imp-123",
							RoundID:  roundID,
						},
					}), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScorecardParseRequestedV1,
		},
		{
			name: "Failure - Service Returns Error",
			fakeSetup: func(fake *FakeService) {
				fake.CreateImportJobFunc = func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
					return roundservice.CreateImportJobResult{}, errors.New("db down")
				}
			},
			payload:        payload,
			wantErr:        true,
			expectedErrMsg: "db down",
		},
		{
			name: "Failure - Service Returns Failure Result",
			fakeSetup: func(fake *FakeService) {
				fake.CreateImportJobFunc = func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
					return results.FailureResult[roundtypes.CreateImportJobResult, error](errors.New("validation error")), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
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
				logger:      loggerfrolfbot.NoOpLogger,
			}

			results, err := h.HandleScorecardUploaded(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleScorecardUploaded() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScorecardUploaded() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScorecardUploaded() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScorecardUploaded() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleImportCompleted_SingleTrigger(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-123")
	initiatorID := sharedtypes.DiscordID("admin-user")
	testClubUUID := uuid.New()

	payload := &roundevents.ImportCompletedPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
		UserID:  initiatorID,
		Scores: []sharedtypes.ScoreInfo{
			{UserID: "player-1", Score: -5},
			{UserID: "player-2", Score: 2},
		},
	}

	svc := NewFakeService()
	svc.ApplyImportedScoresFunc = func(ctx context.Context, req roundtypes.ImportApplyScoresInput) (roundservice.ApplyImportedScoresResult, error) {
		s1, s2 := sharedtypes.Score(-5), sharedtypes.Score(2)
		return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
			GuildID: req.GuildID,
			RoundID: req.RoundID,
			Participants: []roundtypes.Participant{
				{UserID: "player-1", Score: &s1},
				{UserID: "player-2", Score: &s2},
			},
		}), nil
	}

	userService := NewFakeUserService()
	userService.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
		return testClubUUID, nil
	}

	h := &RoundHandlers{
		service:     svc,
		userService: userService,
		logger:      loggerfrolfbot.NoOpLogger,
	}

	res, err := h.HandleImportCompleted(ctx, payload)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect 4 results now: RPSU, RASS, Guild-scoped RPSU, Club-scoped RPSU
	if len(res) != 4 {
		t.Errorf("expected 4 trigger results, got %d", len(res))
	}

	if res[0].Topic != roundevents.RoundParticipantScoreUpdatedV1 {
		t.Errorf("expected Topic ScoreUpdated, got %s", res[0].Topic)
	}
}

func TestRoundHandlers_HandleParseScorecardRequest(t *testing.T) {
	ctx := context.Background()
	payload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID: "imp-1",
		FileData: []byte("test-data"),
	}

	svc := NewFakeService()
	svc.ParseScorecardFunc = func(ctx context.Context, req *roundtypes.ImportParseScorecardInput) (roundservice.ParseScorecardResult, error) {
		// Verify the handler actually passed the FileData from payload to service
		if len(req.FileData) == 0 {
			return roundservice.ParseScorecardResult{}, errors.New("no file data passed")
		}
		return results.SuccessResult[roundtypes.ParsedScorecard, error](roundtypes.ParsedScorecard{}), nil
	}

	h := &RoundHandlers{
		service:     svc,
		userService: NewFakeUserService(),
		logger:      loggerfrolfbot.NoOpLogger,
	}
	res, err := h.HandleParseScorecardRequest(ctx, payload)

	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if res[0].Topic != roundevents.ScorecardParsedForNormalizationV1 {
		t.Errorf("wrong topic: %s", res[0].Topic)
	}

	// Verify trace
	if svc.Trace()[0] != "ParseScorecard" {
		t.Errorf("service method not called, trace: %v", svc.Trace())
	}
}
