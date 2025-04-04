package rounddb

import (
	"context"
	"database/sql"
	"fmt"

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
	// In RoundDBImpl.CreateRound
	err := db.DB.NewInsert().
		Model(round).
		ExcludeColumn("id").
		Returning("id").
		Scan(ctx, &round.ID)
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

// RemoveParticipant removes a participant from a round
func (db *RoundDBImpl) RemoveParticipant(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) error {
	// First, fetch the round
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
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
		// Participant wasn't in the round
		return nil
	}

	// Update the round with the modified participants list
	_, err = db.DB.NewUpdate().
		Model(&round).
		Set("participants = ?", updatedParticipants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	return nil
}

// UpdateRound updates an existing round in the database.
func (db *RoundDBImpl) UpdateRound(ctx context.Context, roundID sharedtypes.RoundID, round *roundtypes.Round) error {
	result, err := db.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteRound "soft deletes" a round by setting its state to DELETED.
func (db *RoundDBImpl) DeleteRound(ctx context.Context, roundID sharedtypes.RoundID) error {
	return db.UpdateRoundState(ctx, roundID, roundtypes.RoundState(roundtypes.RoundStateDeleted))
}

// UpdateParticipant updates a participant's response or tag number in a round and returns the updated participant lists.
func (db *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find the participant and update their response or tag number
	found := false
	for i, p := range round.Participants {
		if p.UserID == participant.UserID {
			if participant.Response != "" {
				round.Participants[i].Response = participant.Response
			}

			if participant.TagNumber != nil {
				round.Participants[i].TagNumber = participant.TagNumber
			}
			if participant.Score != nil {
				round.Participants[i].Score = participant.Score
			}
			found = true
			break
		}
	}

	if !found {
		round.Participants = append(round.Participants, participant)
	}

	_, err = db.DB.NewUpdate().
		Model(&round).
		Set("participants = ?", round.Participants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}

	return round.Participants, nil
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
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	return round.Participants, nil
}

// UpdateEventMessageID updates the EventMessageID(messageID) for an existing round.
func (db *RoundDBImpl) UpdateEventMessageID(ctx context.Context, roundID sharedtypes.RoundID, eventMessageID string) error {
	_, err := db.DB.NewUpdate().
		Model(&roundtypes.Round{}).
		Set("event_message_id =?", eventMessageID).
		Where("id =?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update discord event ID: %w", err)
	}
	return nil
}

// GetEventMessageID retrieves the EventMessageID for a given round.
func (db *RoundDBImpl) GetEventMessageID(ctx context.Context, roundID sharedtypes.RoundID) (*sharedtypes.RoundID, error) {
	var eventMessageID sharedtypes.RoundID

	err := db.DB.NewSelect().
		Model((*roundtypes.Round)(nil)). // Using nil because we only need one field
		Column("event_message_id").
		Where("id = ?", roundID).
		Scan(ctx, &eventMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch EventMessageID for round %d: %w", roundID, err)
	}

	return &eventMessageID, nil
}

// UpdateRound updates a round in the database.
func (db *RoundDBImpl) TagUpdates(ctx context.Context, bun bun.IDB, round *roundtypes.Round) error {
	_, err := bun.NewUpdate().Model(round).Where("id = ?", round.ID).Exec(ctx)
	return err
}

// UpdateRoundsAndParticipants updates multiple rounds and participants in a single transaction.
func (db *RoundDBImpl) UpdateRoundsAndParticipants(ctx context.Context, updates []roundtypes.RoundUpdate) error {
	return db.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for _, update := range updates {
			if err := db.TagUpdates(ctx, tx, &roundtypes.Round{
				ID:           update.RoundID,
				Participants: update.Participants,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
