package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_StartRound(t *testing.T) {
	var (
		testGuildID    = sharedtypes.GuildID("guild-123")
		testRoundID    = sharedtypes.RoundID(uuid.New())
		testEventMsgID = "event-msg-456"
		testStartTime  = sharedtypes.StartTime(time.Now())
		testRoundTitle = roundtypes.Title("Test Round")
		testLocation   = roundtypes.Location("Test Location")
	)

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name          string
		setup         func(*FakeRepo)
		req           *roundtypes.StartRoundRequest
		expectedState roundtypes.RoundState
		expectNoOp    bool
		expectError   bool
		expectFailure bool
	}{
		{
			name: "successful start",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						State:          roundtypes.RoundStateUpcoming,
						EventMessageID: testEventMsgID,
						Title:          testRoundTitle,
						Location:       testLocation,
						StartTime:      &testStartTime,
					}, nil
				}
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {
					if id != testRoundID {
						return errors.New("wrong round id")
					}
					if state != roundtypes.RoundStateInProgress {
						return errors.New("wrong state")
					}
					return nil
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectedState: roundtypes.RoundStateInProgress,
			expectError:   false,
		},
		{
			name: "successful start without event message id (pwa-only)",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						State:          roundtypes.RoundStateUpcoming,
						EventMessageID: "", // Missing
						Title:          testRoundTitle,
						Location:       testLocation,
						StartTime:      &testStartTime,
					}, nil
				}
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {
					if state != roundtypes.RoundStateInProgress {
						return errors.New("wrong state")
					}
					return nil
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectedState: roundtypes.RoundStateInProgress,
			expectError:   false,
		},
		{
			name: "already started returns no-op success",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						State:          roundtypes.RoundStateInProgress,
						EventMessageID: testEventMsgID,
						Title:          testRoundTitle,
						Location:       testLocation,
						StartTime:      &testStartTime,
					}, nil
				}
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {
					t.Fatal("UpdateRoundState should not be called for an already started round")
					return nil
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectedState: roundtypes.RoundStateInProgress,
			expectNoOp:    true,
			expectError:   false,
		},
		{
			name: "failure - finalized round cannot be started",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						State:          roundtypes.RoundStateFinalized,
						EventMessageID: testEventMsgID,
					}, nil
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectError:   false,
			expectFailure: true,
		},
		{
			name: "failure - get round error",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("db error")
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectError:   false,
			expectFailure: true,
		},
		{
			name: "failure - update state error",
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						State:          roundtypes.RoundStateUpcoming,
						EventMessageID: testEventMsgID,
					}, nil
				}
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {
					return errors.New("update error")
				}
			},
			req: &roundtypes.StartRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
			},
			expectError:   false,
			expectFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setup != nil {
				tt.setup(repo)
			}

			s := NewRoundService(repo, nil, nil, nil, mockMetrics, nil, logger, tracer, nil, nil)

			result, err := s.StartRound(context.Background(), tt.req)

			if (err != nil) != tt.expectError {
				t.Errorf("StartRound() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectFailure {
				if result.Failure == nil {
					t.Errorf("StartRound() expected failure result, got success")
				}
			} else {
				if result.Success == nil {
					t.Errorf("StartRound() expected success result, got failure")
				} else {
					round := *result.Success
					if round.State != tt.expectedState {
						t.Errorf("StartRound() expected state %v, got %v", tt.expectedState, round.State)
					}
				}
				if result.AlreadyStarted != tt.expectNoOp {
					t.Errorf("StartRound() AlreadyStarted = %v, want %v", result.AlreadyStarted, tt.expectNoOp)
				}
			}
		})
	}
}
