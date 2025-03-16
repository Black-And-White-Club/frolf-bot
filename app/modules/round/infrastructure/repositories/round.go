package rounddb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// RoundDBImpl is the concrete implementation of the RoundDB interface using bun.
type RoundDBImpl struct {
	DB *bun.DB
}

// CreateRound creates a new round in the database and retrieves the generated ID.
func (db *RoundDBImpl) CreateRound(ctx context.Context, round *roundtypes.Round) error {
	slog.DebugContext(ctx, "Executing RoundDBImpl.CreateRound 🚀 ", slog.Any("round", round))
	// In RoundDBImpl.CreateRound
	err := db.DB.NewInsert().
		Model(round).
		ExcludeColumn("id").
		Returning("id").
		Scan(ctx, &round.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create round", slog.String("error", err.Error()))
		return fmt.Errorf("failed to create round: %w", err)
	}
	slog.InfoContext(ctx, "Round created successfully in DB", slog.Int64("round_id", int64(round.ID)))
	return nil
}

// GetRound retrieves a specific round by ID.
func (db *RoundDBImpl) GetRound(ctx context.Context, roundID roundtypes.ID) (*roundtypes.Round, error) {
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
func (db *RoundDBImpl) GetParticipant(ctx context.Context, roundID roundtypes.ID, userID string) (*roundtypes.Participant, error) {
	slog.DebugContext(ctx, "Executing RoundDBImpl.GetParticipant", slog.Int64("round_id", int64(roundID)), slog.String("discord_id", userID))

	round := new(roundtypes.Round)
	err := db.DB.NewSelect().
		Model(round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Look for the participant in the round's participants
	for _, p := range round.Participants {
		if p.UserID == roundtypes.UserID(userID) {
			return &p, nil
		}
	}

	// Participant not found
	return nil, nil
}

// RemoveParticipant removes a participant from a round
func (db *RoundDBImpl) RemoveParticipant(ctx context.Context, roundID roundtypes.ID, userID string) error {
	slog.DebugContext(ctx, "Executing RoundDBImpl.RemoveParticipant", slog.Int64("round_id", int64(roundID)), slog.String("discord_id", userID))

	// First, fetch the round
	round := new(roundtypes.Round)
	err := db.DB.NewSelect().
		Model(round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find and remove the participant
	found := false
	updatedParticipants := make([]roundtypes.Participant, 0, len(round.Participants))
	for _, p := range round.Participants {
		if p.UserID != roundtypes.UserID(userID) {
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
		Model(round).
		Set("participants = ?", updatedParticipants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	slog.InfoContext(ctx, "Participant removed successfully", slog.Int64("round_id", int64(roundID)), slog.String("discord_id", userID))
	return nil
}

// UpdateRound updates an existing round in the database.
func (db *RoundDBImpl) UpdateRound(ctx context.Context, roundID roundtypes.ID, round *roundtypes.Round) error {
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
func (db *RoundDBImpl) DeleteRound(ctx context.Context, roundID roundtypes.ID) error {
	return db.UpdateRoundState(ctx, roundID, roundtypes.RoundState(roundtypes.RoundStateDeleted))
}

// UpdateParticipant updates a participant's response or tag number in a round.
func (db *RoundDBImpl) UpdateParticipant(ctx context.Context, roundID roundtypes.ID, participant roundtypes.Participant) error {
	round := new(roundtypes.Round)
	err := db.DB.NewSelect().
		Model(round).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find the participant and update their response or tag number
	found := false
	for i, p := range round.Participants {
		if p.UserID == participant.UserID {
			// Update response if provided
			if participant.Response != "" {
				round.Participants[i].Response = participant.Response
			}
			// Update tag number if provided
			if participant.TagNumber != nil {
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
		Model(round).
		Set("participants = ?", round.Participants).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant response: %w", err)
	}

	return nil
}

// UpdateRoundState updates the state of a round.
func (db *RoundDBImpl) UpdateRoundState(ctx context.Context, roundID roundtypes.ID, state roundtypes.RoundState) error {
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

// GetUpcomingRounds retrieves rounds that are upcoming within the given time range.
func (db *RoundDBImpl) GetUpcomingRounds(ctx context.Context, startTime time.Time, endTime time.Time) ([]*roundtypes.Round, error) {
	var rounds []*roundtypes.Round
	err := db.DB.NewSelect().
		Model(&rounds).
		Where("start_time >= ? AND start_time <= ?", startTime, endTime).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}
	return rounds, nil
}

// UpdateParticipantScore updates the score for a participant in a round.
func (db *RoundDBImpl) UpdateParticipantScore(ctx context.Context, roundID roundtypes.ID, participantID string, score int) error {
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
		if p.UserID == roundtypes.UserID(participantID) {
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
func (db *RoundDBImpl) GetParticipantsWithResponses(ctx context.Context, roundID roundtypes.ID, responses ...string) ([]roundtypes.Participant, error) {
	var participants []roundtypes.Participant
	err := db.DB.NewSelect().
		Model(&participants).
		Where("round_id = ? AND response IN (?)", roundID, bun.In(responses)).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with responses: %w", err)
	}

	return participants, nil
}

// GetRoundState retrieves the state of a round.
func (db *RoundDBImpl) GetRoundState(ctx context.Context, roundID roundtypes.ID) (roundtypes.RoundState, error) {
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
func (db *RoundDBImpl) GetParticipants(ctx context.Context, roundID roundtypes.ID) ([]roundtypes.Participant, error) {
	var participants []roundtypes.Participant
	err := db.DB.NewSelect().
		Model(&participants).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	return participants, nil
}

// UpdateDiscordEventID updates the DiscordEventID for an existing round.
func (db *RoundDBImpl) UpdateDiscordEventID(ctx context.Context, roundID roundtypes.ID, discordEventID string) error {
	_, err := db.DB.NewUpdate().
		Model(&roundtypes.Round{}).
		Set("discord_event_id =?", discordEventID).
		Where("id =?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update discord event ID: %w", err)
	}
	return nil
}
