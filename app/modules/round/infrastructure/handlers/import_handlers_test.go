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
		name         string
		setupService func(s *FakeService)
		expectTopic  string
		expectError  bool
	}{
		{
			name: "Success - Routes to Parse Requested",
			setupService: func(s *FakeService) {
				s.CreateImportJobFn = func(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{Success: &p}, nil
				}
			},
			expectTopic: string(roundevents.ScorecardParseRequestedV1),
		},
		{
			name: "Failure - Service Returns Error",
			setupService: func(s *FakeService) {
				s.CreateImportJobFn = func(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{}, errors.New("db down")
				}
			},
			expectError: true,
		},
		{
			name: "Failure - Service Returns Failure Result",
			setupService: func(s *FakeService) {
				s.CreateImportJobFn = func(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{Failure: &roundevents.ImportFailedPayloadV1{}}, nil
				}
			},
			expectTopic: string(roundevents.ImportFailedV1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewFakeService()
			tt.setupService(svc)
			h := &RoundHandlers{service: svc}

			results, err := h.HandleScorecardUploaded(ctx, payload)

			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
			if !tt.expectError && results[0].Topic != tt.expectTopic {
				t.Errorf("expected topic %s, got %s", tt.expectTopic, results[0].Topic)
			}
		})
	}
}

func TestRoundHandlers_HandleImportCompleted_SingleTrigger(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	initiatorID := sharedtypes.DiscordID("admin-user")
	payload := &roundevents.ImportCompletedPayloadV1{
		RoundID: roundID,
		UserID:  initiatorID,
	}

	svc := NewFakeService()
	svc.ApplyImportedScoresFn = func(ctx context.Context, p roundevents.ImportCompletedPayloadV1) (results.OperationResult, error) {
		s1, s2 := sharedtypes.Score(-5), sharedtypes.Score(2)
		return results.OperationResult{
			Success: &roundevents.ImportScoresAppliedPayloadV1{
				GuildID: p.GuildID,
				RoundID: p.RoundID,
				Participants: []roundtypes.Participant{
					{UserID: "player-1", Score: &s1},
					{UserID: "player-2", Score: &s2},
				},
			},
		}, nil
	}

	h := &RoundHandlers{
		service: svc,
		logger:  loggerfrolfbot.NoOpLogger, // Use NoOp or a mock for tests
	}

	res, err := h.HandleImportCompleted(ctx, payload)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Errorf("expected 1 trigger result, got %d", len(res))
	}

	if res[0].Topic != string(roundevents.RoundParticipantScoreUpdatedV1) {
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
	svc.ParseScorecardFn = func(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1, fd []byte) (results.OperationResult, error) {
		// Verify the handler actually passed the FileData from payload to service
		if len(fd) == 0 {
			return results.OperationResult{}, errors.New("no file data passed")
		}
		return results.OperationResult{Success: &roundevents.ParsedScorecardPayloadV1{}}, nil
	}

	h := &RoundHandlers{service: svc, logger: loggerfrolfbot.NoOpLogger}
	res, err := h.HandleParseScorecardRequest(ctx, payload)

	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if res[0].Topic != string(roundevents.ScorecardParsedForNormalizationV1) {
		t.Errorf("wrong topic: %s", res[0].Topic)
	}

	// Verify trace
	if svc.Trace()[0] != "ParseScorecard" {
		t.Errorf("service method not called, trace: %v", svc.Trace())
	}
}
