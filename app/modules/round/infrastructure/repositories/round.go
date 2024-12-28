package rounddb

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// CreateRound creates a new round in the database.
func (r *RoundDBImpl) CreateRound(ctx context.Context, round *Round) error { // Accept the Round model
	_, err := r.DB.NewInsert().
		Model(round).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}
	return nil
}

// GetRound retrieves a specific round by ID.
func (r *RoundDBImpl) GetRound(ctx context.Context, roundID string) (*Round, error) {
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

// UpdateRound updates an existing round in the database.
func (r *RoundDBImpl) UpdateRound(ctx context.Context, roundID string, round *Round) error {
	updateQuery := r.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID)

	// Dynamically add Set clauses based on non-zero values
	if round.Title != "" {
		updateQuery = updateQuery.Set("title = ?", round.Title)
	}
	if round.Location != "" {
		updateQuery = updateQuery.Set("location = ?", round.Location)
	}
	if round.EventType != nil {
		updateQuery = updateQuery.Set("event_type = ?", round.EventType)
	}
	if !round.Date.IsZero() {
		updateQuery = updateQuery.Set("date = ?", round.Date)
	}
	if !round.Time.IsZero() {
		updateQuery = updateQuery.Set("time = ?", round.Time)
	}

	_, err := updateQuery.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}
	return nil
}

// DeleteRound "soft deletes" a round by setting its state to DELETED.
func (r *RoundDBImpl) DeleteRound(ctx context.Context, roundID string) error {
	return r.UpdateRoundState(ctx, roundID, RoundStateDeleted)
}

// UpdateParticipant updates a participant's response or tag number in a round.
func (r *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID string, participant Participant) error {
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
func (r *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID string, state RoundState) error {
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

// UpdateParticipantScore updates the score for a participant in a round.
func (r *RoundDBImpl) UpdateParticipantScore(ctx context.Context, roundID, participantID string, score int) error {
	// 1. Fetch the round
	round, err := r.GetRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Find the participant and update their score
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

	// 3. Update the round in the database
	_, err = r.DB.NewUpdate().
		Model(round).
		Where("id = ?", roundID).
		Column("participants"). // Update the participants column
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant score: %w", err)
	}

	return nil
}

// GetParticipantsWithResponses retrieves participants with the specified responses from a round.
func (r *RoundDBImpl) GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...Response) ([]Participant, error) {
	var participants []Participant
	err := r.DB.NewSelect().
		Model(&participants).
		Where("round_id = ?", roundID). // Assuming you have a round_id field in the Participant struct
		Where("response IN (?)", bun.In(responses)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants: %w", err)
	}
	return participants, nil
}

// GetRoundState retrieves the state of a round.
func (r *RoundDBImpl) GetRoundState(ctx context.Context, roundID string) (RoundState, error) {
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

type ScoreUpdateType int

const (
	ScoreUpdateTypeRegular ScoreUpdateType = iota
	ScoreUpdateTypeManual
)

// LogRound logs the round data by updating the existing round entry.
func (r *RoundDBImpl) LogRound(ctx context.Context, round *Round, updateType ScoreUpdateType) error {
	updateQuery := r.DB.NewUpdate().
		Model(round).
		Where("id = ?", round.ID)

	if updateType == ScoreUpdateTypeRegular {
		// For regular updates, update the entire Participants array
		updateQuery = updateQuery.Set("participants = ?", round.Participants)
	} else {
		// For manual updates, append the new score to the Participants array
		// Assuming the new score is in round.Participants[0]
		updateQuery = updateQuery.Set("participants = jsonb_insert(participants, '{0}', ?)", round.Participants[0])
	}

	_, err := updateQuery.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to log round: %w", err)
	}
	return nil
}
