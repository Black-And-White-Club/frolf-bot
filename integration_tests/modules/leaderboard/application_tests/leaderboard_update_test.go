package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardService "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestUpdateLeaderboard(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error)
		roundID         sharedtypes.RoundID
		sortedTags      []string
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string)
	}{
		{
			name: "Successful update - updates existing and adds new users", // Renamed test case
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(5)
				users[0].UserID = "user_A"
				users[1].UserID = "user_B"
				users[2].UserID = "user_C"
				users[3].UserID = "user_D"
				users[4].UserID = "user_E"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: tagPtr(10)},
						{UserID: "user_C", TagNumber: tagPtr(20)},
						{UserID: "user_E", TagNumber: tagPtr(30)},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"1:user_A",
				"2:user_B",
				"3:user_D",
				"20:user_C",
				"30:user_E",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Success payload missing RoundID")
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				// Expecting 5 entries now as GenerateUpdatedLeaderboard should add new users
				expectedDBEntries := 5
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d. Actual data: %+v", expectedDBEntries, len(leaderboardEntries), activeLeaderboard.LeaderboardData)
					return
				}

				// Simulate the expected leaderboard data based on the input sorted tags
				expectedLeaderboardDataMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
				for _, tagUserID := range inputSortedTags {
					parts := strings.Split(tagUserID, ":")
					if len(parts) == 2 {
						tagNum := sharedtypes.TagNumber(0)
						_, scanErr := fmt.Sscan(parts[0], &tagNum)
						if scanErr != nil {
							t.Errorf("Failed to parse tag number from string '%s': %v", parts[0], scanErr)
							continue
						}
						userID := sharedtypes.DiscordID(parts[1])
						expectedLeaderboardDataMap[userID] = tagNum
					}
				}

				expectedLeaderboardDataSlice := make(leaderboardtypes.LeaderboardData, 0, len(expectedLeaderboardDataMap))
				for userID, tagNum := range expectedLeaderboardDataMap {
					expectedLeaderboardDataSlice = append(expectedLeaderboardDataSlice, leaderboardtypes.LeaderboardEntry{
						UserID:    userID,
						TagNumber: &tagNum,
					})
				}

				// Corrected sorting logic to return bool
				sort.SliceStable(activeLeaderboard.LeaderboardData, func(i, j int) bool {
					// Handle potential nil TagNumber pointers
					if activeLeaderboard.LeaderboardData[i].TagNumber == nil && activeLeaderboard.LeaderboardData[j].TagNumber == nil {
						return false // Order doesn't matter for stability if both are nil
					}
					if activeLeaderboard.LeaderboardData[i].TagNumber == nil {
						return true // Nil comes first
					}
					if activeLeaderboard.LeaderboardData[j].TagNumber == nil {
						return false // b is nil, a is not, so a does not come before b
					}
					return *activeLeaderboard.LeaderboardData[i].TagNumber < *activeLeaderboard.LeaderboardData[j].TagNumber
				})
				// Corrected sorting logic to return bool
				sort.SliceStable(expectedLeaderboardDataSlice, func(i, j int) bool {
					// Handle potential nil TagNumber pointers
					if expectedLeaderboardDataSlice[i].TagNumber == nil && expectedLeaderboardDataSlice[j].TagNumber == nil {
						return false // Order doesn't matter for stability if both are nil
					}
					if expectedLeaderboardDataSlice[i].TagNumber == nil {
						return true // Nil comes first
					}
					if expectedLeaderboardDataSlice[j].TagNumber == nil {
						return false // b is nil, a is not, so a does not come before b
					}
					return *expectedLeaderboardDataSlice[i].TagNumber < *expectedLeaderboardDataSlice[j].TagNumber
				})

				if len(activeLeaderboard.LeaderboardData) != len(expectedLeaderboardDataSlice) {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d. Actual data: %+v", len(expectedLeaderboardDataSlice), len(activeLeaderboard.LeaderboardData), activeLeaderboard.LeaderboardData)
					return
				}

				for i := range activeLeaderboard.LeaderboardData {
					actual := activeLeaderboard.LeaderboardData[i]
					expected := expectedLeaderboardDataSlice[i]
					if actual.UserID != expected.UserID || (actual.TagNumber == nil || expected.TagNumber == nil || *actual.TagNumber != *expected.TagNumber) {
						t.Errorf("Mismatch at index %d: Expected %+v, got %+v", i, expected, actual)
					}
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Successful update with no changes",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_X"
				users[1].UserID = "user_Y"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_X", TagNumber: tagPtr(5)},
						{UserID: "user_Y", TagNumber: tagPtr(6)},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"5:user_X",
				"6:user_Y",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				_, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				expectedLeaderboardDataMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
				for _, entry := range initialLeaderboard.LeaderboardData {
					expectedLeaderboardDataMap[entry.UserID] = *entry.TagNumber
				}

				expectedLeaderboardDataSlice := make(leaderboardtypes.LeaderboardData, 0, len(expectedLeaderboardDataMap))
				for userID, tagNum := range expectedLeaderboardDataMap {
					expectedLeaderboardDataSlice = append(expectedLeaderboardDataSlice, leaderboardtypes.LeaderboardEntry{
						UserID:    userID,
						TagNumber: &tagNum,
					})
				}

				// Corrected sorting logic to return bool
				sort.SliceStable(activeLeaderboard.LeaderboardData, func(i, j int) bool {
					// Handle potential nil TagNumber pointers
					if activeLeaderboard.LeaderboardData[i].TagNumber == nil && activeLeaderboard.LeaderboardData[j].TagNumber == nil {
						return false // Order doesn't matter for stability if both are nil
					}
					if activeLeaderboard.LeaderboardData[i].TagNumber == nil {
						return true // Nil comes first
					}
					if activeLeaderboard.LeaderboardData[j].TagNumber == nil {
						return false // b is nil, a is not, so a does not come before b
					}
					return *activeLeaderboard.LeaderboardData[i].TagNumber < *activeLeaderboard.LeaderboardData[j].TagNumber
				})
				// Corrected sorting logic to return bool
				sort.SliceStable(expectedLeaderboardDataSlice, func(i, j int) bool {
					// Handle potential nil TagNumber pointers
					if expectedLeaderboardDataSlice[i].TagNumber == nil && expectedLeaderboardDataSlice[j].TagNumber == nil {
						return false // Order doesn't matter for stability if both are nil
					}
					if expectedLeaderboardDataSlice[i].TagNumber == nil {
						return true // Nil comes first
					}
					if expectedLeaderboardDataSlice[j].TagNumber == nil {
						return false // b is nil, a is not, so a does not come before b
					}
					return *expectedLeaderboardDataSlice[i].TagNumber < *expectedLeaderboardDataSlice[j].TagNumber
				})

				if len(activeLeaderboard.LeaderboardData) != len(expectedLeaderboardDataSlice) {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d. Actual data: %+v", len(expectedLeaderboardDataSlice), len(activeLeaderboard.LeaderboardData), activeLeaderboard.LeaderboardData)
					return
				}

				for i := range activeLeaderboard.LeaderboardData {
					actual := activeLeaderboard.LeaderboardData[i]
					expected := expectedLeaderboardDataSlice[i]
					if actual.UserID != expected.UserID || (actual.TagNumber == nil || expected.TagNumber == nil || *actual.TagNumber != *expected.TagNumber) {
						t.Errorf("Mismatch at index %d: Expected %+v, got %+v", i, expected, actual)
					}
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Empty sorted participant tags",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_initial", TagNumber: tagPtr(99)},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}
				return nil, initialLeaderboard, nil
			},
			roundID:         sharedtypes.RoundID(uuid.New()),
			sortedTags:      []string{},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.LeaderboardUpdateFailedPayload, but got %T", result.Failure)
					return
				}
				if !strings.Contains(failurePayload.Reason, "invalid input: empty sorted participant tags") {
					t.Errorf("Expected failure reason to contain 'invalid input: empty sorted participant tags', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}

				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected exactly one active leaderboard, got %d", len(activeLeaderboards))
				} else if activeLeaderboards[0].ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboards[0].ID)
				}

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 1 {
					t.Errorf("Expected only one leaderboard record after empty input, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "No active leaderboard initially",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				return nil, nil, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"1:user_A",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.LeaderboardUpdateFailedPayload, but got %T", result.Failure)
					return
				}
				if !strings.Contains(failurePayload.Reason, "database connection error") && !strings.Contains(failurePayload.Reason, "no active leaderboard found") {
					t.Errorf("Expected failure reason to indicate DB error or no active leaderboard, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}

				if len(activeLeaderboards) != 0 {
					t.Errorf("Expected no active leaderboards, got %d", len(activeLeaderboards))
				}

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 0 {
					t.Errorf("Expected no leaderboard records, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "Initial active leaderboard with empty data",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
					IsActive:        true,
					UpdateSource:    leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}
				return nil, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"1:user_A",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.LeaderboardUpdateFailedPayload, but got %T", result.Failure)
					return
				}
				if !strings.Contains(failurePayload.Reason, "invalid leaderboard data") {
					t.Errorf("Expected failure reason to indicate invalid leaderboard data, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}

				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected exactly one active leaderboard, got %d", len(activeLeaderboards))
				} else if activeLeaderboards[0].ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboards[0].ID)
				}

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 1 {
					t.Errorf("Expected only one leaderboard record after empty data input, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "Successful update with one new tag (updates existing and adds one)",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_A"
				users[1].UserID = "user_B"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: tagPtr(10)},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"5:user_A",
				"6:user_B",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				_, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				// Now expecting 2 entries: user_A (updated tag) and user_B (new tag)
				expectedDBEntries := 2
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d. Actual data: %+v", expectedDBEntries, len(leaderboardEntries), activeLeaderboard.LeaderboardData)
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_A": 5,
					"user_B": 6,
				}

				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == nil || expectedTag == 0 || (entry.TagNumber != nil && *entry.TagNumber != expectedTag) {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries for users: %v", expectedDBState)
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testutils.CleanLeaderboardIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean leaderboard integration tables: %v", err)
			}
			if err := testutils.CleanUserIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean user integration tables: %v", err)
			}

			var initialUsers []testutils.User
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialUsers, initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			result, err := deps.Service.UpdateLeaderboard(sharedCtx, tt.roundID, tt.sortedTags)

			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if tt.expectedSuccess && result.Success == nil {
				t.Errorf("Expected a success result, but got nil")
			}
			if !tt.expectedSuccess && result.Success != nil {
				t.Errorf("Expected no success result, but got: %+v", result.Success)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps, result)
			}

			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialUsers, initialLeaderboard, tt.roundID, tt.sortedTags)
			}
		})
	}
}
