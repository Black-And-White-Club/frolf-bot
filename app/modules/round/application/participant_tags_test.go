package roundservice

import (
	"context"
	"errors"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundService_UpdateScheduledRoundsWithNewTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	// Test data
	user1ID := sharedtypes.DiscordID("user1")
	user2ID := sharedtypes.DiscordID("user2")
	user3ID := sharedtypes.DiscordID("user3")

	round1ID := sharedtypes.RoundID(uuid.New())
	round2ID := sharedtypes.RoundID(uuid.New())

	// Define tag numbers
	tag1 := sharedtypes.TagNumber(42)
	tag2 := sharedtypes.TagNumber(17)
	tag3 := sharedtypes.TagNumber(99)

	newTag1 := sharedtypes.TagNumber(23)
	newTag2 := sharedtypes.TagNumber(31)

	// Create participants with existing tags
	participant1 := roundtypes.Participant{
		UserID:    user1ID,
		TagNumber: &tag1,
		Response:  roundtypes.ResponseAccept,
	}
	participant2 := roundtypes.Participant{
		UserID:    user2ID,
		TagNumber: &tag2,
		Response:  roundtypes.ResponseAccept,
	}
	participant3 := roundtypes.Participant{
		UserID:    user3ID,
		TagNumber: &tag3,
		Response:  roundtypes.ResponseAccept,
	}

	// Setup test rounds with participants
	round1 := roundtypes.Round{
		ID:             round1ID,
		EventMessageID: "msg1", // Add EventMessageID
		Participants:   []roundtypes.Participant{participant1, participant2},
	}

	round2 := roundtypes.Round{
		ID:             round2ID,
		EventMessageID: "msg2", // Add EventMessageID
		Participants:   []roundtypes.Participant{participant2, participant3},
	}

	// Create slice of pointers to rounds
	upcomingRounds := []*roundtypes.Round{&round1, &round2}

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ScheduledRoundTagUpdatePayload
		expectedResult func(result RoundOperationResult) bool
		expectError    bool
	}{
		{
			name: "successful update with valid tags",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(upcomingRounds, nil)
				mockDB.EXPECT().UpdateRoundsAndParticipants(ctx, gomock.Any()).Return(nil)
			},
			payload: roundevents.ScheduledRoundTagUpdatePayload{
				ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					user1ID: &newTag1,
					user2ID: &newTag2,
				},
			},
			expectedResult: func(result RoundOperationResult) bool {
				if result.Success == nil {
					return false
				}
				payload, ok := result.Success.(*roundevents.DiscordRoundUpdatePayload)
				if !ok {
					return false
				}
				// Should have 2 rounds (both rounds have participants that need updates)
				return len(payload.RoundIDs) == 2 && len(payload.Participants) == 3 // user2 appears in both rounds
			},
			expectError: false,
		},
		{
			name: "error fetching rounds",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(nil, errors.New("database error"))
			},
			payload: roundevents.ScheduledRoundTagUpdatePayload{
				ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					user1ID: &newTag1,
				},
			},
			expectedResult: func(result RoundOperationResult) bool {
				if result.Failure == nil {
					return false
				}
				errorPayload, ok := result.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					return false
				}
				return errorPayload.Error == "failed to get upcoming rounds: database error"
			},
			expectError: false, // Error is in the result, not returned
		},
		{
			name: "error updating rounds",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(upcomingRounds, nil)
				mockDB.EXPECT().UpdateRoundsAndParticipants(ctx, gomock.Any()).Return(errors.New("update failed"))
			},
			payload: roundevents.ScheduledRoundTagUpdatePayload{
				ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					user1ID: &newTag1,
				},
			},
			expectedResult: func(result RoundOperationResult) bool {
				if result.Failure == nil {
					return false
				}
				errorPayload, ok := result.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					return false
				}
				return errorPayload.Error == "update failed"
			},
			expectError: false, // Error is in the result, not returned
		},
		{
			name: "no updates needed - empty changedTags",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(upcomingRounds, nil)
				// No UpdateRoundsAndParticipants call expected when no updates
			},
			payload: roundevents.ScheduledRoundTagUpdatePayload{
				ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{}, // Empty map
			},
			expectedResult: func(result RoundOperationResult) bool {
				if result.Success == nil {
					return false
				}
				payload, ok := result.Success.(*roundevents.DiscordRoundUpdatePayload)
				if !ok {
					return false
				}
				// Should have empty arrays when no updates
				return len(payload.RoundIDs) == 0 && len(payload.Participants) == 0 && len(payload.EventMessageIDs) == 0
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh mock for each test to avoid conflicts
			mockDB := rounddb.NewMockRoundDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.UpdateScheduledRoundsWithNewTags(ctx, tt.payload)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("expected an error, but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("expected no error, but got: %v", err)
			}

			// Check result expectation
			if !tt.expectedResult(result) {
				t.Errorf("result validation failed. Got result: Success=%v, Failure=%v", result.Success, result.Failure)
			}
		})
	}
}
