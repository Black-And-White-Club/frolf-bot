package roundservice

import (
	"context"
	"errors"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundService_UpdateScheduledRoundsWithNewTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
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
		ID:           round1ID,
		Participants: []roundtypes.Participant{participant1, participant2},
	}

	round2 := roundtypes.Round{
		ID:           round2ID,
		Participants: []roundtypes.Participant{participant2, participant3},
	}

	// Create slice of pointers to rounds
	upcomingRounds := []*roundtypes.Round{&round1, &round2}

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ScheduledRoundTagUpdatePayload
		expectedResult RoundOperationResult
		expectedError  error
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
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundUpdateSuccess,
			},
			expectedError: nil,
		},
		{
			name: "error fetching rounds",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(nil, errors.New("database error"))
			},
			payload:        roundevents.ScheduledRoundTagUpdatePayload{},
			expectedResult: RoundOperationResult{},
			expectedError:  errors.New("failed to get rounds and participants: failed to get upcoming rounds: database error"),
		},
		{
			name: "error updating rounds",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetUpcomingRounds(ctx).Return(upcomingRounds, nil)
				mockDB.EXPECT().UpdateRoundsAndParticipants(ctx, gomock.Any()).Return(errors.New("update failed"))
			},
			payload:        roundevents.ScheduledRoundTagUpdatePayload{},
			expectedResult: RoundOperationResult{},
			expectedError:  errors.New("failed to update rounds: update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			_, err := s.UpdateScheduledRoundsWithNewTags(ctx, tt.payload)
			if (err != nil) && (tt.expectedError == nil || err.Error() != tt.expectedError.Error()) {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

		})
	}
}
