// db/round.go

package rounddb

import (
	"context"
	"fmt"
	"time"

	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/uptrace/bun"
)

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// CreateRoundScores implements the CreateRoundScores method of the RoundDB interface.
func (r *RoundDBImpl) CreateRoundScores(ctx context.Context, roundID int64, scores map[string]int) error {
	// 1. Fetch the round from the database
	round, err := r.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Update the round's Scores field
	round.Scores = scores

	// 3. Update the round in the database
	_, err = r.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Column("scores").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round scores: %w", err)
	}

	return nil
}

// GetRounds retrieves all rounds.
func (r *RoundDBImpl) GetRounds(ctx context.Context) ([]*Round, error) {
	var rounds []*Round
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
func (r *RoundDBImpl) GetRound(ctx context.Context, roundID int64) (*Round, error) {
	var round Round
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
func (r *RoundDBImpl) CreateRound(ctx context.Context, input rounddto.CreateRoundInput) (*Round, error) {
	round := &Round{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
		CreatorID: input.DiscordID,
		State:     RoundStateUpcoming, // Set initial state to "UPCOMING"
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
func (r *RoundDBImpl) UpdateRound(ctx context.Context, roundID int64, updates map[string]interface{}) error {
	_, err := r.DB.NewUpdate().
		Model((*Round)(nil)).
		Where("id = ?", roundID).
		Set("title = ?", updates["title"]).
		Set("location = ?", updates["location"]).
		Set("event_type = ?", updates["eventType"]).
		Set("date = ?", updates["date"]).
		Set("time = ?", updates["time"]).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	return nil
}

// DeleteRound deletes a round by ID.
func (r *RoundDBImpl) DeleteRound(ctx context.Context, roundID int64) error {
	_, err := r.DB.NewDelete().
		Model((*Round)(nil)).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}
	return nil
}

func (r *RoundDBImpl) RecordScores(ctx context.Context, roundID int64, scores map[string]int) error {
	// 1. Fetch the round from the database
	round, err := r.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// 2. Update the round's Scores field
	round.Scores = scores

	// 3. Update the round in the database
	_, err = r.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Column("scores").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round scores: %w", err)
	}

	return nil
}

// SubmitScore updates the scores map for a round in the database.
func (r *RoundDBImpl) SubmitScore(ctx context.Context, roundID int64, discordID string, score int) error {
	// 1. Fetch the round from the database
	round, err := r.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// 2. Update the score in the PendingScores slice
	found := false
	for i, s := range round.PendingScores {
		if s.ParticipantID == discordID {
			round.PendingScores[i].Score = score
			found = true
			break
		}
	}
	if !found {
		round.PendingScores = append(round.PendingScores, Score{
			ParticipantID: discordID,
			Score:         score,
		})
	}

	// 3. Update the round in the database
	_, err = r.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Column("pending_scores"). // Update the pending_scores column
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round scores: %w", err)
	}

	return nil
}

// UpdateParticipantResponse updates a participant's response or tag number in a round.
func (r *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID int64, participant Participant) error {
	var round Round
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
func (r *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error {
	_, err := r.DB.NewUpdate().
		Model((*Round)(nil)).
		Set("state = ?", state).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	return nil
}

// GetUpcomingRounds retrieves rounds that are upcoming within the given time range.
func (r *RoundDBImpl) GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*Round, error) {
	var rounds []*Round
	err := r.DB.NewSelect().
		Model(&rounds).
		Where("state = ? AND date = ? AND time BETWEEN ? AND ?", RoundStateUpcoming, now.Format("2006-01-02"), now.Format("15:04"), oneHourFromNow.Format("15:04")).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}
	return rounds, nil
}

// IsRoundFinalized checks if a round is finalized.
func (r *RoundDBImpl) IsRoundFinalized(ctx context.Context, roundID int64) (bool, error) {
	var round Round
	err := r.DB.NewSelect().
		Model(&round).
		Column("finalized").
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check round finalized status: %w", err)
	}
	return round.Finalized, nil
}

// GetRoundState retrieves the state of a round.
func (r *RoundDBImpl) GetRoundState(ctx context.Context, roundID int64) (RoundState, error) {
	var round Round
	err := r.DB.NewSelect().
		Model(&round).
		Column("state").
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get round state: %w", err)
	}
	return round.State, nil
}

// RoundExists checks if a round with the given ID exists.
func (r *RoundDBImpl) RoundExists(ctx context.Context, roundID int64) (bool, error) {
	exists, err := r.DB.NewSelect().
		Model((*Round)(nil)).
		Where("id = ?", roundID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if round exists: %w", err)
	}
	return exists, nil
}

// GetParticipant retrieves a specific participant from a round by Discord ID.
func (r *RoundDBImpl) GetParticipant(ctx context.Context, roundID int64, discordID string) (Participant, error) {
	var round Round
	err := r.DB.NewSelect().
		Model(&round).
		Where("id = ?", roundID).
		Relation("Participants").
		Scan(ctx)
	if err != nil {
		return Participant{}, fmt.Errorf("failed to fetch round: %w", err)
	}

	for _, p := range round.Participants {
		if p.DiscordID == discordID {
			return p, nil
		}
	}

	return Participant{}, fmt.Errorf("participant not found")
}
