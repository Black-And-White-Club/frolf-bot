package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)
	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		guildID        sharedtypes.GuildID
		mockDBSetup    func(*leaderboarddb.MockRepository, sharedtypes.GuildID)
		expectedResult *leaderboardevents.GetLeaderboardResponsePayloadV1
		expectedFail   *leaderboardevents.GetLeaderboardFailedPayloadV1
		expectedError  error
	}{
		{
			name:    "Successfully retrieves leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: tag1, UserID: "user1"},
						{TagNumber: tag2, UserID: "user2"},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID: sharedtypes.GuildID("test-guild"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(2)
							return val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name:    "Fails to fetch active leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(nil, errors.New("database connection error"))
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				Reason: "database error",
			},
			expectedError: nil,
		},
		{
			name:    "Successfully retrieves empty leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
					IsActive:        true,
					UpdateSource:    sharedtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID:     sharedtypes.GuildID("test-guild"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name:    "Successfully retrieves leaderboard with mixed tag numbers",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: tag1, UserID: "user1"},
						{TagNumber: tag2, UserID: "user2"},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID: sharedtypes.GuildID("test-guild"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(2)
							return val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name:    "Handles no active leaderboard",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				Reason: "database error",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockRepository(ctrl)
			tt.mockDBSetup(mockDB, tt.guildID)

			s := &LeaderboardService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			got, err := s.GetLeaderboard(ctx, tt.guildID)

			// ...existing code...
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					if !strings.Contains(err.Error(), tt.expectedError.Error()) {
						t.Errorf("expected error to contain: %q, got: %q", tt.expectedError.Error(), err.Error())
					}
				}
			} else {
				if err != nil {
					// If a failure payload is expected, the service may return a non-nil error
					// (operation wrapper includes error context). Accept that and continue
					// to validate the OperationResult payload.
					if tt.expectedFail == nil {
						t.Errorf("expected no error, got: %v", err)
					}
				}
			}

			// The service now returns a results.OperationResult where Success/Failure
			// payloads contain the leaderboard or failure details.
			if tt.expectedResult != nil {
				if !got.IsSuccess() {
					t.Fatalf("expected success result, got failure: %v", got.Failure)
				}
				payload, ok := got.Success.(*leaderboardevents.GetLeaderboardResponsePayloadV1)
				if !ok {
					t.Fatalf("unexpected success payload type: %T", got.Success)
				}
				if len(payload.Leaderboard) != len(tt.expectedResult.Leaderboard) {
					t.Errorf("Leaderboard length mismatch: got %d, want %d", len(payload.Leaderboard), len(tt.expectedResult.Leaderboard))
					return
				}
				for i := range payload.Leaderboard {
					gotEntry := payload.Leaderboard[i]
					wantEntry := tt.expectedResult.Leaderboard[i]
					if gotEntry.UserID != wantEntry.UserID {
						t.Errorf("Leaderboard entry[%d] UserID mismatch: got %q, want %q", i, gotEntry.UserID, wantEntry.UserID)
					}
					if gotEntry.TagNumber != wantEntry.TagNumber {
						t.Errorf("Leaderboard entry[%d] TagNumber mismatch: got %v, want %v", i, gotEntry.TagNumber, wantEntry.TagNumber)
					}
				}
			} else {
				if tt.expectedFail != nil {
					if !got.IsFailure() {
						t.Fatalf("expected failure result, got success: %v", got.Success)
					}
					payload, ok := got.Failure.(*leaderboardevents.GetLeaderboardFailedPayloadV1)
					if !ok {
						t.Fatalf("unexpected failure payload type: %T", got.Failure)
					}
					if !strings.Contains(payload.Reason, tt.expectedFail.Reason) {
						t.Errorf("LeaderboardService.GetLeaderboard() failure reason mismatch: expected reason to contain %q, got %q",
							tt.expectedFail.Reason, payload.Reason)
					}
				} else {
					// Neither success nor explicit failure expected: ensure no payloads
					if got.IsSuccess() {
						p, _ := got.Success.(*leaderboardevents.GetLeaderboardResponsePayloadV1)
						if p != nil && len(p.Leaderboard) != 0 {
							t.Errorf("Expected nil/empty leaderboard payload, got %v", p.Leaderboard)
						}
					}
					if got.IsFailure() {
						t.Errorf("Expected no failure result, got: %v", got.Failure)
					}
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

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name        string
		guildID     sharedtypes.GuildID
		payload     sharedevents.DiscordTagLookupRequestedPayloadV1
		mockDBSetup func(*leaderboarddb.MockRepository, sharedtypes.GuildID)
		expectedTag sharedtypes.TagNumber
		expectedErr error
	}{
		{
			name:    "Successfully retrieves tag number",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: sharedevents.DiscordTagLookupRequestedPayloadV1{UserID: testUserID},
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				tagNumber := sharedtypes.TagNumber(5)
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(&tagNumber, nil)
			},
			expectedTag: sharedtypes.TagNumber(5),
			expectedErr: nil,
		},
		{
			name:    "Fails to retrieve tag number due to unexpected DB error",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: sharedevents.DiscordTagLookupRequestedPayloadV1{UserID: testUserID},
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, errors.New("unexpected DB error"))
			},
			expectedTag: 0,
			expectedErr: fmt.Errorf("system error retrieving tag: %w", errors.New("unexpected DB error")),
		},
		{
			name:    "User ID not found in database (sql.ErrNoRows)",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: sharedevents.DiscordTagLookupRequestedPayloadV1{UserID: testUserID},
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, sql.ErrNoRows)
			},
			expectedTag: 0,
			expectedErr: sql.ErrNoRows,
		},
		{
			name:    "No active leaderboard found",
			guildID: sharedtypes.GuildID("test-guild"),
			payload: sharedevents.DiscordTagLookupRequestedPayloadV1{UserID: testUserID},
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			expectedTag: 0,
			expectedErr: leaderboarddbtypes.ErrNoActiveLeaderboard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockRepository(ctrl)
			tt.mockDBSetup(mockDB, tt.guildID)

			s := &LeaderboardService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			tag, err := s.GetTagByUserID(ctx, tt.guildID, tt.payload.UserID)

			if tt.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tag != tt.expectedTag {
					t.Fatalf("expected tag %v, got %v", tt.expectedTag, tag)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.expectedErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr.Error()) {
					t.Fatalf("expected error to contain %q, got %q", tt.expectedErr.Error(), err.Error())
				}
			}
		})
	}
}

func TestLeaderboardService_RoundGetTagByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("user1")
	testRoundID := sharedtypes.RoundID(uuid.New())

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	dummyRequestPayload := sharedevents.RoundTagLookupRequestedPayloadV1{
		UserID:     testUserID,
		RoundID:    testRoundID,
		Response:   "ACCEPT",
		JoinedLate: func() *bool { b := false; return &b }(),
	}

	tests := []struct {
		name           string
		guildID        sharedtypes.GuildID
		mockDBSetup    func(*leaderboarddb.MockRepository, sharedtypes.GuildID)
		requestPayload sharedevents.RoundTagLookupRequestedPayloadV1
		expectPresent  bool
		expectedTag    sharedtypes.TagNumber
	}{
		{
			name:    "Successfully retrieves tag number",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				tagNumber := sharedtypes.TagNumber(5)
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(&tagNumber, nil)
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  true,
			expectedTag:    sharedtypes.TagNumber(5),
		},
		{
			name:    "Fails to retrieve tag number (operational error)",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, errors.New("database connection error"))
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  false,
		},
		{
			name:    "User ID not found in database (sql.ErrNoRows)",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, sql.ErrNoRows)
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  false,
		},
		{
			name:    "User ID not found in database (string match)",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, errors.New("user not found in DB"))
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  false,
		},
		{
			name:    "Nil tag number returned from database",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, nil)
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  false,
		},
		{
			name:    "Handles no active leaderboard (Round)",
			guildID: sharedtypes.GuildID("test-guild"),
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository, guildID sharedtypes.GuildID) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), guildID, testUserID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			requestPayload: dummyRequestPayload,
			expectPresent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockRepository(ctrl)
			tt.mockDBSetup(mockDB, tt.guildID)

			s := &LeaderboardService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			got, err := s.RoundGetTagByUserID(ctx, tt.guildID, tt.requestPayload)

			if err != nil {
				t.Fatalf("unexpected error from RoundGetTagByUserID: %v", err)
			}

			if tt.expectPresent {
				if !got.IsSuccess() {
					t.Fatalf("expected success result, got failure: %v", got.Failure)
				}
				payload, ok := got.Success.(*sharedevents.RoundTagLookupResultPayloadV1)
				if !ok {
					t.Fatalf("unexpected success payload type: %T", got.Success)
				}
				if !payload.Found {
					t.Fatalf("expected Found=true in payload, got Found=false")
				}
				if payload.TagNumber == nil {
					t.Fatalf("expected TagNumber present, got nil")
				}
				if *payload.TagNumber != tt.expectedTag {
					t.Fatalf("expected TagNumber %v, got %v", tt.expectedTag, *payload.TagNumber)
				}
			} else {
				if !got.IsFailure() {
					t.Fatalf("expected failure result for missing tag, got success: %v", got.Success)
				}
				payload, ok := got.Failure.(*sharedevents.RoundTagLookupResultPayloadV1)
				if !ok {
					t.Fatalf("unexpected failure payload type: %T", got.Failure)
				}
				if payload.Found {
					t.Fatalf("expected Found=false in failure payload, got Found=true")
				}
			}
		})
	}
}

func TestLeaderboardService_CheckTagAvailability(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	guildID := sharedtypes.GuildID("test-guild")
	userID := sharedtypes.DiscordID("user-123")
	tagNumber := sharedtypes.TagNumber(42)

	tests := []struct {
		name           string
		payloadTag     *sharedtypes.TagNumber
		mockDBSetup    func(*leaderboarddb.MockRepository)
		expectResult   sharedevents.TagAvailabilityCheckResultPayloadV1
		expectFailure  *sharedevents.TagAvailabilityCheckFailedPayloadV1
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name:       "missing tag number",
			payloadTag: nil,
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				// No DB call expected
			},
			expectResult:  sharedevents.TagAvailabilityCheckResultPayloadV1{},
			expectFailure: &sharedevents.TagAvailabilityCheckFailedPayloadV1{GuildID: guildID, UserID: userID, TagNumber: nil, Reason: "tag number is required"},
			expectErr:     false,
		},
		{
			name:       "db error",
			payloadTag: &tagNumber,
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, userID, tagNumber).Return(leaderboarddbtypes.TagAvailabilityResult{}, fmt.Errorf("db error"))
			},
			expectResult:   sharedevents.TagAvailabilityCheckResultPayloadV1{},
			expectFailure:  nil,
			expectErr:      true,
			expectedErrMsg: "db error",
		},
		{
			name:       "tag available",
			payloadTag: &tagNumber,
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, userID, tagNumber).Return(leaderboarddbtypes.TagAvailabilityResult{Available: true}, nil)
			},
			expectResult: sharedevents.TagAvailabilityCheckResultPayloadV1{
				GuildID:   guildID,
				UserID:    userID,
				TagNumber: &tagNumber,
				Available: true,
			},
			expectFailure: nil,
			expectErr:     false,
		},
		{
			name:       "tag unavailable",
			payloadTag: &tagNumber,
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				mockDB.EXPECT().CheckTagAvailability(gomock.Any(), guildID, userID, tagNumber).Return(leaderboarddbtypes.TagAvailabilityResult{Available: false, Reason: "tag already taken"}, nil)
			},
			expectResult: sharedevents.TagAvailabilityCheckResultPayloadV1{
				GuildID:   guildID,
				UserID:    userID,
				TagNumber: &tagNumber,
				Available: false,
				Reason:    "tag already taken",
			},
			expectFailure: nil,
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockRepository(ctrl)
			if tt.mockDBSetup != nil {
				tt.mockDBSetup(mockDB)
			}

			s := &LeaderboardService{
				repo:    mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			result, failure, err := s.CheckTagAvailability(ctx, guildID, userID, tt.payloadTag)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectedErrMsg != "" && !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Fatalf("expected error message to contain %q, got %q", tt.expectedErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.expectFailure != nil {
				if failure == nil {
					t.Fatalf("expected failure payload, got nil")
				}
				if failure.Reason != tt.expectFailure.Reason {
					t.Errorf("expected failure reason %q, got %q", tt.expectFailure.Reason, failure.Reason)
				}
				if failure.TagNumber != nil || tt.expectFailure.TagNumber != nil {
					if failure.TagNumber == nil || tt.expectFailure.TagNumber == nil || *failure.TagNumber != *tt.expectFailure.TagNumber {
						t.Errorf("expected failure tag number %v, got %v", tt.expectFailure.TagNumber, failure.TagNumber)
					}
				}
				return
			}

			if failure != nil {
				t.Fatalf("expected no failure payload, got %+v", failure)
			}

			if result.GuildID != tt.expectResult.GuildID {
				t.Errorf("expected guild %s, got %s", tt.expectResult.GuildID, result.GuildID)
			}
			if result.UserID != tt.expectResult.UserID {
				t.Errorf("expected user %s, got %s", tt.expectResult.UserID, result.UserID)
			}
			if tt.expectResult.TagNumber == nil {
				if result.TagNumber != nil {
					t.Errorf("expected nil tag number, got %v", result.TagNumber)
				}
			} else if result.TagNumber == nil || *result.TagNumber != *tt.expectResult.TagNumber {
				t.Errorf("expected tag number %v, got %v", tt.expectResult.TagNumber, result.TagNumber)
			}
			if result.Available != tt.expectResult.Available {
				t.Errorf("expected available %v, got %v", tt.expectResult.Available, result.Available)
			}
			if result.Reason != tt.expectResult.Reason {
				t.Errorf("expected reason %q, got %q", tt.expectResult.Reason, result.Reason)
			}
		})
	}
}
