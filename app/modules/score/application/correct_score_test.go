package scoreservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestScoreService_CorrectScore(t *testing.T) {
	ctx := context.Background()
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &scoremetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		userID         sharedtypes.DiscordID
		score          sharedtypes.Score
		tagNumber      *sharedtypes.TagNumber
		setupFake      func(*FakeScoreRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res ScoreOperationResult, infraErr error, fake *FakeScoreRepository)
	}{
		{
			name:      "success - corrects score and preserves existing tag",
			userID:    testUserID,
			score:     testScore,
			tagNumber: nil,
			setupFake: func(f *FakeScoreRepository) {
				existingTag := sharedtypes.TagNumber(7)
				f.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{{UserID: testUserID, Score: 12, TagNumber: &existingTag}}, nil
				}
				f.UpdateOrAddScoreFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID, si sharedtypes.ScoreInfo) error {
					return nil
				}
			},
			verify: func(t *testing.T, res ScoreOperationResult, infraErr error, fake *FakeScoreRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Success == nil {
					t.Fatal("expected success result")
				}

				// Directly access fields
				if res.Success.Score != testScore {
					t.Errorf("expected score %d, got %d", testScore, res.Success.Score)
				}

				if res.Success.TagNumber == nil || *res.Success.TagNumber != 7 {
					t.Errorf("expected preserved tag 7, got %v", res.Success.TagNumber)
				}
			},
		},
		{
			name:      "domain failure - invalid score range",
			userID:    testUserID,
			score:     sharedtypes.Score(99), // Invalid > 72
			tagNumber: &testTag,
			verify: func(t *testing.T, res ScoreOperationResult, infraErr error, fake *FakeScoreRepository) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidScore) {
					t.Errorf("expected ErrInvalidScore, got %v", res.Failure)
				}
				if len(fake.Trace()) > 0 {
					t.Errorf("repo should not be called for invalid domain input")
				}
			},
		},
		{
			name:      "infra failure - database error on update",
			userID:    testUserID,
			score:     testScore,
			tagNumber: &testTag,
			setupFake: func(f *FakeScoreRepository) {
				f.UpdateOrAddScoreFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID, si sharedtypes.ScoreInfo) error {
					return errors.New("db connection lost")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res ScoreOperationResult, infraErr error, fake *FakeScoreRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "db connection lost") {
					t.Errorf("expected infra error 'db connection lost', got %v", infraErr)
				}
			},
		},
		{
			name:      "success - update with new tag number",
			userID:    testUserID,
			score:     testScore,
			tagNumber: &testTag,
			setupFake: func(f *FakeScoreRepository) {
				f.UpdateOrAddScoreFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID, si sharedtypes.ScoreInfo) error {
					return nil
				}
			},
			verify: func(t *testing.T, res ScoreOperationResult, infraErr error, fake *FakeScoreRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected error: %v", infraErr)
				}
				// 1. Ensure Success is not nil
				if res.Success == nil {
					t.Fatal("expected success result, got nil")
				}
				// 2. Access directly (no type assertion needed because it's already *ScoreInfo)
				success := res.Success

				if success.TagNumber == nil {
					t.Fatal("expected tag number to be set, but it was nil")
				}
				if *success.TagNumber != testTag {
					t.Errorf("expected tag %d, got %d", testTag, *success.TagNumber)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeScoreRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &ScoreService{
				repo:    fakeRepo,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			res, err := s.CorrectScore(ctx, testGuildID, testRoundID, tt.userID, tt.score, tt.tagNumber)

			if tt.expectInfraErr && err == nil {
				t.Error("expected infrastructure error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Errorf("unexpected infrastructure error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}
