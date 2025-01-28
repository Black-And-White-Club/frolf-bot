package rounddb

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
)

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// CreateRound creates a new round in the database.
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
func (db *RoundDBImpl) GetRound(ctx context.Context, roundID string) (*roundtypes.Round, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}
	return &round, nil
}

// UpdateRound updates an existing round in the database.
func (db *RoundDBImpl) UpdateRound(ctx context.Context, roundID string, round *roundtypes.Round) error {
	_, err := db.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}
	return nil
}

// DeleteRound "soft deletes" a round by setting its state to DELETED.
func (db *RoundDBImpl) DeleteRound(ctx context.Context, roundID string) error {
	return db.UpdateRoundState(ctx, roundID, roundtypes.RoundStateDeleted)
}

// UpdateParticipant updates a participant's response or tag number in a round.
func (db *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID string, participant roundtypes.RoundParticipant) error {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find the participant and update their response or tag number
	found := false
	for i, p := range round.Participants {
		if p.DiscordID == participant.DiscordID {
			// Update response if provided
			if participant.Response != "" {
				round.Participants[i].Response = participant.Response
			}
			// Update tag number if provided
			if participant.TagNumber != 0 {
				round.Participants[i].TagNumber = participant.TagNumber
			}
			// Update score if provided
			if participant.Score != nil {
				round.Participants[i].Score = participant.Score
			}
			found = true
			break
		}
	}
	if !found {
		// If participant not found, add them to the round
		round.Participants = append(round.Participants, participant)
	}

	_, err = db.DB.NewUpdate().
		Model(&round).
		Set("participants = ?", round.Participants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant response: %w", err)
	}

	return nil
}

// UpdateRoundState updates the state of a round.
func (db *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID string, state roundtypes.RoundState) error {
	var round roundtypes.Round
	_, err := db.DB.NewUpdate().
		Model(&round).
		Set("state = ?", state).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	return nil
}

// GetUpcomingRounds retrieves rounds that are upcoming within the given time range.
func (db *RoundDBImpl) GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*roundtypes.Round, error) {
	var rounds []*roundtypes.Round
	err := db.DB.NewSelect().
		Model(&rounds).
		Where("state = ?", roundtypes.RoundStateUpcoming).
		Where("start_time >= ?", now).
		Where("start_time <= ?", oneHourFromNow).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}
	return rounds, nil
}

// UpdateParticipantScore updates the score for a participant in a round.
func (db *RoundDBImpl) UpdateParticipantScore(ctx context.Context, roundID string, participantID string, score int) error {
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
		if p.DiscordID == participantID {
			round.Participants[i].Score = &score // Update the Score field
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
		Set("participants = ?", round.Participants). // Let bun handle marshaling
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant score: %w", err)
	}

	return nil
}

// GetParticipantsWithResponses retrieves participants with the specified responses from a round.
func (db *RoundDBImpl) GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...roundtypes.Response) ([]roundtypes.RoundParticipant, error) {
	var round roundtypes.Round
	err := db.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}
	var participants []roundtypes.RoundParticipant
	for _, p := range round.Participants {
		for _, r := range responses {
			if p.Response == r {
				participants = append(participants, p)
			}
		}
	}
	return participants, nil
}

// GetRoundState retrieves the state of a round.
func (db *RoundDBImpl) GetRoundState(ctx context.Context, roundID string) (roundtypes.RoundState, error) {
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
func (db *RoundDBImpl) GetParticipants(ctx context.Context, roundID string) ([]roundtypes.RoundParticipant, error) {
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
