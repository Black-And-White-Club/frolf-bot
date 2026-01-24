package roundservice

import (
	"context"
	"fmt"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ApplyImportedScores(t *testing.T) {
	ctx := context.Background()

	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-1")
	importID := "imp-apply-1"

	u1 := sharedtypes.DiscordID("user-1")
	u2 := sharedtypes.DiscordID("user-2")

	score1 := sharedtypes.Score(54)
	score2 := sharedtypes.Score(56)

	tests := []struct {
		name          string
		payload       roundevents.ImportCompletedPayloadV1
		setupRepo     func(r *FakeRepo)
		expectSuccess bool
		expectedError string
	}{
		{
			name: "success singles - all scores applied",
			payload: roundevents.ImportCompletedPayloadV1{
				GuildID:   guildID,
				RoundID:   roundID,
				ImportID:  importID,
				RoundMode: sharedtypes.RoundModeSingles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: u1, Score: score1},
					{UserID: u2, Score: score2},
				},
			},
			setupRepo: func(r *FakeRepo) {
				r.UpdateParticipantScoreFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score) error {
					return nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: u1, Score: &score1},
						{UserID: u2, Score: &score2},
					}, nil
				}
				r.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{EventMessageID: "msg-123"}, nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "partial success singles - one fails but still succeeds",
			payload: roundevents.ImportCompletedPayloadV1{
				GuildID:   guildID,
				RoundID:   roundID,
				ImportID:  importID,
				RoundMode: sharedtypes.RoundModeSingles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: u1, Score: score1},
					{UserID: u2, Score: score2},
				},
			},
			setupRepo: func(r *FakeRepo) {
				r.UpdateParticipantScoreFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score) error {
					if uID == u2 {
						return fmt.Errorf("db error")
					}
					return nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: u1, Score: &score1},
					}, nil
				}
				r.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{}, nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "singles - unmatched users added and persisted",
			payload: roundevents.ImportCompletedPayloadV1{
				GuildID:   guildID,
				RoundID:   roundID,
				ImportID:  importID,
				RoundMode: sharedtypes.RoundModeSingles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: u1, Score: score1},
				},
			},
			setupRepo: func(r *FakeRepo) {
				// Simulate repository-level update error (legacy behavior) but
				// Add/append path and batch persist should succeed by default.
				r.UpdateParticipantScoreFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score) error {
					return fmt.Errorf("db error")
				}
				r.GetParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "success doubles - batch update and completion check",
			payload: roundevents.ImportCompletedPayloadV1{
				GuildID:   guildID,
				RoundID:   roundID,
				ImportID:  importID,
				RoundMode: sharedtypes.RoundModeDoubles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: u1, Score: score1},
				},
			},
			setupRepo: func(r *FakeRepo) {
				r.GetParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{{UserID: u1}}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, rID sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, u []roundtypes.RoundUpdate) error {
					return nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "failure doubles - get participants error",
			payload: roundevents.ImportCompletedPayloadV1{
				GuildID:   guildID,
				RoundID:   roundID,
				ImportID:  importID,
				RoundMode: sharedtypes.RoundModeDoubles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: u1, Score: score1},
				},
			},
			setupRepo: func(r *FakeRepo) {
				r.GetParticipantsFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return nil, fmt.Errorf("fatal db error")
				}
			},
			expectSuccess: false,
			expectedError: "fatal db error",
		},

		{
			name:    "failure doubles - round has groups error",
			payload: doublesPayload(guildID, roundID, importID, u1, score1),
			setupRepo: func(r *FakeRepo) {
				r.GetParticipantsFunc = okParticipants
				r.RoundHasGroupsFunc = func(ctx context.Context, _ sharedtypes.RoundID) (bool, error) {
					return false, fmt.Errorf("group lookup failed")
				}
			},
			expectSuccess: false,
			expectedError: "group lookup failed",
		},

		{
			name:    "failure doubles - create groups error",
			payload: doublesPayload(guildID, roundID, importID, u1, score1),
			setupRepo: func(r *FakeRepo) {
				r.GetParticipantsFunc = okParticipants
				r.RoundHasGroupsFunc = func(ctx context.Context, _ sharedtypes.RoundID) (bool, error) {
					return false, nil
				}
				r.CreateRoundGroupsFunc = func(_ sharedtypes.RoundID, _ []roundtypes.Participant) error {
					return fmt.Errorf("create groups failed")
				}
			},
			expectSuccess: false,
			expectedError: "create groups failed",
		},

		{
			name:    "failure doubles - batch update error",
			payload: doublesPayload(guildID, roundID, importID, u1, score1),
			setupRepo: func(r *FakeRepo) {
				r.GetParticipantsFunc = okParticipants
				r.RoundHasGroupsFunc = func(ctx context.Context, _ sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, _ sharedtypes.GuildID, _ []roundtypes.RoundUpdate) error {
					return fmt.Errorf("batch update failed")
				}
			},
			expectSuccess: false,
			expectedError: "batch update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			svc := &RoundService{
				repo:    repo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			result, err := svc.ApplyImportedScores(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectSuccess {
				if result.Success == nil {
					t.Fatalf("expected success, got failure: %+v", result.Failure)
				}
			} else {
				if result.Failure == nil {
					t.Fatalf("expected failure, got success")
				}

				var msg string
				switch f := result.Failure.(type) {
				case *roundevents.ImportFailedPayloadV1:
					msg = f.Error
				case *roundevents.RoundErrorPayloadV1:
					msg = f.Error
				}

				if tt.expectedError != "" && !strings.Contains(msg, tt.expectedError) {
					t.Fatalf("expected error containing %q, got %q", tt.expectedError, msg)
				}
			}
		})
	}
}

func doublesPayload(guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, userID sharedtypes.DiscordID, score sharedtypes.Score) roundevents.ImportCompletedPayloadV1 {
	return roundevents.ImportCompletedPayloadV1{
		GuildID:   guildID,
		RoundID:   roundID,
		ImportID:  importID,
		RoundMode: sharedtypes.RoundModeDoubles,
		Scores: []sharedtypes.ScoreInfo{
			{UserID: userID, Score: score},
		},
	}
}

func okParticipants(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID,
) ([]roundtypes.Participant, error) {
	return []roundtypes.Participant{
		{
			UserID:   sharedtypes.DiscordID("user-1"),
			Response: roundtypes.ResponseAccept,
		},
	}, nil
}
