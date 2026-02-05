package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	testScoreRoundID     = sharedtypes.RoundID(uuid.New())
	testParticipant      = sharedtypes.DiscordID("user1")
	testScore            = sharedtypes.Score(10)
	testDiscordMessageID = "12345"
)

func TestRoundService_ValidateScoreUpdateRequest(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.ScoreUpdateRequest
		expectedResult results.OperationResult[*roundtypes.ScoreUpdateRequest, error]
		expectedError  error
	}{
		{
			name: "successful validation",
			setup: func(f *FakeRepo) {
				// No DB interactions expected for validation
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{
				Success: ptr(&roundtypes.ScoreUpdateRequest{
					GuildID: guildID,
					RoundID: testScoreRoundID,
					UserID:  testParticipant,
					Score:   &testScore,
				}),
			},
			expectedError: nil,
		},
		{
			name: "invalid round ID",
			setup: func(f *FakeRepo) {
				// No DB interactions expected for validation
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{
				Failure: ptr(errors.New("validation errors: round ID cannot be zero")),
			},
			expectedError: nil,
		},
		{
			name: "empty participant",
			setup: func(f *FakeRepo) {
				// No DB interactions expected for validation
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  "",
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{
				Failure: ptr(errors.New("validation errors: participant Discord ID cannot be empty")),
			},
			expectedError: nil,
		},
		{
			name: "nil score",
			setup: func(f *FakeRepo) {
				// No DB interactions expected for validation
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   nil,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{
				Failure: ptr(errors.New("validation errors: score cannot be empty")),
			},
			expectedError: nil,
		},
		{
			name: "multiple validation errors",
			setup: func(f *FakeRepo) {
				// No DB interactions expected for validation
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  "",
				Score:   nil,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateRequest, error]{
				Failure: ptr(errors.New("validation errors: round ID cannot be zero; participant Discord ID cannot be empty; score cannot be empty")),
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepo{}
			if tt.setup != nil {
				tt.setup(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: &FakeRoundValidator{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.ValidateScoreUpdateRequest(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					successPayload := *result.Success
					expectedSuccessPayload := *tt.expectedResult.Success
					if successPayload.RoundID != expectedSuccessPayload.RoundID {
						t.Errorf("expected RoundID %v, got %v", expectedSuccessPayload.RoundID, successPayload.RoundID)
					}
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					failurePayload := *result.Failure
					expectedFailurePayload := *tt.expectedResult.Failure
					if failurePayload.Error() != expectedFailurePayload.Error() {
						t.Errorf("expected error message %q, got %q", expectedFailurePayload.Error(), failurePayload.Error())
					}
				}
			}
		})
	}
}

func TestRoundService_UpdateParticipantScore(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.ScoreUpdateRequest
		expectedResult results.OperationResult[*roundtypes.ScoreUpdateResult, error]
		expectedError  error
	}{
		{
			name: "successful update",
			setup: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:           r,
						GuildID:      g,
						Participants: []roundtypes.Participant{{UserID: testParticipant}},
					}, nil
				}
				// Updated to mock UpdateParticipant instead of UpdateParticipantScore
				f.UpdateParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{{UserID: testParticipant, Score: &testScore}}, nil
				}
				// GetParticipants is still called to return the full list
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: testParticipant, Score: &testScore},
					}, nil
				}
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateResult, error]{
				Success: ptr(&roundtypes.ScoreUpdateResult{
					RoundID: testScoreRoundID,
					GuildID: guildID,
					UpdatedParticipants: []roundtypes.Participant{
						{UserID: testParticipant, Score: &testScore},
					},
				}),
			},
			expectedError: nil,
		},
		{
			name: "successful auto-join",
			setup: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:           r,
						GuildID:      g,
						Participants: []roundtypes.Participant{}, // Empty participants
					}, nil
				}
				f.UpdateParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error) {
					// Verify auto-join behavior
					if p.Response != roundtypes.ResponseAccept {
						return nil, errors.New("expected ResponseAccept for auto-join")
					}
					return []roundtypes.Participant{p}, nil
				}
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: testParticipant, Score: &testScore, Response: roundtypes.ResponseAccept},
					}, nil
				}
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateResult, error]{
				Success: ptr(&roundtypes.ScoreUpdateResult{
					RoundID: testScoreRoundID,
					GuildID: guildID,
					UpdatedParticipants: []roundtypes.Participant{
						{UserID: testParticipant, Score: &testScore, Response: roundtypes.ResponseAccept},
					},
				}),
			},
			expectedError: nil,
		},
		{
			name: "error updating score",
			setup: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{Participants: []roundtypes.Participant{{UserID: testParticipant}}}, nil
				}
				// Updated to mock UpdateParticipant
				f.UpdateParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error) {
					return nil, errors.New("database error")
				}
			},
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: results.OperationResult[*roundtypes.ScoreUpdateResult, error]{
				Failure: ptr(errors.New("failed to update score in database: database error")),
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepo{}
			if tt.setup != nil {
				tt.setup(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: &FakeRoundValidator{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.UpdateParticipantScore(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedResult.Success != nil && result.Success == nil {
				t.Errorf("expected success result, got failure")
			}
			if tt.expectedResult.Failure != nil && result.Failure == nil {
				t.Errorf("expected failure result, got success")
			}
		})
	}
}

func TestRoundService_CheckAllScoresSubmitted(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.CheckAllScoresSubmittedRequest
		expectedResult results.OperationResult[*roundtypes.AllScoresSubmittedResult, error]
		expectedError  error
	}{
		{
			name: "all scores submitted",
			setup: func(f *FakeRepo) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
					}, nil
				}
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:      testScoreRoundID,
						GuildID: g,
					}, nil
				}
			},
			payload: &roundtypes.CheckAllScoresSubmittedRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: testScoreRoundID,
			},
			expectedResult: results.OperationResult[*roundtypes.AllScoresSubmittedResult, error]{
				Success: ptr(&roundtypes.AllScoresSubmittedResult{
					IsComplete: true,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
					},
					Round: &roundtypes.Round{
						ID:      testScoreRoundID,
						GuildID: sharedtypes.GuildID("guild-123"),
					},
				}),
			},
			expectedError: nil,
		},
		{
			name: "not all scores submitted",
			setup: func(f *FakeRepo) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: nil},
					}, nil
				}
			},
			payload: &roundtypes.CheckAllScoresSubmittedRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: testScoreRoundID,
			},
			expectedResult: results.OperationResult[*roundtypes.AllScoresSubmittedResult, error]{
				Success: ptr(&roundtypes.AllScoresSubmittedResult{
					IsComplete: false,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: nil},
					},
					Round: nil,
				}),
			},
			expectedError: nil,
		},
		{
			name: "error checking if all scores submitted",
			setup: func(f *FakeRepo) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return nil, errors.New("database error")
				}
			},
			payload: &roundtypes.CheckAllScoresSubmittedRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: testScoreRoundID,
			},
			expectedResult: results.OperationResult[*roundtypes.AllScoresSubmittedResult, error]{
				Failure: ptr(errors.New("database error")),
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepo{}
			if tt.setup != nil {
				tt.setup(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: &FakeRoundValidator{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.CheckAllScoresSubmitted(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedResult.Success != nil && result.Success == nil {
				t.Errorf("expected success result, got failure")
			}
			if tt.expectedResult.Failure != nil && result.Failure == nil {
				t.Errorf("expected failure result, got success")
			}
		})
	}
}
