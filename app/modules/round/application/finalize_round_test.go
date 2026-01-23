package roundservice

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_FinalizeRound(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name       string
		setupRepo  func(f *FakeRepo)
		payload    roundevents.AllScoresSubmittedPayloadV1
		assertFunc func(t *testing.T, res results.OperationResult)
	}{
		{
			name: "success",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
				}
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:           r,
						GuildID:      g,
						Participants: []roundtypes.Participant{},
					}, nil
				}
			},
			payload: roundevents.AllScoresSubmittedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				payload := res.Success.(*roundevents.RoundFinalizedPayloadV1)
				if payload.RoundID != roundID {
					t.Errorf("expected roundID %v, got %v", roundID, payload.RoundID)
				}
			},
		},
		{
			name: "fail update round state",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error {
					return errors.New("db error")
				}
			},
			payload: roundevents.AllScoresSubmittedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				if res.Failure.(*roundevents.RoundFinalizationErrorPayloadV1).RoundID != roundID {
					t.Errorf("unexpected roundID: %v", res.Failure.(*roundevents.RoundFinalizationErrorPayloadV1).RoundID)
				}
			},
		},
		{
			name: "fail get round after state update",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
				}
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("db get error")
				}
			},
			payload: roundevents.AllScoresSubmittedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				if res.Failure.(*roundevents.RoundFinalizationErrorPayloadV1).RoundID != roundID {
					t.Errorf("unexpected roundID: %v", res.Failure.(*roundevents.RoundFinalizationErrorPayloadV1).RoundID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			s := &RoundService{
				repo:    repo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			res, err := s.FinalizeRound(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.assertFunc(t, res)
		})
	}
}

func TestRoundService_NotifyScoreModule(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name       string
		payload    roundevents.RoundFinalizedPayloadV1
		assertFunc func(t *testing.T, res results.OperationResult)
	}{
		{
			name: "success singles",
			payload: roundevents.RoundFinalizedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
				RoundData: roundtypes.Round{
					ID:      roundID,
					GuildID: guildID,
					Participants: []roundtypes.Participant{
						{UserID: "user1", Score: ptrScore(3), TagNumber: ptrTag(1)},
						{UserID: "user2", Score: ptrScore(5), TagNumber: ptrTag(2)},
					},
				},
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				payload := res.Success.(*sharedevents.ProcessRoundScoresRequestedPayloadV1)
				if len(payload.Scores) != 2 {
					t.Errorf("expected 2 scores, got %d", len(payload.Scores))
				}
			},
		},
		{
			name: "failure no scores",
			payload: roundevents.RoundFinalizedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
				RoundData: roundtypes.Round{
					ID:           roundID,
					GuildID:      guildID,
					Participants: []roundtypes.Participant{},
				},
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RoundService{
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			res, err := s.NotifyScoreModule(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.assertFunc(t, res)
		})
	}
}

// --- helpers ---
func ptrScore(v int) *sharedtypes.Score   { s := sharedtypes.Score(v); return &s }
func ptrTag(v int) *sharedtypes.TagNumber { t := sharedtypes.TagNumber(v); return &t }
