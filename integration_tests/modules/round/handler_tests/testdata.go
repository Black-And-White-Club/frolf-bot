package roundhandler_integration_tests

import (
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// TestData contains unique test data for a single test.
// Each test should create its own TestData to avoid interference.
type TestData struct {
	UserID  sharedtypes.DiscordID
	RoundID sharedtypes.RoundID
	GuildID sharedtypes.GuildID
}

// NewTestData creates a new TestData with guaranteed unique IDs.
// Each call returns completely unique data suitable for parallel tests.
func NewTestData() TestData {
	return TestData{
		UserID:  sharedtypes.DiscordID(uuid.New().String()),
		GuildID: "test-guild", // Use consistent guild ID for all tests
	}
}

// WithRoundID sets the RoundID (useful after creating a round in DB).
func (td TestData) WithRoundID(id sharedtypes.RoundID) TestData {
	td.RoundID = id
	return td
}

// WithUserID sets a specific UserID (useful for edge case testing).
func (td TestData) WithUserID(id sharedtypes.DiscordID) TestData {
	td.UserID = id
	return td
}
