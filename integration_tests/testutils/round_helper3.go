package testutils

import (
	"context"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// Add this method to the existing file:

// ParticipantData represents test data for creating a participant
type ParticipantData struct {
	UserID   sharedtypes.DiscordID
	Response roundtypes.Response
	Score    *sharedtypes.Score
}

// CreateRoundWithParticipants creates a round with multiple participants directly in the database
func (h *RoundTestHelper) CreateRoundWithParticipants(t *testing.T, db bun.IDB, creatorID sharedtypes.DiscordID, participantsData []ParticipantData) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundOptions := RoundOptions{
		CreatedBy:        DiscordID(creatorID),
		ParticipantCount: 0,
		Users:            []User{},
		State:            roundtypes.RoundStateInProgress, // Default to in progress for score updates
	}
	roundData := generator.GenerateRoundWithConstraints(roundOptions)

	// Create participants from the provided data
	participants := make([]roundtypes.Participant, len(participantsData))
	for i, pd := range participantsData {
		// Generate a tag number for each participant
		tagNumber := sharedtypes.TagNumber(generator.GenerateTagNumber())

		participants[i] = roundtypes.Participant{
			UserID:    pd.UserID,
			Response:  pd.Response,
			TagNumber: &tagNumber,
			Score:     pd.Score,
		}
	}
	roundData.Participants = participants

	// Set event message ID for testing (required for score updates)
	roundData.EventMessageID = "test-event-message-id"

	// Make sure all required fields are set for DB insertion

	if roundData.Description == nil {
		desc := roundtypes.Description("Test round description")
		roundData.Description = &desc
	}
	if roundData.Location == nil {
		location := roundtypes.Location("Test Location")
		roundData.Location = &location
	}
	if roundData.StartTime == nil {
		startTime := sharedtypes.StartTime(time.Now().Add(time.Hour))
		roundData.StartTime = &startTime
	}

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:             roundData.ID,
		Title:          roundData.Title,
		Description:    *roundData.Description,
		Location:       *roundData.Location,
		EventType:      roundData.EventType,
		StartTime:      *roundData.StartTime,
		Finalized:      roundData.Finalized,
		CreatedBy:      roundData.CreatedBy,
		State:          roundData.State,
		Participants:   roundData.Participants,
		EventMessageID: roundData.EventMessageID,
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round with participants: %v", err)
	}

	t.Logf("Created round %s with %d participants", roundData.ID, len(participants))

	return roundData.ID
}
