// db/round.go

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetRounds(ctx context.Context) ([]*models.Round, error)
	GetRound(ctx context.Context, roundID int64) (*models.Round, error)
	CreateRound(ctx context.Context, round models.ScheduleRoundInput) (*models.Round, error)
	UpdateRound(ctx context.Context, roundID int64, input models.EditRoundInput) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateParticipant(ctx context.Context, roundID int64, participant models.Participant) error
	UpdateRoundState(ctx context.Context, roundID int64, state models.RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*models.Round, error)
	SubmitScore(ctx context.Context, roundID int64, discordID string, score int) error
}

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// GetRounds retrieves all rounds.
func (r *RoundDBImpl) GetRounds(ctx context.Context) ([]*models.Round, error) {
	var rounds []*models.Round
	err := r.DB.NewSelect().
		Model(&rounds).
		Relation("Participants").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rounds: %w", err)
	}
	return rounds, nil
}

// GetRound retrieves a specific round by ID.
func (r *RoundDBImpl) GetRound(ctx context.Context, roundID int64) (*models.Round, error) {
	var round models.Round
	err := r.DB.NewSelect().
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
func (r *RoundDBImpl) CreateRound(ctx context.Context, input models.ScheduleRoundInput) (*models.Round, error) {
	round := &models.Round{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
		CreatorID: input.DiscordID,
		State:     models.RoundStateUpcoming, // Set initial state to "UPCOMING"
	}
	_, err := r.DB.NewInsert().
		Model(round).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create round: %w", err)
	}
	return round, nil
}

// UpdateRound updates an existing round in the database.
func (r *RoundDBImpl) UpdateRound(ctx context.Context, roundID int64, input models.EditRoundInput) error {
	round := &models.Round{
		ID:        roundID,
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
	}

	_, err := r.DB.NewUpdate().
		Model(round).
		WherePK().
		Column("title", "location", "event_type", "date", "time"). // Use Column to specify fields
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	return nil
}

// DeleteRound deletes a round by ID.
func (r *RoundDBImpl) DeleteRound(ctx context.Context, roundID int64) error { // No userID parameter
	_, err := r.DB.NewDelete().
		Model((*models.Round)(nil)).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}
	return nil
}

// SubmitScore updates the scores map for a round in the database.
func (r *RoundDBImpl) SubmitScore(ctx context.Context, roundID int64, discordID string, score int) error {
	var round models.Round
	err := r.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	if round.Scores == nil {
		round.Scores = make(map[string]int)
	}
	round.Scores[discordID] = score

	_, err = r.DB.NewUpdate().
		Model(&round).
		Where("id = ?", roundID).
		Column("scores").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round scores: %w", err)
	}

	return nil
}

// UpdateParticipantResponse updates a participant's response or tag number in a round.
func (r *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID int64, participant models.Participant) error {
	var round models.Round
	err := r.DB.NewSelect().
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
			if participant.Response != "" { // Update response if provided
				round.Participants[i].Response = participant.Response
			}
			if participant.TagNumber != nil { // Update tag number if provided
				round.Participants[i].TagNumber = participant.TagNumber
			}
			found = true
			break
		}
	}
	if !found {
		// If participant not found, add them to the round
		round.Participants = append(round.Participants, participant)
	}

	_, err = r.DB.NewUpdate().
		Model(&round).
		Where("id = ?", roundID).
		Column("participants").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant response: %w", err)
	}

	return nil
}

// UpdateRoundState updates the state of a round.
func (r *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID int64, state models.RoundState) error {
	_, err := r.DB.NewUpdate().
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
func (r *RoundDBImpl) GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*models.Round, error) {
	var rounds []*models.Round
	err := r.DB.NewSelect().
		Model(&rounds).
		Where("state = ? AND date = ? AND time BETWEEN ? AND ?", models.RoundStateUpcoming, now.Format("2006-01-02"), now.Format("15:04"), oneHourFromNow.Format("15:04")).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}
	return rounds, nil
}
