// internal/db/bundb/round.go
package bundb

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

// roundDB implements the RoundDB interface using bun.
type roundDB struct {
	db *bun.DB
}

// GetRounds retrieves all rounds.
func (r *roundDB) GetRounds(ctx context.Context) ([]*models.Round, error) {
	var rounds []*models.Round
	err := r.db.NewSelect().
		Model(&rounds).
		Relation("Participants").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rounds: %w", err)
	}
	return rounds, nil
}

// GetRound retrieves a specific round by ID.
func (r *roundDB) GetRound(ctx context.Context, roundID int64) (*models.Round, error) {
	var round models.Round
	err := r.db.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Relation("Participants").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}
	return &round, nil
}

// CreateRound creates a new round in the database.
func (r *roundDB) CreateRound(ctx context.Context, round *models.Round) (*models.Round, error) {
	_, err := r.db.NewInsert().
		Model(round).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create round: %w", err)
	}
	return round, nil
}

// UpdateRound updates an existing round in the database.
func (r *roundDB) UpdateRound(ctx context.Context, round *models.Round) error {
	_, err := r.db.NewUpdate().
		Model(round).
		Where("id = ?", round.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}
	return nil
}

// DeleteRound deletes a round by ID.
func (r *roundDB) DeleteRound(ctx context.Context, roundID int64, userID string) error {
	_, err := r.db.NewDelete().
		Model((*models.Round)(nil)).
		Where("id = ? AND creator_id = ?", roundID, userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}
	return nil
}

// FindParticipant finds a participant in a round.
func (r *roundDB) FindParticipant(ctx context.Context, roundID int64, discordID string) (*models.Participant, error) {
	var participant models.Participant
	err := r.db.NewSelect().
		Model(&participant).
		Where("discord_id = ? AND round_id = ?", discordID, roundID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// UpdateParticipantResponse updates a participant's response in a round.
func (r *roundDB) UpdateParticipantResponse(ctx context.Context, roundID int64, discordID string, response models.Response) (*models.Round, error) {
	// This logic should be partially handled in the service layer (refreshing the round).
	_, err := r.db.NewUpdate().
		Model((*models.Participant)(nil)).
		Set("response = ?", response).
		Where("discord_id = ? AND round_id = ?", discordID, roundID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}
	return r.GetRound(ctx, roundID)
}

// UpdateRoundState updates the state of a round.
func (r *roundDB) UpdateRoundState(ctx context.Context, roundID int64, state models.RoundState) error {
	_, err := r.db.NewUpdate().
		Model((*models.Round)(nil)).
		Set("state = ?", state).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	return nil
}

// GetUpcomingRounds retrieves rounds that are upcoming within the given time range.
func (r *roundDB) GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*models.Round, error) {
	var rounds []*models.Round
	err := r.db.NewSelect().
		Model(&rounds).
		Where("state = ? AND date = ? AND time BETWEEN ? AND ?", models.RoundStateUpcoming, now.Format("2006-01-02"), now.Format("15:04"), oneHourFromNow.Format("15:04")).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}
	return rounds, nil
}
