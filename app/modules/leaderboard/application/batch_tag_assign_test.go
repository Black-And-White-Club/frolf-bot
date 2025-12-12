package leaderboardservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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

func TestLeaderboardService_ProcessTagAssignments(t *testing.T) {
	tests := []struct {
		name                      string
		mockDBSetup               func(*leaderboarddb.MockLeaderboardDB)
		source                    interface{}
		requests                  []sharedtypes.TagAssignmentRequest
		requestingUserID          *sharedtypes.DiscordID
		operationID               uuid.UUID
		batchID                   uuid.UUID
		expectedSuccessType       string
		expectedBatchPayload      *leaderboardevents.BatchTagAssignedPayload
		expectedTagAssigned       *leaderboardevents.TagAssignedPayload
		expectedLeaderboardUpdate *leaderboardevents.LeaderboardUpdatedPayload
		expectedSwapRequested     *leaderboardevents.TagSwapRequestedPayload
		expectedFailurePayload    interface{}
		expectedError             error
	}{
		{
			name: "Successfully assigns tags in batch",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "existing_user", TagNumber: 10},
					},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
						{UserID: "existing_user", TagNumber: 10},
					},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
				mockDB.EXPECT().
					UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).
					Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
				{UserID: "user2", TagNumber: 2},
			},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				AssignmentCount:  3,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
					{UserID: "existing_user", TagNumber: 10},
				},
			},
			expectedError: nil,
		},
		{
			name: "Single user creation returns BatchTagAssigned payload",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
					},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
				mockDB.EXPECT().
					UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).
					Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceCreateUser,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedError: nil,
		},
		{
			name: "Score processing returns LeaderboardUpdated event",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
						{UserID: "user2", TagNumber: 6},
					},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
					},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
				mockDB.EXPECT().
					UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).
					Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceProcessScores,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
				{UserID: "user2", TagNumber: 2},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "leaderboard_updated",
			expectedLeaderboardUpdate: &leaderboardevents.LeaderboardUpdatedPayload{
				RoundID: sharedtypes.RoundID{},
			},
			expectedError: nil,
		},
		{
			name: "Single assignment with swap needed",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
						{UserID: "target_user", TagNumber: 1},
					},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceManual,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("user1"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "swap_requested",
			expectedSwapRequested: &leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user1",
				TargetID:    "target_user",
			},
			expectedError: nil,
		},
		{
			name: "Invalid tag assignments filtered out in batch",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user2", TagNumber: 2},
					},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
				mockDB.EXPECT().
					UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).
					Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: -1},
				{UserID: "user2", TagNumber: 2},
			},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error on UpdateLeaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), gomock.Any()).
					Return(currentLeaderboard, nil)
				mockDB.EXPECT().
					UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:      uuid.New(),
			batchID:          uuid.New(),
			expectedFailurePayload: &leaderboardevents.BatchTagAssignmentFailedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				Reason:           "database error",
			},
			expectedError: errors.New("database error"),
		},
		{
			name: "No valid assignments in batch",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(currentLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: -1},
				{UserID: "user2", TagNumber: -5},
			},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				AssignmentCount:  0,
				Assignments:      []leaderboardevents.TagAssignmentInfo{},
			},
			expectedError: nil,
		},
		{
			name: "Empty assignments list",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), gomock.Any()).
					Return(currentLeaderboard, nil)
			},
			source:              sharedtypes.ServiceUpdateSourceAdminBatch,
			requests:            []sharedtypes.TagAssignmentRequest{},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				AssignmentCount:  0,
				Assignments:      []leaderboardevents.TagAssignmentInfo{},
			},
			expectedError: nil,
		},
		{
			name: "GetActiveLeaderboard fails",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().
					GetActiveLeaderboard(gomock.Any(), guildID).
					Return(nil, errors.New("failed to get leaderboard"))
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_user_id"); return &id }(),
			operationID:      uuid.New(),
			batchID:          uuid.New(),
			expectedFailurePayload: &leaderboardevents.BatchTagAssignmentFailedPayload{
				RequestingUserID: "test_user_id",
				BatchID:          "",
				Reason:           "failed to get leaderboard",
			},
			expectedError: errors.New("failed to get leaderboard"),
		},
		{
			name: "String source type - user_creation",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: "user_creation",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedError: nil,
		},
		{
			name: "String source type - defaults to manual",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: "some_other_string",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid source type",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
			},
			source:        123,
			requests:      []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			operationID:   uuid.New(),
			batchID:       uuid.New(),
			expectedError: errors.New("invalid source type: int"),
		},
		{
			name: "Batch operation with swap needed - should skip",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
						{UserID: "target_user", TagNumber: 1},
					},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "target_user", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
						{UserID: "user1", TagNumber: 5},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), gomock.Any()).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
				{UserID: "user2", TagNumber: 2},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  3,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "target_user", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
					{UserID: "user1", TagNumber: 5},
				},
			},
			expectedError: nil,
		},
		{
			name: "Single assignment with prepare failure for existing user",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceManual,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 0}, // Invalid tag number
			},
			requestingUserID: nil,
			operationID:      uuid.New(),
			batchID:          uuid.New(),
			expectedFailurePayload: &leaderboardevents.BatchTagAssignmentFailedPayload{
				RequestingUserID: "system",
				BatchID:          "", // Will be set in test setup
				Reason:           "invalid tag number: 0",
			},
			expectedError: nil, // Business logic error, not infrastructure error
		},
		{
			name: "Single assignment - user already has tag (no-op)",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
					},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceManual,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 5},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 5},
				},
			},
			expectedError: nil,
		},
		{
			name: "Multiple user creation uses batch response",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
						{UserID: "user2", TagNumber: 2},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceCreateUser,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
				{UserID: "user2", TagNumber: 2},
			},
			requestingUserID:    func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("admin"); return &id }(),
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "admin",
				BatchID:          "",
				AssignmentCount:  2,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedError: nil,
		},
		{
			name: "Unknown source type uses fallback batch response",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSource("unknown_source"),
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedError: nil,
		},
		{
			name: "Validation uses PrepareTagUpdateForExistingUser path",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 10},
					},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 5},
					},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			source: sharedtypes.ServiceUpdateSourceAdminBatch,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 5},
			},
			requestingUserID:    nil,
			operationID:         uuid.New(),
			batchID:             uuid.New(),
			expectedSuccessType: "batch",
			expectedBatchPayload: &leaderboardevents.BatchTagAssignedPayload{
				RequestingUserID: "system",
				BatchID:          "",
				AssignmentCount:  1,
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 5},
				},
			},
			expectedError: nil,
		},
		{
			name: "Score processing with UpdateLeaderboard failure",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), guildID, gomock.Any(), gomock.Any()).Return(nil, errors.New("database error"))
			},
			source: sharedtypes.ServiceUpdateSourceProcessScores,
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user1", TagNumber: 1},
			},
			requestingUserID: nil,
			operationID:      uuid.New(),
			batchID:          uuid.New(),
			expectedFailurePayload: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: sharedtypes.RoundID{},
				Reason:  "database error",
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &leaderboardmetrics.NoOpMetrics{}

			guildID := sharedtypes.GuildID("test-guild")

			// Patch all mockDB expectations to expect guildID as the first argument
			if tt.mockDBSetup != nil {
				// Patch all mockDB expectations to expect guildID as the first argument
				// This is a manual process for each test case below
				// Ensure all mockDBSetup functions use guildID as the second argument for GetActiveLeaderboard and UpdateLeaderboard
				tt.mockDBSetup(mockDB)
			}

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			// Set up expected values with actual IDs BEFORE calling ProcessTagAssignments
			if tt.expectedBatchPayload != nil {
				tt.expectedBatchPayload.BatchID = tt.batchID.String()
			}
			if batchFailure, ok := tt.expectedFailurePayload.(*leaderboardevents.BatchTagAssignmentFailedPayload); ok {
				batchFailure.BatchID = tt.batchID.String()
			}
			if tt.expectedTagAssigned != nil {
				tt.expectedTagAssigned.AssignmentID = sharedtypes.RoundID(tt.operationID)
			}
			if tt.expectedLeaderboardUpdate != nil {
				tt.expectedLeaderboardUpdate.RoundID = sharedtypes.RoundID(tt.operationID)
			}
			if leaderboardFailure, ok := tt.expectedFailurePayload.(*leaderboardevents.LeaderboardUpdateFailedPayload); ok {
				leaderboardFailure.RoundID = sharedtypes.RoundID(tt.operationID)
			}

			// Only call ProcessTagAssignments ONCE
			got, err := s.ProcessTagAssignments(ctx, guildID, tt.source, tt.requests, tt.requestingUserID, tt.operationID, tt.batchID)

			// Validate error expectations
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError.Error()) {
					t.Errorf("expected error to contain: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate success responses
			if tt.expectedSuccessType != "" {
				if got.Success == nil {
					t.Errorf("expected success result, got: nil")
					return
				}

				switch tt.expectedSuccessType {
				case "batch":
					batchPayload, ok := got.Success.(*leaderboardevents.BatchTagAssignedPayload)
					if !ok {
						t.Errorf("expected result to be *leaderboardevents.BatchTagAssignedPayload, got: %T", got.Success)
					} else {
						validateBatchPayload(t, tt.expectedBatchPayload, batchPayload)
					}
				case "tag_assigned":
					tagPayload, ok := got.Success.(*leaderboardevents.TagAssignedPayload)
					if !ok {
						t.Errorf("expected result to be *leaderboardevents.TagAssignedPayload, got: %T", got.Success)
					} else {
						validateTagAssignedPayload(t, tt.expectedTagAssigned, tagPayload)
					}
				case "leaderboard_updated":
					leaderboardPayload, ok := got.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
					if !ok {
						t.Errorf("expected result to be *leaderboardevents.LeaderboardUpdatedPayload, got: %T", got.Success)
					} else {
						validateLeaderboardUpdatedPayload(t, tt.expectedLeaderboardUpdate, leaderboardPayload)
					}
				case "swap_requested":
					swapPayload, ok := got.Success.(*leaderboardevents.TagSwapRequestedPayload)
					if !ok {
						t.Errorf("expected result to be *leaderboardevents.TagSwapRequestedPayload, got: %T", got.Success)
					} else {
						validateTagSwapRequestedPayload(t, tt.expectedSwapRequested, swapPayload)
					}
				}
			}

			// Validate failure responses
			if tt.expectedFailurePayload != nil {
				if got.Failure == nil {
					t.Errorf("expected failure result, got: nil")
				} else {
					switch expectedFailure := tt.expectedFailurePayload.(type) {
					case *leaderboardevents.BatchTagAssignmentFailedPayload:
						failurePayload, ok := got.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
						if !ok {
							t.Errorf("expected result to be *leaderboardevents.BatchTagAssignmentFailedPayload, got: %T", got.Failure)
						} else {
							validateFailurePayload(t, expectedFailure, failurePayload)
						}
					case *leaderboardevents.LeaderboardUpdateFailedPayload:
						failurePayload, ok := got.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
						if !ok {
							t.Errorf("expected result to be *leaderboardevents.LeaderboardUpdateFailedPayload, got: %T", got.Failure)
						} else {
							validateLeaderboardUpdateFailedPayload(t, expectedFailure, failurePayload)
						}
					}
				}
			}
		})
	}
}

func validateBatchPayload(t *testing.T, expected, actual *leaderboardevents.BatchTagAssignedPayload) {
	if actual.RequestingUserID != expected.RequestingUserID {
		t.Errorf("RequestingUserID mismatch: expected %q, got %q", expected.RequestingUserID, actual.RequestingUserID)
	}
	if actual.BatchID != expected.BatchID {
		t.Errorf("BatchID mismatch: expected %q, got %q", expected.BatchID, actual.BatchID)
	}
	if actual.AssignmentCount != expected.AssignmentCount {
		t.Errorf("AssignmentCount mismatch: expected %d, got %d", expected.AssignmentCount, actual.AssignmentCount)
	}
	if len(actual.Assignments) != len(expected.Assignments) {
		t.Errorf("Assignments count mismatch: expected %d, got %d", len(expected.Assignments), len(actual.Assignments))
		return
	}
	for i, expectedAssignment := range expected.Assignments {
		actualAssignment := actual.Assignments[i]
		if actualAssignment.UserID != expectedAssignment.UserID {
			t.Errorf("Assignment[%d] UserID mismatch: expected %q, got %q", i, expectedAssignment.UserID, actualAssignment.UserID)
		}
		if actualAssignment.TagNumber != expectedAssignment.TagNumber {
			t.Errorf("Assignment[%d] TagNumber mismatch: expected %d, got %d", i, expectedAssignment.TagNumber, actualAssignment.TagNumber)
		}
	}
}

func validateTagAssignedPayload(t *testing.T, expected, actual *leaderboardevents.TagAssignedPayload) {
	if actual.UserID != expected.UserID {
		t.Errorf("UserID mismatch: expected %q, got %q", expected.UserID, actual.UserID)
	}
	if actual.TagNumber == nil || expected.TagNumber == nil {
		if actual.TagNumber != expected.TagNumber {
			t.Errorf("TagNumber mismatch: expected %v, got %v", expected.TagNumber, actual.TagNumber)
		}
	} else if *actual.TagNumber != *expected.TagNumber {
		t.Errorf("TagNumber mismatch: expected %d, got %d", *expected.TagNumber, *actual.TagNumber)
	}
	if actual.AssignmentID != expected.AssignmentID {
		t.Errorf("AssignmentID mismatch: expected %v, got %v", expected.AssignmentID, actual.AssignmentID)
	}
	if actual.Source != expected.Source {
		t.Errorf("Source mismatch: expected %q, got %q", expected.Source, actual.Source)
	}
}

func validateFailurePayload(t *testing.T, expected, actual *leaderboardevents.BatchTagAssignmentFailedPayload) {
	if actual.RequestingUserID != expected.RequestingUserID {
		t.Errorf("RequestingUserID mismatch: expected %q, got %q", expected.RequestingUserID, actual.RequestingUserID)
	}
	if actual.BatchID != expected.BatchID {
		t.Errorf("BatchID mismatch: expected %q, got %q", expected.BatchID, actual.BatchID)
	}
	if expected.Reason != "" && !strings.Contains(actual.Reason, expected.Reason) {
		t.Errorf("Reason mismatch: expected to contain %q, got %q", expected.Reason, actual.Reason)
	}
}

func validateLeaderboardUpdatedPayload(t *testing.T, expected, actual *leaderboardevents.LeaderboardUpdatedPayload) {
	if actual.RoundID != expected.RoundID {
		t.Errorf("RoundID mismatch: expected %v, got %v", expected.RoundID, actual.RoundID)
	}
}

func validateTagSwapRequestedPayload(t *testing.T, expected, actual *leaderboardevents.TagSwapRequestedPayload) {
	if actual.RequestorID != expected.RequestorID {
		t.Errorf("RequestorID mismatch: expected %q, got %q", expected.RequestorID, actual.RequestorID)
	}
	if actual.TargetID != expected.TargetID {
		t.Errorf("TargetID mismatch: expected %q, got %q", expected.TargetID, actual.TargetID)
	}
}

func validateLeaderboardUpdateFailedPayload(t *testing.T, expected, actual *leaderboardevents.LeaderboardUpdateFailedPayload) {
	if actual.RoundID != expected.RoundID {
		t.Errorf("RoundID mismatch: expected %v, got %v", expected.RoundID, actual.RoundID)
	}
	if expected.Reason != "" && !strings.Contains(actual.Reason, expected.Reason) {
		t.Errorf("Reason mismatch: expected to contain %q, got %q", expected.Reason, actual.Reason)
	}
}
