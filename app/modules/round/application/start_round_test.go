package roundservice

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
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
	testStartRoundID        = sharedtypes.RoundID(uuid.New())
	testRoundTitle          = roundtypes.Title("Test Round")
	testStartLocation       = roundtypes.Location("Test Location")
	testStartRoundTime      = sharedtypes.StartTime(time.Now())
	testStartEventMessageID = "12345"
)

var (
	testStartParticipant1 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user1"),
		TagNumber: nil,
		Response:  roundtypes.ResponseAccept,
	}
	testStartParticipant2 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user2"),
		TagNumber: nil,
		Response:  roundtypes.ResponseTentative,
	}
)

func TestRoundService_ProcessRoundStart(t *testing.T) {
	validRoundID := sharedtypes.RoundID(uuid.New())
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	tests := []struct {
		name           string
		setupFake      func(*FakeRepo)
		setup          func(*FakeRepo)
		payload        roundevents.RoundStartedPayloadV1
		expectedResult results.OperationResult
		expectedError  error
		{
			name: "success - round started",
			name: "successful processing",
			setup: func(r *FakeRepo) {
					ID:             validRoundID,
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       testStartLocation,
					StartTime:      &testStartRoundTime,
				}
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {

					// The service calls GetRound twice: once to check the round and once to refresh it after update.
				}
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {

					round.State = state
					if state != roundtypes.RoundStateInProgress {
						return errors.New("unexpected state")
					}
				}
			},
			verify: func(t *testing.T, res StartRoundResult, infraErr error, fake *FakeRepo) {
			payload: roundevents.RoundStartedPayloadV1{
				GuildID:   sharedtypes.GuildID("guild-123"),
				RoundID:   testStartRoundID,
				Title:     testRoundTitle,
				Location:  testStartLocation,
				StartTime: &testStartRoundTime,
			},
			expectedResult: results.OperationResult{
				Success: &roundevents.DiscordRoundStartPayloadV1{
					GuildID:   sharedtypes.GuildID("guild-123"),
					RoundID:   testStartRoundID,
					Title:     testRoundTitle,
					Location:  testStartLocation,
					StartTime: &testStartRoundTime,
					Participants: []roundevents.RoundParticipantV1{
						{
							UserID:    sharedtypes.DiscordID("user1"),
							TagNumber: nil,
							Response:  roundtypes.ResponseAccept,
							Score:     nil,
						},
						{
							UserID:    sharedtypes.DiscordID("user2"),
							TagNumber: nil,
							Response:  roundtypes.ResponseTentative,
							Score:     nil,
						},
					},
					EventMessageID: testStartEventMessageID,
				},
			},
			expectedError: nil,
		{
			name: "failure - missing event message ID",
			name: "error getting round",
			setup: func(r *FakeRepo) {
					return &roundtypes.Round{
					return &roundtypes.Round{}, errors.New("database error")
				}
			},
			payload: roundevents.RoundStartedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: testStartRoundID,
			},
			expectedResult: results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: sharedtypes.GuildID("guild-123"),
					RoundID: testStartRoundID,
					Error:   "database error",
				},
			},
			expectedError: nil,
		{
			name: "failure - update state fails",
			name: "error updating round",
			setup: func(r *FakeRepo) {
				round := &roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateUpcoming,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}
					if len(r.Trace()) > 1 {
					return round, nil
				r.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
					return errors.New("database error")
			},
			verify: func(t *testing.T, res StartRoundResult, infraErr error, fake *FakeRepo) {
			payload: roundevents.RoundStartedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: testStartRoundID,
			},
			expectedResult: results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: sharedtypes.GuildID("guild-123"),
					RoundID: testStartRoundID,
					Error:   "database error",
				},
			},
			expectedError: nil,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupFake != nil {
			if tt.setup != nil {
				tt.setup(repo)

			s := &RoundService{
				repo:    repo,
				repo:           repo,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: &FakeRoundValidator{},
				eventBus:       &FakeEventBus{},
			}

			result, err := s.ProcessRoundStart(ctx, tt.payload.GuildID, tt.payload.RoundID)

			// Check error expectation
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Check result structure
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					if expectedPayload, ok := tt.expectedResult.Success.(*roundevents.DiscordRoundStartPayloadV1); ok {
						if actualPayload, ok := result.Success.(*roundevents.DiscordRoundStartPayloadV1); ok {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected RoundID %v, got %v", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.Title != expectedPayload.Title {
								t.Errorf("expected Title %v, got %v", expectedPayload.Title, actualPayload.Title)
							}
							if actualPayload.Location != expectedPayload.Location {
								t.Errorf("expected Location %v, got %v", expectedPayload.Location, actualPayload.Location)
							}
							if (actualPayload.StartTime == nil) != (expectedPayload.StartTime == nil) {
								t.Errorf("expected StartTime nil status %v, got %v", expectedPayload.StartTime == nil, actualPayload.StartTime == nil)
							} else if actualPayload.StartTime != nil && expectedPayload.StartTime != nil && !actualPayload.StartTime.AsTime().Equal(expectedPayload.StartTime.AsTime()) {
								t.Errorf("expected StartTime %v, got %v", expectedPayload.StartTime, actualPayload.StartTime)
							}
							if actualPayload.EventMessageID != expectedPayload.EventMessageID {
								t.Errorf("expected EventMessageID %v, got %v", expectedPayload.EventMessageID, actualPayload.EventMessageID)
							}
							if !reflect.DeepEqual(actualPayload.Participants, expectedPayload.Participants) {
								t.Errorf("expected Participants %v, got %v", expectedPayload.Participants, actualPayload.Participants)
							}
						} else {
							t.Errorf("expected result.Success to be *roundevents.DiscordRoundStartPayloadV1, got %T", result.Success)
						}
					}
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					if expectedPayload, ok := tt.expectedResult.Failure.(*roundevents.RoundErrorPayloadV1); ok {
						if actualPayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1); ok {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected Failure RoundID %v, got %v", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.Error != expectedPayload.Error {
								t.Errorf("expected Failure Error %v, got %v", expectedPayload.Error, actualPayload.Error)
							}
						} else {
							t.Errorf("expected result.Failure to be *roundevents.RoundErrorPayloadV1, got %T", result.Failure)
						}
					}
				}
		})
	}
}
