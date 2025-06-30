package testutils

import (
	"time"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

type (
	RoundID      sharedtypes.RoundID
	DiscordID    sharedtypes.DiscordID
	TagNumber    sharedtypes.TagNumber
	Score        sharedtypes.Score
	StartTime    sharedtypes.StartTime
	UserRoleEnum string
)

// TestDataGenerator provides methods to create test data for integration tests
type TestDataGenerator struct {
	faker *gofakeit.Faker
	seed  int64
}

// NewTestDataGenerator creates a new test data generator with optional seed
func NewTestDataGenerator(seed ...int64) *TestDataGenerator {
	var s int64
	if len(seed) > 0 {
		s = seed[0]
	} else {
		s = time.Now().UnixNano()
	}

	faker := gofakeit.New(uint64(s))

	return &TestDataGenerator{
		faker: faker,
		seed:  s,
	}
}

// User represents the user model in your application
type User struct {
	ID     int64        `json:"id"`
	UserID DiscordID    `json:"user_id"`
	Role   UserRoleEnum `json:"role"`
}

// ScoreModel represents the score model in your application
type ScoreModel struct {
	RoundID   RoundID                 `json:"round_id"`
	RoundData []sharedtypes.ScoreInfo `json:"round_data"`
	Source    string                  `json:"source"`
}

// Leaderboard represents the leaderboard model in your application
type Leaderboard struct {
	ID                  int64                               `json:"id"`
	LeaderboardData     []leaderboardtypes.LeaderboardEntry `json:"leaderboard_data"`
	IsActive            bool                                `json:"is_active"`
	UpdateSource        string                              `json:"update_source"`
	UpdateID            RoundID                             `json:"update_id"`
	RequestingDiscordID DiscordID                           `json:"requesting_discord_id"`
}

// GenerateUsers creates a specified number of test users
func (g *TestDataGenerator) GenerateUsers(count int) []User {
	users := make([]User, count)
	roles := []UserRoleEnum{"User", "Editor", "Admin"}

	for i := 0; i < count; i++ {
		users[i] = User{
			ID:     int64(i + 1),
			UserID: DiscordID(g.faker.Numerify("#########")),
			Role:   roles[g.faker.Number(0, len(roles)-1)],
		}
	}

	return users
}

// GenerateRound creates a test round with the specified number of participants
// Now returns roundtypes.Round directly
func (g *TestDataGenerator) GenerateRound(createdBy DiscordID, participantCount int, users []User) roundtypes.Round {
	eventTypes := []roundtypes.EventType{roundtypes.EventType("ONLINE"), roundtypes.EventType("IN_PERSON"), roundtypes.EventType("HYBRID")}
	randomEventType := eventTypes[g.faker.Number(0, len(eventTypes)-1)]
	eventType := randomEventType

	stateTypes := []roundtypes.RoundState{roundtypes.RoundStateUpcoming, roundtypes.RoundStateInProgress, roundtypes.RoundStateFinalized}

	participants := make([]roundtypes.Participant, 0, participantCount)

	// Use provided users or generate random ones
	usersToUse := users
	if len(users) == 0 {
		usersToUse = g.GenerateUsers(participantCount)
	}

	// Select random users if we have more users than participants
	if len(usersToUse) > participantCount {
		// Shuffle the users slice
		g.faker.ShuffleAnySlice(usersToUse)
		usersToUse = usersToUse[:participantCount]
	}

	// Create participants from users
	for i := 0; i < participantCount; i++ { // Changed 'count' to 'participantCount'
		var tagNumber *sharedtypes.TagNumber
		var score *sharedtypes.Score

		// Some participants have tag numbers and scores
		if g.faker.Bool() {
			tn := sharedtypes.TagNumber(i + 1)
			tagNumber = &tn
			s := sharedtypes.Score(g.faker.Float64Range(0, 1000))
			score = &s
		}

		participants = append(participants, roundtypes.Participant{
			UserID:    sharedtypes.DiscordID(usersToUse[i].UserID),
			TagNumber: tagNumber,
			Response:  roundtypes.Response(g.faker.RandomString([]string{string(roundtypes.ResponseAccept), string(roundtypes.ResponseTentative), string(roundtypes.ResponseDecline)})),
			Score:     score,
		})
	}

	var description *roundtypes.Description
	descStr := g.faker.Paragraph(1, 3, 3, "\n")
	if descStr != "" {
		d := roundtypes.Description(descStr)
		description = &d
	}

	var location *roundtypes.Location
	locStr := g.faker.Address().Street
	if locStr != "" {
		l := roundtypes.Location(locStr)
		location = &l
	}

	// --- FIX START: Ensure StartTime is always a non-zero value ---
	var startTimeVal sharedtypes.StartTime
	// Generate a future time to ensure it's not zero and is valid for upcoming rounds
	stTime := g.faker.DateRange(time.Now().Add(time.Hour), time.Now().AddDate(0, 1, 0))
	if stTime.IsZero() {
		// Fallback: if gofakeit.DateRange somehow returns a zero time, use a default future time
		startTimeVal = sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
	} else {
		startTimeVal = sharedtypes.StartTime(stTime)
	}
	// --- FIX END ---

	finalized := roundtypes.Finalized(g.faker.Bool())

	return roundtypes.Round{ // Now returning roundtypes.Round directly
		ID:             sharedtypes.RoundID(uuid.New()),
		Title:          roundtypes.Title(g.faker.Sentence(g.faker.Number(2, 5))),
		Description:    description,
		Location:       location,
		EventType:      &eventType,
		StartTime:      &startTimeVal,
		Finalized:      finalized,
		CreatedBy:      sharedtypes.DiscordID(createdBy),
		State:          stateTypes[g.faker.Number(0, len(stateTypes)-1)],
		Participants:   participants,
		EventMessageID: g.faker.UUID(),
	}
}

// GenerateRoundWithConstraints creates a round with specific constraints
// Now returns roundtypes.Round directly
func (g *TestDataGenerator) GenerateRoundWithConstraints(options RoundOptions) roundtypes.Round {
	// Generate a base round. GenerateRound now guarantees a non-zero StartTime.
	round := g.GenerateRound(options.CreatedBy, options.ParticipantCount, options.Users)

	// Apply constraints if provided
	if options.Title != "" {
		round.Title = options.Title
	}

	if options.State != "" {
		round.State = options.State
	}

	if options.StartTime != nil {
		// If StartTime is provided in options, use its dereferenced value.
		// This will override the one generated by GenerateRound if `options.StartTime` is non-nil.
		round.StartTime = options.StartTime
	}
	// If options.StartTime is nil, `round.StartTime` will retain the non-zero
	// value generated by `g.GenerateRound`, which is correct.

	// Note: ID, EventMessageID, and Participants are not in RoundOptions,
	// so they are expected to be set manually in the test setup after this call,
	// as currently done in the test case.
	if options.ID != sharedtypes.RoundID(uuid.Nil) { // Allow setting ID if provided in options
		round.ID = sharedtypes.RoundID(options.ID)
	}

	return round
}

// RoundOptions provides options for creating constrained rounds
type RoundOptions struct {
	ID               sharedtypes.RoundID // Added ID field here for consistency in testutils
	CreatedBy        DiscordID
	ParticipantCount int
	Users            []User
	Title            roundtypes.Title
	State            roundtypes.RoundState
	StartTime        *sharedtypes.StartTime
	Finalized        *roundtypes.Finalized
	// EventMessageID and Participants are NOT in this struct, as per your previous clarification.
	// They are set manually in the test setup after GenerateRoundWithConstraints.
}

// GenerateScore creates a test score for a round
// Now accepts roundtypes.Round
func (g *TestDataGenerator) GenerateScore(round roundtypes.Round) ScoreModel {
	scoreInfos := make([]sharedtypes.ScoreInfo, 0, len(round.Participants))

	for _, participant := range round.Participants {
		// Only include participants with scores
		if participant.Score != nil {
			scoreInfos = append(scoreInfos, sharedtypes.ScoreInfo{
				UserID:    participant.UserID,
				Score:     *participant.Score,
				TagNumber: participant.TagNumber,
			})
		} else if g.faker.Bool() {
			// Add some random scores for participants that don't have them
			score := sharedtypes.Score(g.faker.Float64Range(0, 1000))
			scoreInfos = append(scoreInfos, sharedtypes.ScoreInfo{
				UserID:    participant.UserID,
				Score:     score,
				TagNumber: participant.TagNumber,
			})
		}
	}

	return ScoreModel{
		RoundID:   RoundID(round.ID), // Cast round.ID to testutils.RoundID
		RoundData: scoreInfos,
		Source:    g.faker.RandomString([]string{"API", "DISCORD", "MANUAL"}),
	}
}

// GenerateLeaderboard creates a test leaderboard based on multiple rounds
// Now accepts []roundtypes.Round
func (g *TestDataGenerator) GenerateLeaderboard(rounds []roundtypes.Round, isActive bool) Leaderboard {
	entries := make([]leaderboardtypes.LeaderboardEntry, 0)
	userScores := make(map[sharedtypes.DiscordID]float64)
	userTags := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)

	// Aggregate scores across all rounds
	for _, round := range rounds {
		for _, participant := range round.Participants {
			if participant.Score != nil {
				userID := participant.UserID
				userScores[userID] += float64(*participant.Score)
				if participant.TagNumber != nil {
					userTags[userID] = participant.TagNumber
				}
			}
		}
	}

	// Convert to leaderboard entries
	for userID := range userScores {
		// Only include users who have a tag assigned
		if userTags[userID] != nil {
			entries = append(entries, leaderboardtypes.LeaderboardEntry{
				TagNumber: *userTags[userID],
				UserID:    userID,
			})
		}
	}

	return Leaderboard{
		ID:                  int64(g.faker.Number(1, 1000)),
		LeaderboardData:     entries,
		IsActive:            isActive,
		UpdateSource:        g.faker.RandomString([]string{"ROUND_COMPLETION", "MANUAL_UPDATE", "SCHEDULED_UPDATE"}),
		UpdateID:            RoundID(rounds[len(rounds)-1].ID), // Cast round.ID to testutils.RoundID
		RequestingDiscordID: DiscordID(g.faker.Numerify("#########")),
	}
}

// GenerateTestData creates a complete set of test data with interrelated entities
// Now returns []roundtypes.Round
func (g *TestDataGenerator) GenerateTestData(userCount, roundCount int) ([]User, []roundtypes.Round, []ScoreModel, Leaderboard) {
	users := g.GenerateUsers(userCount)
	rounds := make([]roundtypes.Round, roundCount) // Changed to roundtypes.Round
	scores := make([]ScoreModel, roundCount)

	// Create rounds and scores
	for i := 0; i < roundCount; i++ {
		// Use a random admin user as the creator
		var creatorID DiscordID
		for _, user := range users {
			if user.Role == "Admin" {
				creatorID = user.UserID
				break
			}
		}
		if creatorID == "" {
			creatorID = users[0].UserID
		}

		// Number of participants varies per round
		participantCount := g.faker.Number(2, userCount)
		rounds[i] = g.GenerateRound(creatorID, participantCount, users)
		scores[i] = g.GenerateScore(rounds[i])
	}

	// Create a leaderboard based on all rounds
	leaderboard := g.GenerateLeaderboard(rounds, true)

	return users, rounds, scores, leaderboard
}

// GenerateTagNumber creates a random tag number for testing
func (g *TestDataGenerator) GenerateTagNumber() int {
	// Generate a tag number between 1 and 100 (typical disc golf tag range)
	return g.faker.Number(1, 100)
}

// StartTimePtr is a helper to create a pointer to sharedtypes.StartTime.
// This is moved here as it's a general utility for creating test data.
func StartTimePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

// RoundDescriptionPtr is a helper to create a pointer to roundtypes.Description.
func RoundDescriptionPtr(s string) *roundtypes.Description {
	d := roundtypes.Description(s)
	return &d
}

// RoundLocationPtr is a helper to create a pointer to roundtypes.Location.
func RoundLocationPtr(s string) *roundtypes.Location {
	l := roundtypes.Location(s)
	return &l
}

// RoundFinalizedPtr is a helper to create a pointer to roundtypes.Finalized.
func RoundFinalizedPtr(b bool) *roundtypes.Finalized {
	f := roundtypes.Finalized(b)
	return &f
}

// Example usage:
func ExampleUsage() {
	// Create a generator with a fixed seed for reproducibility
	generator := NewTestDataGenerator(42)

	// Generate a basic dataset
	users, rounds, scores, leaderboard := generator.GenerateTestData(10, 3)

	// Or generate specific entities
	adminUsers := generator.GenerateUsers(2)
	roundOptions := RoundOptions{
		CreatedBy:        adminUsers[0].UserID,
		ParticipantCount: 5,
		Users:            users,
		State:            roundtypes.RoundState("PUBLISHED"),
	}

	specialRound := generator.GenerateRoundWithConstraints(roundOptions)
	specialScore := generator.GenerateScore(specialRound)

	// Use the generated data in your tests
	_ = users
	_ = rounds
	_ = scores
	_ = leaderboard
	_ = specialRound
	_ = specialScore
}
