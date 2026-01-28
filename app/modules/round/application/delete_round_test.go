package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ValidateRoundDeletion(t *testing.T) {
	ctx := context.Background()

	validRoundID := sharedtypes.RoundID(uuid.New())
	validGuildID := sharedtypes.GuildID("guild-1")
	validUserID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name           string
		setupFake      func(*FakeRepo)
		input          *roundtypes.DeleteRoundInput
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo)
	}{
		{
			name: "success - valid request",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:        roundID,
						GuildID:   guildID,
						CreatedBy: validUserID,
					}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				if res.Success == nil {
					t.Fatal("expected success result")
				}
				if (*res.Success).ID != validRoundID {
					t.Errorf("expected round ID %s, got %s", validRoundID, (*res.Success).ID)
				}
			},
		},
		{
			name: "failure - round ID is nil",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  validUserID,
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "round ID cannot be zero" {
					t.Errorf("expected error 'round ID cannot be zero', got %v", res.Failure)
				}
			},
		},
		{
			name: "failure - user ID is empty",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  "",
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "requesting user's Discord ID cannot be empty" {
					t.Errorf("expected error 'requesting user's Discord ID cannot be empty', got %v", res.Failure)
				}
			},
		},
		{
			name: "failure - round not found",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("db error")
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || !strings.Contains((*res.Failure).Error(), "round not found") {
					t.Errorf("expected error containing 'round not found', got %v", res.Failure)
				}
			},
		},
		{
			name: "failure - unauthorized",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:        roundID,
						GuildID:   guildID,
						CreatedBy: "other-user",
					}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "unauthorized: only the round creator can delete the round" {
					t.Errorf("expected error 'unauthorized: only the round creator can delete the round', got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeRepo()
			if tt.setupFake != nil {
				tt.setupFake(repo)
			}

			s := &RoundService{
				repo:           repo,
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				parserFactory:  &StubFactory{},
			}

			res, err := s.ValidateRoundDeletion(ctx, tt.input)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, repo)
			}
		})
	}
}

func TestRoundService_DeleteRound(t *testing.T) {
	ctx := context.Background()
	validRoundID := sharedtypes.RoundID(uuid.New())
	validGuildID := sharedtypes.GuildID("guild-1")
	validUserID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name           string
		setupFake      func(*FakeRepo, *FakeQueueService)
		input          *roundtypes.DeleteRoundInput
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[bool, error], infraErr error, fakeRepo *FakeRepo, fakeQueue *FakeQueueService)
	}{
		{
			name: "success - round deleted and jobs cancelled",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo, q *FakeQueueService) {
				r.DeleteRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) error {
					return nil
				}
				q.CancelRoundJobsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) error {
					return nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error, fakeRepo *FakeRepo, fakeQueue *FakeQueueService) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				if !res.IsSuccess() || *res.Success != true {
					t.Fatal("expected success result true")
				}

				trace := fakeRepo.Trace()
				foundDelete := false
				for _, call := range trace {
					if call == "DeleteRound" {
						foundDelete = true
						break
					}
				}
				if !foundDelete {
					t.Error("expected DeleteRound to be called on repo")
				}

				queueTrace := fakeQueue.Trace()
				foundCancel := false
				for _, call := range queueTrace {
					if call == "CancelRoundJobs" {
						foundCancel = true
						break
					}
				}
				if !foundCancel {
					t.Error("expected CancelRoundJobs to be called on queue service")
				}
			},
		},
		{
			name: "failure - nil round ID",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  validUserID,
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error, fakeRepo *FakeRepo, fakeQueue *FakeQueueService) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "round ID cannot be nil" {
					t.Errorf("expected error 'round ID cannot be nil', got %v", res.Failure)
				}
			},
		},
		{
			name: "failure - db delete error",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo, q *FakeQueueService) {
				r.DeleteRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) error {
					return errors.New("db error")
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error, fakeRepo *FakeRepo, fakeQueue *FakeQueueService) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || !strings.Contains((*res.Failure).Error(), "failed to delete round from database") {
					t.Errorf("expected error containing 'failed to delete round from database', got %v", res.Failure)
				}
			},
		},
		{
			name: "success - jobs cancellation fails but round deleted",
			input: &roundtypes.DeleteRoundInput{
				GuildID: validGuildID,
				RoundID: validRoundID,
				UserID:  validUserID,
			},
			setupFake: func(r *FakeRepo, q *FakeQueueService) {
				r.DeleteRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) error {
					return nil
				}
				q.CancelRoundJobsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) error {
					return errors.New("queue error")
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error, fakeRepo *FakeRepo, fakeQueue *FakeQueueService) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if !res.IsSuccess() || *res.Success != true {
					t.Fatal("expected success result true even if queue cancellation fails")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeRepo()
			queue := NewFakeQueueService()
			if tt.setupFake != nil {
				tt.setupFake(repo, queue)
			}

			s := &RoundService{
				repo:         repo,
				queueService: queue,
				logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:      &roundmetrics.NoOpMetrics{},
				tracer:       noop.NewTracerProvider().Tracer("test"),
				parserFactory: &StubFactory{},
			}

			res, err := s.DeleteRound(ctx, tt.input)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, repo, queue)
			}
		})
	}
}
