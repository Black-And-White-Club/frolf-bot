package testutils

import (
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

// Replace these with your actual import paths
type (
	RoundID      string
	DiscordID    string
	TagNumber    int
	Score        float64
	StartTime    time.Time
	Title        string
	Description  string
	Location     string
	EventType    string
	Finalized    bool
	RoundState   string
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

// Response represents a participant response in a round
type Response struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

// Participant represents a participant in a round
type Participant struct {
	UserID    DiscordID  `json:"user_id"`
	TagNumber *TagNumber `json:"tag_number,omitempty"`
	Response  Response   `json:"response"`
	Score     *Score     `json:"score"`
}

// Round represents a round in your application
type Round struct {
	ID             RoundID       `json:"id"`
	Title          Title         `json:"title"`
	Description    Description   `json:"description"`
	Location       Location      `json:"location"`
	EventType      *EventType    `json:"event_type"`
	StartTime      StartTime     `json:"start_time"`
	Finalized      Finalized     `json:"finalized"`
	CreatedBy      DiscordID     `json:"created_by"`
	State          RoundState    `json:"state"`
	Participants   []Participant `json:"participants"`
	EventMessageID string        `json:"event_message_id"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// ScoreInfo represents score information for a user
type ScoreInfo struct {
	UserID    DiscordID  `json:"user_id"`
	Score     Score      `json:"score"`
	TagNumber *TagNumber `json:"tag_number"`
}

// ScoreModel represents the score model in your application
type ScoreModel struct {
	RoundID   RoundID     `json:"round_id"`
	RoundData []ScoreInfo `json:"round_data"`
	Source    string      `json:"source"`
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	TagNumber *TagNumber `json:"tag_number"`
	UserID    DiscordID  `json:"user_id"`
}

// Leaderboard represents the leaderboard model in your application
type Leaderboard struct {
	ID                  int64              `json:"id"`
	LeaderboardData     []LeaderboardEntry `json:"leaderboard_data"`
	IsActive            bool               `json:"is_active"`
	UpdateSource        string             `json:"update_source"`
	UpdateID            RoundID            `json:"update_id"`
	RequestingDiscordID DiscordID          `json:"requesting_discord_id"`
}

// GenerateUsers creates a specified number of test users
func (g *TestDataGenerator) GenerateUsers(count int) []User {
	users := make([]User, count)
	roles := []UserRoleEnum{"Rattler", "Admin", "Moderator"}

	for i := 0; i < count; i++ {
		users[i] = User{
			ID:     int64(i + 1),
			UserID: DiscordID(fmt.Sprintf("%d", g.faker.Number(100000000, 999999999))),
			Role:   roles[g.faker.Number(0, len(roles)-1)],
		}
	}

	return users
}

// GenerateRound creates a test round with the specified number of participants
func (g *TestDataGenerator) GenerateRound(createdBy DiscordID, participantCount int, users []User) Round {
	eventTypes := []string{"ONLINE", "IN_PERSON", "HYBRID"}
	randomEventType := eventTypes[g.faker.Number(0, len(eventTypes)-1)]
	eventType := EventType(randomEventType)

	stateTypes := []RoundState{"DRAFT", "PUBLISHED", "COMPLETED"}

	participants := make([]Participant, 0, participantCount)

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
	for i, user := range usersToUse {
		var tagNumber *TagNumber
		var score *Score

		// Some participants have tag numbers and scores
		if g.faker.Bool() {
			tn := TagNumber(i + 1)
			tagNumber = &tn
			s := Score(g.faker.Float64Range(0, 1000))
			score = &s
		}

		participants = append(participants, Participant{
			UserID:    user.UserID,
			TagNumber: tagNumber,
			Response: Response{
				Status:    g.faker.RandomString([]string{"ACCEPTED", "DECLINED", "PENDING"}),
				Timestamp: g.faker.Date(),
				Value:     g.faker.Sentence(g.faker.Number(1, 5)),
			},
			Score: score,
		})
	}

	return Round{
		ID:             RoundID(uuid.New().String()),
		Title:          Title(g.faker.Sentence(g.faker.Number(2, 5))),
		Description:    Description(g.faker.Paragraph(1, 3, 3, "\n")),
		Location:       Location(g.faker.Address().Street),
		EventType:      &eventType,
		StartTime:      StartTime(g.faker.DateRange(time.Now(), time.Now().AddDate(0, 1, 0))),
		Finalized:      Finalized(g.faker.Bool()),
		CreatedBy:      createdBy,
		State:          stateTypes[g.faker.Number(0, len(stateTypes)-1)],
		Participants:   participants,
		EventMessageID: g.faker.UUID(),
		CreatedAt:      g.faker.DateRange(time.Now().AddDate(0, -1, 0), time.Now()),
		UpdatedAt:      time.Now(),
	}
}

// GenerateRoundWithConstraints creates a round with specific constraints
func (g *TestDataGenerator) GenerateRoundWithConstraints(options RoundOptions) Round {
	round := g.GenerateRound(options.CreatedBy, options.ParticipantCount, options.Users)

	// Apply constraints if provided
	if options.Title != "" {
		round.Title = Title(options.Title)
	}

	if options.State != "" {
		round.State = options.State
	}

	if options.StartTime != nil {
		round.StartTime = *options.StartTime
	}

	if options.Finalized != nil {
		round.Finalized = *options.Finalized
	}

	return round
}

// RoundOptions provides options for creating constrained rounds
type RoundOptions struct {
	CreatedBy        DiscordID
	ParticipantCount int
	Users            []User
	Title            string
	State            RoundState
	StartTime        *StartTime
	Finalized        *Finalized
}

// GenerateScore creates a test score for a round
func (g *TestDataGenerator) GenerateScore(round Round) ScoreModel {
	scoreInfos := make([]ScoreInfo, 0, len(round.Participants))

	for _, participant := range round.Participants {
		// Only include participants with scores
		if participant.Score != nil {
			scoreInfos = append(scoreInfos, ScoreInfo{
				UserID:    participant.UserID,
				Score:     *participant.Score,
				TagNumber: participant.TagNumber,
			})
		} else if g.faker.Bool() {
			// Add some random scores for participants that don't have them
			score := Score(g.faker.Float64Range(0, 1000))
			scoreInfos = append(scoreInfos, ScoreInfo{
				UserID:    participant.UserID,
				Score:     score,
				TagNumber: participant.TagNumber,
			})
		}
	}

	return ScoreModel{
		RoundID:   round.ID,
		RoundData: scoreInfos,
		Source:    g.faker.RandomString([]string{"API", "DISCORD", "MANUAL"}),
	}
}

// GenerateLeaderboard creates a test leaderboard based on multiple rounds
func (g *TestDataGenerator) GenerateLeaderboard(rounds []Round, isActive bool) Leaderboard {
	entries := make([]LeaderboardEntry, 0)
	userScores := make(map[DiscordID]float64)
	userTags := make(map[DiscordID]*TagNumber)

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
		entries = append(entries, LeaderboardEntry{
			TagNumber: userTags[userID],
			UserID:    userID,
		})
	}

	return Leaderboard{
		ID:                  int64(g.faker.Number(1, 1000)),
		LeaderboardData:     entries,
		IsActive:            isActive,
		UpdateSource:        g.faker.RandomString([]string{"ROUND_COMPLETION", "MANUAL_UPDATE", "SCHEDULED_UPDATE"}),
		UpdateID:            rounds[len(rounds)-1].ID, // Last round as update source
		RequestingDiscordID: DiscordID(fmt.Sprintf("%d", g.faker.Number(100000000, 999999999))),
	}
}

// GenerateTestData creates a complete set of test data with interrelated entities
func (g *TestDataGenerator) GenerateTestData(userCount, roundCount int) ([]User, []Round, []ScoreModel, Leaderboard) {
	users := g.GenerateUsers(userCount)
	rounds := make([]Round, roundCount)
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
		State:            "PUBLISHED",
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
