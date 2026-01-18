package rounddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new round repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// UpdateImportStatus updates import fields on a round with minimal surface area.
func (r *Impl) UpdateImportStatus(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error {
	update := r.db.NewUpdate().
		Model((*Round)(nil)).
		Set("import_status = ?", status).
		Set("updated_at = now()")

	if importID != "" {
		update = update.Set("import_id = ?", importID)
	}

	if errorMessage != "" {
		update = update.Set("import_error = ?", errorMessage)
	}

	if errorCode != "" {
		update = update.Set("import_error_code = ?", errorCode)
	}

	_, err := update.Where("id = ? AND guild_id = ?", roundID, guildID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update import status: %w", err)
	}
	return nil
}

// CreateRound creates a new round in the database and retrieves the generated ID.
func (r *Impl) CreateRound(ctx context.Context, guildID sharedtypes.GuildID, round *roundtypes.Round) error {
	// Ensure GuildID is set on the round object before insertion
	if round.GuildID == "" {
		round.GuildID = guildID
	}

	// Convert to local model to ensure Bun tags are respected
	localRound := toLocalRound(round)

	_, err := r.db.NewInsert().
		Model(localRound).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}
	return nil
}

// GetRound retrieves a specific round by ID.
func (r *Impl) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	localRound := new(Round)
	err := r.db.NewSelect().
		Model(localRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}
	return toSharedRound(localRound), nil
}

// GetParticipant retrieves a participant's information for a specific round
func (r *Impl) GetParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error) {
	var localRound Round
	err := r.db.NewSelect().
		Model(&localRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Look for the participant in the round's participants
	for _, p := range localRound.Participants {
		if p.UserID == userID {
			return &p, nil
		}
	}

	// Participant not found
	return nil, nil
}

// Helper functions to convert between local and shared models
func toLocalRound(r *roundtypes.Round) *Round {
	local := &Round{
		ID:              r.ID,
		Title:           r.Title,
		EventType:       r.EventType,
		Finalized:       r.Finalized,
		CreatedBy:       r.CreatedBy,
		State:           r.State,
		Participants:    r.Participants,
		EventMessageID:  r.EventMessageID,
		GuildID:         r.GuildID,
		ImportID:        r.ImportID,
		ImportStatus:    ImportStatus(r.ImportStatus),
		ImportType:      ImportType(r.ImportType),
		FileData:        r.FileData,
		FileName:        r.FileName,
		UDiscURL:        r.UDiscURL,
		ImportNotes:     r.ImportNotes,
		ImportError:     r.ImportError,
		ImportErrorCode: r.ImportErrorCode,
		ImportedAt:      r.ImportedAt,
		ImportUserID:    r.ImportUserID,
		ImportChannelID: r.ImportChannelID,
	}

	if r.Description != nil {
		local.Description = *r.Description
	}
	if r.Location != nil {
		local.Location = *r.Location
	}
	if r.StartTime != nil {
		local.StartTime = *r.StartTime
	}

	return local
}

func toSharedRound(r *Round) *roundtypes.Round {
	return &roundtypes.Round{
		ID:              r.ID,
		Title:           r.Title,
		Description:     &r.Description,
		Location:        &r.Location,
		EventType:       r.EventType,
		StartTime:       &r.StartTime,
		Finalized:       r.Finalized,
		CreatedBy:       r.CreatedBy,
		State:           r.State,
		Participants:    r.Participants,
		EventMessageID:  r.EventMessageID,
		GuildID:         r.GuildID,
		ImportID:        r.ImportID,
		ImportStatus:    string(r.ImportStatus),
		ImportType:      string(r.ImportType),
		FileData:        r.FileData,
		FileName:        r.FileName,
		UDiscURL:        r.UDiscURL,
		ImportNotes:     r.ImportNotes,
		ImportError:     r.ImportError,
		ImportErrorCode: r.ImportErrorCode,
		ImportedAt:      r.ImportedAt,
		ImportUserID:    r.ImportUserID,
		ImportChannelID: r.ImportChannelID,
	}
}

// RemoveParticipant removes a participant from a round and returns updated participants
func (r *Impl) RemoveParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
	// First, fetch the round
	var localRound Round
	err := r.db.NewSelect().
		Model(&localRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find and remove the participant
	found := false
	updatedParticipants := make([]roundtypes.Participant, 0, len(localRound.Participants))
	for _, p := range localRound.Participants {
		if p.UserID != userID {
			updatedParticipants = append(updatedParticipants, p)
		} else {
			found = true
		}
	}

	if !found {
		// Participant wasn't in the round - return current participants (graceful handling)
		return localRound.Participants, nil
	}

	// Update the round with the modified participants list
	_, err = r.db.NewUpdate().
		Model(&localRound).
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
		ID:              dbRound.ID,
		Title:           dbRound.Title,
		Description:     &dbRound.Description,
		Location:        &dbRound.Location,
		EventType:       dbRound.EventType,
		StartTime:       &dbRound.StartTime,
		Finalized:       dbRound.Finalized,
		CreatedBy:       dbRound.CreatedBy,
		State:           dbRound.State,
		Participants:    dbRound.Participants,
		EventMessageID:  dbRound.EventMessageID,
		GuildID:         dbRound.GuildID,
		ImportID:        dbRound.ImportID,
		ImportStatus:    string(dbRound.ImportStatus),
		ImportType:      string(dbRound.ImportType),
		FileData:        dbRound.FileData,
		FileName:        dbRound.FileName,
		UDiscURL:        dbRound.UDiscURL,
		ImportNotes:     dbRound.ImportNotes,
		ImportError:     dbRound.ImportError,
		ImportErrorCode: dbRound.ImportErrorCode,
		ImportedAt:      dbRound.ImportedAt,
		ImportUserID:    dbRound.ImportUserID,
		ImportChannelID: dbRound.ImportChannelID,
	}
}

// UpdateRound updates specific fields of an existing round in the database and returns the updated round.
func (r *Impl) UpdateRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error) {
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

	// Import/scorecard fields
	if round.ImportID != "" {
		dbRound.ImportID = round.ImportID
	}
	if round.ImportStatus != "" {
		dbRound.ImportStatus = ImportStatus(round.ImportStatus)
	}
	if round.ImportType != "" {
		dbRound.ImportType = ImportType(round.ImportType)
	}
	if len(round.FileData) > 0 {
		dbRound.FileData = round.FileData
	}
	if round.FileName != "" {
		dbRound.FileName = round.FileName
	}
	if round.UDiscURL != "" {
		dbRound.UDiscURL = round.UDiscURL
	}
	if round.ImportNotes != "" {
		dbRound.ImportNotes = round.ImportNotes
	}
	if round.ImportError != "" {
		dbRound.ImportError = round.ImportError
	}
	if round.ImportErrorCode != "" {
		dbRound.ImportErrorCode = round.ImportErrorCode
	}
	if round.ImportedAt != nil {
		dbRound.ImportedAt = round.ImportedAt
	}
	if round.ImportUserID != "" {
		dbRound.ImportUserID = round.ImportUserID
	}
	if round.ImportChannelID != "" {
		dbRound.ImportChannelID = round.ImportChannelID
	}

	var updatedDbRound Round

	// Now use the database model for both Model() and scan target
	_, err := r.db.NewUpdate().
		Model(&dbRound).
		OmitZero(). // This will ignore zero values
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Returning("*").
		Exec(ctx, &updatedDbRound)
	if err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	// Convert back to domain model and return COMPLETE round
	return convertToDomainRound(updatedDbRound), nil
}

// DeleteRound "soft deletes" a round by setting its state to DELETED.
func (r *Impl) DeleteRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) error {
	// Validate the round ID isn't nil/zero
	if roundID == sharedtypes.RoundID(uuid.Nil) {
		return fmt.Errorf("cannot delete round: nil UUID provided")
	}

	// Check if the round exists first
	exists, err := r.db.NewSelect().
		Model(&Round{}).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if round exists: %w", err)
	}

	if !exists {
		return ErrNotFound
	}

	// Update the round state
	res, err := r.db.NewUpdate().
		Model(&Round{}).
		Set("state = ?", roundtypes.RoundState(roundtypes.RoundStateDeleted)).
		Set("updated_at = ?", time.Now()).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

// UpdateParticipant updates a participant's response or tag number in a round and returns updated domain participants.
func (r *Impl) UpdateParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error) {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var dbRound Round
	err = tx.NewSelect().
		Model(&dbRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch round: %w", err)
	}

	// Initialize participants if null
	if dbRound.Participants == nil {
		dbRound.Participants = []roundtypes.Participant{}
	}

	// Modify participants
	updated := false
	for i := range dbRound.Participants { // Iterate by index to modify in place
		p := &dbRound.Participants[i] // Use pointer to modify the slice element

		if p.UserID == participant.UserID {
			if participant.Response != "" {
				p.Response = participant.Response
			}

			// Always update TagNumber, whether it's setting a new value or clearing an existing one.
			p.TagNumber = participant.TagNumber

			if participant.Score != nil {
				p.Score = participant.Score
			}

			updated = true
			break // Found the participant, exit the loop
		}
	}

	if !updated {
		dbRound.Participants = append(dbRound.Participants, participant)
	}

	// Update the record
	_, err = tx.NewUpdate().
		Model(&dbRound).
		Set("participants = ?", dbRound.Participants). // Assuming participants are stored as JSONB or similar
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update round: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return dbRound.Participants, nil
}

// UpdateRoundState updates the state of a round.
func (r *Impl) UpdateRoundState(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error {
	_, err := r.db.NewUpdate().
		Model(&Round{}).
		Set("state = ?", state).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	return nil
}

// GetUpcomingRounds retrieves rounds that are upcoming
func (r *Impl) GetUpcomingRounds(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	var localRounds []*Round
	err := r.db.NewSelect().
		Model(&localRounds).
		Where("state = ? AND guild_id = ?", roundtypes.RoundStateUpcoming, guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}

	rounds := make([]*roundtypes.Round, len(localRounds))
	for i, r := range localRounds {
		rounds[i] = toSharedRound(r)
	}
	return rounds, nil
}

// GetUpcomingRoundsByParticipant retrieves upcoming rounds that contain a specific participant
func (r *Impl) GetUpcomingRoundsByParticipant(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) ([]*roundtypes.Round, error) {
	var localRounds []*Round
	err := r.db.NewSelect().
		Model(&localRounds).
		Where("state = ? AND guild_id = ?", roundtypes.RoundStateUpcoming, guildID).
		Where("participants @> ?", fmt.Sprintf(`[{"user_id": "%s"}]`, userID)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming rounds by participant: %w", err)
	}

	rounds := make([]*roundtypes.Round, len(localRounds))
	for i, r := range localRounds {
		rounds[i] = toSharedRound(r)
	}
	return rounds, nil
}

// UpdateParticipantScore updates the score for a participant in a round.
func (r *Impl) UpdateParticipantScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participantID sharedtypes.DiscordID, score sharedtypes.Score) error {
	var localRound Round
	err := r.db.NewSelect().
		Model(&localRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch round: %w", err)
	}

	// Find the participant and update their score
	found := false
	for i, p := range localRound.Participants {
		if p.UserID == participantID {
			localRound.Participants[i].Score = &score
			found = true
			break
		}
	}
	if !found {
		return ErrParticipantNotFound
	}

	// Update the round in the database
	_, err = r.db.NewUpdate().
		Model(&localRound).
		Set("participants = ?", localRound.Participants).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update participant score: %w", err)
	}

	return nil
}

// GetParticipantsWithResponses retrieves participants with the specified responses from a round.
func (r *Impl) GetParticipantsWithResponses(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, responses ...string) ([]roundtypes.Participant, error) {
	var localRound Round
	err := r.db.NewSelect().
		Model(&localRound).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	var participants []roundtypes.Participant
	for _, p := range localRound.Participants {
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
func (r *Impl) GetRoundState(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (roundtypes.RoundState, error) {
	var round roundtypes.Round
	err := r.db.NewSelect().
		Model(&round).
		Column("state").
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get round state: %w", err)
	}
	return round.State, nil
}

// GetParticipants retrieves all participants from a round.
func (r *Impl) GetParticipants(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
	var round roundtypes.Round
	err := r.db.NewSelect().
		Model(&round).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch round: %w", err)
	}

	return round.Participants, nil
}

// UpdateEventMessageID updates the EventMessageID(messageID) for an existing round.
func (r *Impl) UpdateEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error) {
	var dbRound Round

	// Build update with conditional guild filter
	upd := r.db.NewUpdate().
		Model(&dbRound).
		Set("event_message_id = ?", eventMessageID)
	if string(guildID) == "" {
		// No guild provided: update by round ID only (test helper may omit guild)
		upd = upd.Where("id = ?", roundID)
	} else {
		upd = upd.Where("id = ? AND guild_id = ?", roundID, guildID)
	}

	_, err := upd.Returning("*").Exec(ctx, &dbRound)
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
		GuildID:        dbRound.GuildID,
	}

	return round, nil
}

// GetEventMessageID retrieves the EventMessageID for a given round.
func (r *Impl) GetEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (string, error) {
	var round Round
	err := r.db.NewSelect().
		Model(&round).
		Column("event_message_id").
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get event message ID: %w", err)
	}

	return round.EventMessageID, nil // Return the string directly
}

// UpdateRoundsAndParticipants updates multiple rounds and participants in a single transaction.
func (r *Impl) UpdateRoundsAndParticipants(ctx context.Context, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		for _, update := range updates {
			// Only update the participants column, not the entire round
			_, err := tx.NewUpdate().
				Model((*roundtypes.Round)(nil)).
				Set("participants = ?", update.Participants).
				Where("id = ? AND guild_id = ?", update.RoundID, guildID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update participants for round %s: %w", update.RoundID, err)
			}
		}
		return nil
	})
}
