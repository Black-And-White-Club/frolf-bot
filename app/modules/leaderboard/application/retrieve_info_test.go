package leaderboardservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testRoundID := sharedtypes.RoundID(uuid.New())

	// No-Op implementations for logging, metrics, and tracing
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		expectedResult *leaderboardevents.GetLeaderboardResponsePayload
		expectedFail   *leaderboardevents.GetLeaderboardFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: 1, UserID: "user1"},
						{TagNumber: 2, UserID: "user2"},
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardevents.LeaderboardEntry{
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return &val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(2)
							return &val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails to fetch active leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database connection error"))
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetLeaderboardFailedPayload{
				Reason: "failed to get active leaderboard",
			},
			expectedError: errors.New("database connection error"),
		},
		// New test case: Empty leaderboard
		{
			name: "Successfully retrieves empty leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardevents.LeaderboardEntry{},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		// New test case: Leaderboard with mixed tag numbers
		{
			name: "Successfully retrieves leaderboard with mixed tag numbers",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: 1, UserID: "user1"},
						{TagNumber: 0, UserID: "user2"}, // Mixed tag number
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardevents.LeaderboardEntry{
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return &val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(0) // Mixed tag number
							return &val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc()
				},
			}

			got, err := s.GetLeaderboard(ctx, testMsg)

			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.expectedError)
				return
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload, got nil")
					return
				}

				successPayload, ok := got.Success.(*leaderboardevents.GetLeaderboardResponsePayload)
				if !ok {
					t.Errorf("Expected success payload type, got %T", got.Success)
					return
				}

				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.GetLeaderboard() result mismatch: got %v, want %v", successPayload, tt.expectedResult)
				}
			} else {
				if got.Success != nil {
					t.Errorf("Expected nil success payload, got %v", got.Success)
				}
			}

			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("Expected failure payload, got nil")
					return
				}

				failurePayload, ok := got.Failure.(*leaderboardevents.GetLeaderboardFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload type, got %T", got.Failure)
					return
				}

				if failurePayload.Reason != tt.expectedFail.Reason {
					t.Errorf("LeaderboardService.GetLeaderboard() failure reason mismatch: got %v, want %v", failurePayload.Reason, tt.expectedFail.Reason)
				}
			} else {
				if got.Failure != nil {
					t.Errorf("Expected nil failure payload, got %v", got.Failure)
				}
			}
		})
	}
}

func TestLeaderboardService_GetTagByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("user1")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testMsg := message.NewMessage("test-id", nil)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		expectedResult *leaderboardevents.GetTagNumberResponsePayload
		expectedFail   *leaderboardevents.GetTagNumberFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves tag number",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				tagNumber := int(5)
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(&tagNumber, nil)
			},
			expectedResult: &leaderboardevents.GetTagNumberResponsePayload{
				TagNumber: func() *sharedtypes.TagNumber {
					val := sharedtypes.TagNumber(5)
					return &val
				}(),
				UserID:  testUserID,
				RoundID: testRoundID,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails to retrieve tag number",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("user not found"))
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetTagNumberFailedPayload{
				Reason: "failed to get tag by UserID",
			},
			expectedError: errors.New("user not found"),
		},
		// New test case: UserID not found
		{
			name: "User  ID not found in database",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("user not found"))
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetTagNumberFailedPayload{
				Reason: "failed to get tag by UserID",
			},
			expectedError: errors.New("user not found"),
		},
		// New test case: Nil tag number returned
		{
			name: "Nil tag number returned",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, nil)
			},
			expectedResult: &leaderboardevents.GetTagNumberResponsePayload{
				TagNumber: nil,
				UserID:    testUserID,
				RoundID:   testRoundID,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc()
				},
			}

			got, err := s.GetTagByUserID(ctx, testMsg, testUserID, testRoundID)

			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetTagByUserID() error = %v, wantErr %v", err, tt.expectedError)
				return
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetTagByUserID() error = %v, wantErr %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload, got nil")
					return
				}
				successPayload, ok := got.Success.(*leaderboardevents.GetTagNumberResponsePayload)
				if !ok {
					t.Errorf("Expected success payload type, got %T", got.Success)
					return
				}

				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.GetTagByUserID() result mismatch: got %v, want %v", successPayload, tt.expectedResult)
				}
			} else {
				if got.Success != nil {
					t.Errorf("Expected nil success payload, got %v", got.Success)
				}
			}

			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("Expected failure payload, got nil")
					return
				}

				failurePayload, ok := got.Failure.(*leaderboardevents.GetTagNumberFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload type, got %T", got.Failure)
					return
				}

				if failurePayload.Reason != tt.expectedFail.Reason {
					t.Errorf("LeaderboardService.GetTagByUserID() failure reason mismatch: got %v, want %v", failurePayload.Reason, tt.expectedFail.Reason)
				}
			} else {
				if got.Failure != nil {
					t.Errorf("Expected nil failure payload, got %v", got.Failure)
				}
			}
		})
	}
}
