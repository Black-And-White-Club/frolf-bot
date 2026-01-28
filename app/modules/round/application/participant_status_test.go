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
	testUserID         = sharedtypes.DiscordID("user1")
	testEventMessageID = "discord_message_id_123"
	testTagNumber      = sharedtypes.TagNumber(1)
	joinedLateFalse    = false
)

func TestRoundService_CheckParticipantStatus(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("test-user-123")
	guildID := sharedtypes.GuildID("guild-123")

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.JoinRoundRequest
		expectedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]
		expectedError  error
	}{
		{
			name: "new participant joining",
			setup: func(f *FakeRepo) {
				f.GetParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) (*roundtypes.Participant, error) {
					return nil, nil
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{
				Success: ptr(&roundtypes.ParticipantStatusCheckResult{
					Action:        "VALIDATE",
					CurrentStatus: "",
					RoundID:       testRoundID,
					UserID:        testUserID,
					Response:      roundtypes.ResponseAccept,
					GuildID:       guildID,
				}),
			},
			expectedError: nil,
		},
		{
			name: "participant changing status",
			setup: func(f *FakeRepo) {
				f.GetParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) (*roundtypes.Participant, error) {
					return &roundtypes.Participant{
						UserID:   testUserID,
						Response: roundtypes.ResponseTentative,
					}, nil
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept, // Different from existing status
			},
			expectedResult: results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{
				Success: ptr(&roundtypes.ParticipantStatusCheckResult{
					Action:        "VALIDATE",
					CurrentStatus: "TENTATIVE",
					RoundID:       testRoundID,
					UserID:        testUserID,
					Response:      roundtypes.ResponseAccept,
					GuildID:       guildID,
				}),
			},
			expectedError: nil,
		},
		{
			name: "toggle participant status (same status clicked)",
			setup: func(f *FakeRepo) {
				f.GetParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) (*roundtypes.Participant, error) {
					return &roundtypes.Participant{
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					}, nil
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept, // Same as existing status
			},
			expectedResult: results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{
				Success: ptr(&roundtypes.ParticipantStatusCheckResult{
					Action:        "REMOVE",
					CurrentStatus: "ACCEPT",
					RoundID:       testRoundID,
					UserID:        testUserID,
					Response:      roundtypes.ResponseAccept,
					GuildID:       guildID,
				}),
			},
			expectedError: nil,
		},
		{
			name: "failure checking participant status",
			setup: func(f *FakeRepo) {
				f.GetParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) (*roundtypes.Participant, error) {
					return nil, errors.New("db error")
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{
				Failure: ptr(errors.New("failed to get participant status: db error")),
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
				eventBus:       &FakeEventBus{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.CheckParticipantStatus(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					actual := *result.Success
					expected := *tt.expectedResult.Success

					if actual.Action != expected.Action {
						t.Errorf("expected Action %s, got %s", expected.Action, actual.Action)
					}
					if actual.CurrentStatus != expected.CurrentStatus {
						t.Errorf("expected CurrentStatus %s, got %s", expected.CurrentStatus, actual.CurrentStatus)
					}
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					if (*result.Failure).Error() != (*tt.expectedResult.Failure).Error() {
						t.Errorf("expected failure error %q, got %q", (*tt.expectedResult.Failure).Error(), (*result.Failure).Error())
					}
				}
			}
		})
	}
}

func TestRoundService_ValidateParticipantJoinRequest(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")
	guildID := sharedtypes.GuildID("guild-123")

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.JoinRoundRequest
		expectedResult results.OperationResult[*roundtypes.JoinRoundRequest, error]
		expectedError  error
	}{
		{
			name: "success validating participant join request",
			setup: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: testRoundID, State: roundtypes.RoundStateUpcoming}, nil
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: results.OperationResult[*roundtypes.JoinRoundRequest, error]{
				Success: ptr(&roundtypes.JoinRoundRequest{
					RoundID:    testRoundID,
					GuildID:    guildID,
					UserID:     user1ID,
					Response:   roundtypes.ResponseAccept,
					JoinedLate: &joinedLateFalse,
				}),
			},
			expectedError: nil,
		},
		{
			name: "failure validating participant join request",
			setup: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("db error")
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID:  testRoundID,
				GuildID:  guildID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: results.OperationResult[*roundtypes.JoinRoundRequest, error]{
				Failure: ptr(errors.New("failed to fetch round details: db error")),
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
				eventBus:       &FakeEventBus{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.ValidateParticipantJoinRequest(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					if (*result.Success).JoinedLate == nil || *(*result.Success).JoinedLate != *(*tt.expectedResult.Success).JoinedLate {
						t.Errorf("expected JoinedLate %v, got %v", *(*tt.expectedResult.Success).JoinedLate, (*result.Success).JoinedLate)
					}
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					if (*result.Failure).Error() != (*tt.expectedResult.Failure).Error() {
						t.Errorf("expected failure error %q, got %q", (*tt.expectedResult.Failure).Error(), (*result.Failure).Error())
					}
				}
			}
		})
	}
}

func TestRoundService_ParticipantRemoval(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name           string
		setup          func(*FakeRepo)
		payload        *roundtypes.JoinRoundRequest
		expectedResult results.OperationResult[*roundtypes.Round, error]
		expectedError  error
	}{
		{
			name: "success removing participant",
			setup: func(f *FakeRepo) {
				f.RemoveParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					}, nil
				}
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						EventMessageID: testEventMessageID,
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
						},
					}, nil
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID: testRoundID,
				GuildID: sharedtypes.GuildID("guild-123"),
				UserID:  testUserID,
			},
			expectedResult: results.OperationResult[*roundtypes.Round, error]{
				Success: ptr(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}),
			},
			expectedError: nil,
		},
		{
			name: "failure removing participant",
			setup: func(f *FakeRepo) {
				f.RemoveParticipantFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, u sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
					return nil, errors.New("db error")
				}
			},
			payload: &roundtypes.JoinRoundRequest{
				RoundID: testRoundID,
				GuildID: sharedtypes.GuildID("guild-123"),
				UserID:  testUserID,
			},
			expectedResult: results.OperationResult[*roundtypes.Round, error]{
				Failure: ptr(errors.New("failed to remove participant: db error")),
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
				repo:          fakeRepo,
				logger:        logger,
				metrics:       mockMetrics,
				tracer:        tracer,
				parserFactory: &StubFactory{},
			}

			result, err := s.ParticipantRemoval(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					if len((*result.Success).Participants) != len((*tt.expectedResult.Success).Participants) {
						t.Errorf("expected %d participants, got %d", len((*tt.expectedResult.Success).Participants), len((*result.Success).Participants))
					}
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					if (*result.Failure).Error() != (*tt.expectedResult.Failure).Error() {
						t.Errorf("expected failure error %q, got %q", (*tt.expectedResult.Failure).Error(), (*result.Failure).Error())
					}
				}
			}
		})
	}
}
