package rounddb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// CreateRound creates a new round in the database and retrieves the generated ID.
func (db *RoundDBImpl) CreateRound(ctx context.Context, round *roundtypes.Round) error {
	_, err := db.DB.NewInsert().
		Model(round).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}
	return nil
}

// GetRound retrieves a specific round by ID.
func (db *RoundDBImpl) GetRound(ctx context.Context, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	round := new(roundtypes.Round)
	err := db.DB.NewSelect().
		Model(round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return a more specific error when the round is not found
			return nil, fmt.Errorf("round with ID %s not found", roundID)
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}
	return round, nil
}

// GetParticipant retrieves a participant's information for a specific round
func (db *RoundDBImpl) GetParticipant(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Look for the participant in the round's participants
	for _, p := range round.Participants {
		if p.UserID == userID {
			return &p, nil
		}
	}

	// Participant not found
	return nil, nil
}

// RemoveParticipant removes a participant from a round and returns updated participants
func (db *RoundDBImpl) RemoveParticipant(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
	// First, fetch the round
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("round with ID %s not found", roundID)
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find and remove the participant
	found := false
	updatedParticipants := make([]roundtypes.Participant, 0, len(round.Participants))
	for _, p := range round.Participants {
		if p.UserID != userID {
			updatedParticipants = append(updatedParticipants, p)
		} else {
			found = true
		}
	}

	if !found {
		// Participant wasn't in the round - return current participants (graceful handling)
		return round.Participants, nil
	}

	// Update the round with the modified participants list
	_, err = db.DB.NewUpdate().
		Model(&round).
		Set("participants = ?", updatedParticipants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to remove participant: %w", err)
	}

	return updatedParticipants, nil
}

// convertToDomainRound converts a database Round model to domain Round model
func convertToDomainRound(dbRound Round) *roundtypes.Round {
	return &roundtypes.Round{
		ID:             dbRound.ID,
		Title:          dbRound.Title,
		Description:    &dbRound.Description,
		Location:       &dbRound.Location,
		EventType:      dbRound.EventType,
		StartTime:      &dbRound.StartTime,
		Finalized:      dbRound.Finalized,
		CreatedBy:      dbRound.CreatedBy,
		State:          dbRound.State,
		Participants:   dbRound.Participants,
		EventMessageID: dbRound.EventMessageID,
	}
}

// UpdateRound updates specific fields of an existing round in the database and returns the updated round.
func (db *RoundDBImpl) UpdateRound(ctx context.Context, roundID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error) {
	// Convert domain model to database model for the update
	dbRound := Round{
		ID: roundID,
	}

	// Only set fields that have values in the domain model
	if round.Title != "" {
		dbRound.Title = round.Title
	}
	if round.Description != nil && *round.Description != "" {
		dbRound.Description = *round.Description
	}
	if round.Location != nil && *round.Location != "" {
		dbRound.Location = *round.Location
	}
	if round.StartTime != nil {
		dbRound.StartTime = *round.StartTime
	}
	if round.EventType != nil {
		dbRound.EventType = round.EventType
	}

	var updatedDbRound Round

	// Now use the database model for both Model() and scan target
	_, err := db.DB.NewUpdate().
		Model(&dbRound).
		OmitZero(). // This will ignore zero values
		Where("id = ?", roundID).
		Returning("*").
		Exec(ctx, &updatedDbRound)
	if err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	// Convert back to domain model and return COMPLETE round
	return convertToDomainRound(updatedDbRound), nil
}

// DeleteRound "soft deletes" a round by setting its state to DELETED.
func (db *RoundDBImpl) DeleteRound(ctx context.Context, roundID sharedtypes.RoundID) error {
	// Validate the round ID isn't nil/zero
	if roundID == sharedtypes.RoundID(uuid.Nil) {
		slog.Error("attempted to delete round with nil UUID")
		return fmt.Errorf("cannot delete round: nil UUID provided")
	}

	// Check if the round exists first
	exists, err := db.DB.NewSelect().
		Model(&roundtypes.Round{}).
		Where("id = ?", roundID).
		Exists(ctx)
	if err != nil {
		slog.Error("failed to check if round exists", "error", err)
		return fmt.Errorf("failed to check if round exists: %w", err)
	}

	if !exists {
		slog.Warn("attempted to delete non-existent round")
		return fmt.Errorf("round with ID %s does not exist", roundID.String())
	}

	// Update the round state
	res, err := db.DB.NewUpdate().
		Model(&roundtypes.Round{}).
		Set("state = ?", roundtypes.RoundState(roundtypes.RoundStateDeleted)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		slog.Error("failed to update round state", "error", err)
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.Error("failed to get rows affected", "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("no rows affected when deleting round")
		return fmt.Errorf("no rows affected when deleting round %s", roundID.String())
	}

	slog.Info("round deleted from DB")
	return nil
}

// UpdateParticipant updates a participant's response or tag number in a round and returns updated domain participants.
func (db *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error) {
	// Log incoming participant data
	tagNumberStr := "<nil>"
	if participant.TagNumber != nil {
		tagNumberStr = fmt.Sprintf("%d", *participant.TagNumber)
	}

	// Start a transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		// Replaced slog.Error with attribute
		fmt.Printf(">>> ERROR: Failed to begin transaction: %v\n", err)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var dbRound Round
	err = tx.NewSelect().
		Model(&dbRound).
		Where("id = ?", roundID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		// Replaced slog.Error with attribute
		fmt.Printf(">>> ERROR: Failed to fetch round %v: %v\n", roundID, err)
		return nil, fmt.Errorf("fetch round: %w", err)
	}

	// Initialize participants if null
	if dbRound.Participants == nil {
		fmt.Println(">>> Participants was nil, initializing empty array") // Replaced slog.Info
		dbRound.Participants = []roundtypes.Participant{}
	}

	// Modify participants
	updated := false
	for i := range dbRound.Participants { // Iterate by index to modify in place
		p := &dbRound.Participants[i] // Use pointer to modify the slice element

		if p.UserID == participant.UserID {
			// Replaced slog.Info with attributes
			fmt.Printf(">>> Found existing participant %s at index %d, updating...\n",
				string(p.UserID),
				i,
			)

			if participant.Response != "" {
				// Handle nil pointer gracefully for old value logging
				oldResponse := string(p.Response)
				fmt.Printf(">>> Updating response for user %s: Old=%s, New=%s\n",
					string(p.UserID),
					oldResponse,
					string(participant.Response),
				)
				p.Response = participant.Response
			}

			// Always update TagNumber, whether it's setting a new value or clearing an existing one.
			// Remove the 'if participant.TagNumber != nil' condition.
			oldTag := "<nil>"
			if p.TagNumber != nil {
				oldTag = fmt.Sprintf("%d", *p.TagNumber)
			}
			newTagStr := "<nil>"
			if participant.TagNumber != nil {
				newTagStr = fmt.Sprintf("%d", *participant.TagNumber)
			}

			// Only log if there's an actual change in TagNumber to avoid excessive logging
			tagChanged := false
			if (p.TagNumber == nil && participant.TagNumber != nil) ||
				(p.TagNumber != nil && participant.TagNumber == nil) ||
				(p.TagNumber != nil && participant.TagNumber != nil && *p.TagNumber != *participant.TagNumber) {
				tagChanged = true
			}

			if tagChanged {
				fmt.Printf(">>> Updating tag number for user %s: Old=%s, New=%s\n",
					string(p.UserID),
					oldTag,
					newTagStr,
				)
			}
			p.TagNumber = participant.TagNumber // This line now unconditionally assigns the new TagNumber (can be nil)

			if participant.Score != nil {
				// Handle nil pointer gracefully for old value logging
				oldScore := "<nil>"
				if p.Score != nil {
					oldScore = fmt.Sprintf("%v", *p.Score)
				}
				newScore := "<nil>"
				if participant.Score != nil {
					newScore = fmt.Sprintf("%v", *participant.Score)
				}
				fmt.Printf(">>> Updating score for user %s: Old=%s, New=%s\n",
					string(p.UserID),
					oldScore,
					newScore,
				)
				p.Score = participant.Score
			}

			updated = true
			break // Found the participant, exit the loop
		}
	}

	if !updated {
		// Replaced slog.Info with attributes
		fmt.Printf(">>> Adding new participant: UserID=%s, Response=%s, TagNumber=%s, Score=%v\n",
			string(participant.UserID),
			string(participant.Response),
			tagNumberStr,
			fmt.Sprintf("%v", participant.Score),
		)
		dbRound.Participants = append(dbRound.Participants, participant)
	}

	// Replaced slog.Info with attributes - Note: Printing the whole slice (%v) can be verbose
	fmt.Printf(">>> After update: ParticipantsCount=%d, Participants=%v\n",
		len(dbRound.Participants),
		dbRound.Participants,
	)

	// Update the record
	_, err = tx.NewUpdate().
		Model(&dbRound).
		Set("participants = ?", dbRound.Participants). // Assuming participants are stored as JSONB or similar
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		// Replaced slog.Error with attribute
		fmt.Printf(">>> ERROR: Failed to update round %v: %v\n", roundID, err)
		return nil, fmt.Errorf("update round: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		// Replaced slog.Error with attribute
		fmt.Printf(">>> ERROR: Failed to commit transaction: %v\n", err)
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// Replaced slog.Info with attribute
	fmt.Printf(">>> Update successful. Final participants count: %d\n", len(dbRound.Participants))

	return dbRound.Participants, nil
}

// UpdateRoundState updates the state of a round.
func (db *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID sharedtypes.RoundID, state roundtypes.RoundState) error {
	_, err := db.DB.NewUpdate().
		Model(&roundtypes.Round{}).
		Set("state = ?", state).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	return nil
}

// GetUpcomingRounds retrieves rounds that are upcoming
func (db *RoundDBImpl) GetUpcomingRounds(ctx context.Context) ([]*roundtypes.Round, error) {
	var rounds []*roundtypes.Round
	err := db.DB.NewSelect().
		Model(&rounds).
		Where("state = ?", roundtypes.RoundStateUpcoming).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}
	return rounds, nil
}

// UpdateParticipantScore updates the score for a participant in a round.
func (db *RoundDBImpl) UpdateParticipantScore(ctx context.Context, roundID sharedtypes.RoundID, participantID sharedtypes.DiscordID, score sharedtypes.Score) error {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find the participant and update their score
	found := false
	for i, p := range round.Participants {
		if p.UserID == participantID {
			round.Participants[i].Score = &score
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("participant not found in round")
	}

	// Update the round in the database
	_, err = db.DB.NewUpdate().
		Model(&round).
		Set("participants = ?", round.Participants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant score: %w", err)
	}

	return nil
}

// GetParticipantsWithResponses retrieves participants with the specified responses from a round.
func (db *RoundDBImpl) GetParticipantsWithResponses(ctx context.Context, roundID sharedtypes.RoundID, responses ...string) ([]roundtypes.Participant, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	var participants []roundtypes.Participant
	for _, p := range round.Participants {
		for _, response := range responses {
			if string(p.Response) == response {
				participants = append(participants, p)
				break
			}
		}
	}

	return participants, nil
}

// GetRoundState retrieves the state of a round.
func (db *RoundDBImpl) GetRoundState(ctx context.Context, roundID sharedtypes.RoundID) (roundtypes.RoundState, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Column("state").
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get round state: %w", err)
	}
	return round.State, nil
}

// LogRound logs the round data by updating the existing round entry.
func (db *RoundDBImpl) LogRound(ctx context.Context, round *roundtypes.Round) error {
	_, err := db.DB.NewUpdate().
		Model(round).
		Where("id = ?", round.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to log round: %w", err)
	}
	return nil
}

// GetParticipants retrieves all participants from a round.
func (db *RoundDBImpl) GetParticipants(ctx context.Context, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return a more specific error when the round is not found
			return nil, fmt.Errorf("round with ID %s not found", roundID)
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	return round.Participants, nil
}

// UpdateEventMessageID updates the EventMessageID(messageID) for an existing round.
func (db *RoundDBImpl) UpdateEventMessageID(ctx context.Context, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error) {
	var dbRound Round

	_, err := db.DB.NewUpdate().
		Model(&dbRound).
		Set("event_message_id = ?", eventMessageID).
		Where("id = ?", roundID).
		Returning("*").
		Exec(ctx, &dbRound)
	if err != nil {
		return nil, fmt.Errorf("failed to update discord event ID and return row: %w", err)
	}

	// Convert from DB model to domain model
	round := &roundtypes.Round{
		ID:             dbRound.ID,
		Title:          dbRound.Title,
		Description:    &dbRound.Description,
		Location:       &dbRound.Location,
		EventType:      dbRound.EventType,
		StartTime:      &dbRound.StartTime,
		Finalized:      dbRound.Finalized,
		CreatedBy:      dbRound.CreatedBy,
		State:          dbRound.State,
		Participants:   dbRound.Participants,
		EventMessageID: dbRound.EventMessageID,
	}

	return round, nil
}

// GetEventMessageID retrieves the EventMessageID for a given round.
func (db *RoundDBImpl) GetEventMessageID(ctx context.Context, roundID sharedtypes.RoundID) (string, error) {
	var round Round
	err := db.DB.NewSelect().
		Model(&round).
		Column("event_message_id").
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get event message ID: %w", err)
	}

	return round.EventMessageID, nil // Return the string directly
}

// / UpdateRoundsAndParticipants updates multiple rounds and participants in a single transaction.
func (db *RoundDBImpl) UpdateRoundsAndParticipants(ctx context.Context, updates []roundtypes.RoundUpdate) error {
	return db.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for _, update := range updates {
			// Only update the participants column, not the entire round
			_, err := tx.NewUpdate().
				Model((*roundtypes.Round)(nil)).
				Set("participants = ?", update.Participants).
				Where("id = ?", update.RoundID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update participants for round %s: %w", update.RoundID, err)
			}
		}
		return nil
	})
}

// TagUpdates can be removed since it's no longer needed, or simplified if used elsewhere
func (db *RoundDBImpl) TagUpdates(ctx context.Context, bun bun.IDB, round *roundtypes.Round) error {
	// This method should specify which columns to update to avoid the NULL constraint issue
	_, err := bun.NewUpdate().
		Model(round).
		Column("participants"). // Only update participants column
		Where("id = ?", round.ID).
		Exec(ctx)
	return err
}
