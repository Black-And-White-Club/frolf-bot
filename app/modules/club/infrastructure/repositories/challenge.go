package clubdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func (r *Impl) GetChallengeByUUID(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*ClubChallenge, error) {
	db = r.resolveDB(db)
	challenge := new(ClubChallenge)
	err := db.NewSelect().
		Model(challenge).
		Where("uuid = ?", challengeUUID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get challenge by uuid: %w", err)
	}
	return challenge, nil
}

func (r *Impl) CreateChallenge(ctx context.Context, db bun.IDB, challenge *ClubChallenge) error {
	db = r.resolveDB(db)
	now := time.Now().UTC()
	challenge.CreatedAt = now
	challenge.UpdatedAt = now
	if challenge.OpenedAt.IsZero() {
		challenge.OpenedAt = now
	}
	_, err := db.NewInsert().Model(challenge).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create challenge: %w", err)
	}
	return nil
}

func (r *Impl) UpdateChallenge(ctx context.Context, db bun.IDB, challenge *ClubChallenge) error {
	db = r.resolveDB(db)
	challenge.UpdatedAt = time.Now().UTC()
	result, err := db.NewUpdate().
		Model(challenge).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get challenge update rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Impl) ListChallenges(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*ClubChallenge, error) {
	db = r.resolveDB(db)
	var challenges []*ClubChallenge
	query := db.NewSelect().
		Model(&challenges).
		Where("club_uuid = ?", clubUUID).
		OrderExpr("opened_at DESC")
	if len(statuses) > 0 {
		query = query.Where("status IN (?)", bun.In(statuses))
	}
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to list challenges: %w", err)
	}
	return challenges, nil
}

func (r *Impl) GetOpenOutgoingChallenge(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*ClubChallenge, error) {
	db = r.resolveDB(db)
	challenge := new(ClubChallenge)
	err := db.NewSelect().
		Model(challenge).
		Where("club_uuid = ?", clubUUID).
		Where("challenger_user_uuid = ?", challengerUserUUID).
		Where("status = ?", clubtypes.ChallengeStatusOpen).
		OrderExpr("opened_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get open outgoing challenge: %w", err)
	}
	return challenge, nil
}

func (r *Impl) GetAcceptedChallengeForUser(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*ClubChallenge, error) {
	db = r.resolveDB(db)
	challenge := new(ClubChallenge)
	err := db.NewSelect().
		Model(challenge).
		Where("club_uuid = ?", clubUUID).
		Where("(challenger_user_uuid = ? OR defender_user_uuid = ?)", userUUID, userUUID).
		Where("status = ?", clubtypes.ChallengeStatusAccepted).
		OrderExpr("opened_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get accepted challenge for user: %w", err)
	}
	return challenge, nil
}

func (r *Impl) GetActiveChallengeByPair(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*ClubChallenge, error) {
	db = r.resolveDB(db)
	challenge := new(ClubChallenge)
	err := db.NewSelect().
		Model(challenge).
		Where("club_uuid = ?", clubUUID).
		Where("status IN (?)", bun.In([]clubtypes.ChallengeStatus{clubtypes.ChallengeStatusOpen, clubtypes.ChallengeStatusAccepted})).
		Where("((challenger_user_uuid = ? AND defender_user_uuid = ?) OR (challenger_user_uuid = ? AND defender_user_uuid = ?))", userA, userB, userB, userA).
		OrderExpr("opened_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active challenge by pair: %w", err)
	}
	return challenge, nil
}

func (r *Impl) ListActiveChallengesByUsers(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs []uuid.UUID) ([]*ClubChallenge, error) {
	db = r.resolveDB(db)
	if len(userUUIDs) == 0 {
		return nil, nil
	}
	var challenges []*ClubChallenge
	err := db.NewSelect().
		Model(&challenges).
		Where("club_uuid = ?", clubUUID).
		Where("status IN (?)", bun.In([]clubtypes.ChallengeStatus{clubtypes.ChallengeStatusOpen, clubtypes.ChallengeStatusAccepted})).
		Where("(challenger_user_uuid IN (?) OR defender_user_uuid IN (?))", bun.In(userUUIDs), bun.In(userUUIDs)).
		OrderExpr("opened_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active challenges by users: %w", err)
	}
	return challenges, nil
}

func (r *Impl) BindChallengeMessage(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, guildID, channelID, messageID string) error {
	db = r.resolveDB(db)
	result, err := db.NewUpdate().
		Model((*ClubChallenge)(nil)).
		Set("discord_guild_id = ?", guildID).
		Set("discord_channel_id = ?", channelID).
		Set("discord_message_id = ?", messageID).
		Set("updated_at = ?", time.Now().UTC()).
		Where("uuid = ?", challengeUUID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to bind challenge message: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get challenge message bind rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Impl) CreateChallengeRoundLink(ctx context.Context, db bun.IDB, link *ClubChallengeRoundLink) error {
	db = r.resolveDB(db)
	if link.LinkedAt.IsZero() {
		link.LinkedAt = time.Now().UTC()
	}
	_, err := db.NewInsert().Model(link).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create challenge round link: %w", err)
	}
	return nil
}

func (r *Impl) GetActiveChallengeRoundLink(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*ClubChallengeRoundLink, error) {
	db = r.resolveDB(db)
	link := new(ClubChallengeRoundLink)
	err := db.NewSelect().
		Model(link).
		Where("challenge_uuid = ?", challengeUUID).
		Where("unlinked_at IS NULL").
		OrderExpr("linked_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get active challenge round link: %w", err)
	}
	return link, nil
}

func (r *Impl) GetChallengeByActiveRound(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*ClubChallenge, error) {
	db = r.resolveDB(db)
	challenge := new(ClubChallenge)
	err := db.NewSelect().
		Model(challenge).
		Join("JOIN club_challenge_round_links ccrl ON ccrl.challenge_uuid = cc.uuid").
		Where("ccrl.round_id = ?", roundID).
		Where("ccrl.unlinked_at IS NULL").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get challenge by active round: %w", err)
	}
	return challenge, nil
}

func (r *Impl) UnlinkActiveChallengeRound(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error {
	db = r.resolveDB(db)
	query := db.NewUpdate().
		Model((*ClubChallengeRoundLink)(nil)).
		Set("unlinked_at = ?", unlinkedAt).
		Where("challenge_uuid = ?", challengeUUID).
		Where("unlinked_at IS NULL")
	if actorUserUUID != nil {
		query = query.Set("unlinked_by_user_uuid = ?", *actorUserUUID)
	}
	result, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlink active challenge round: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get challenge round unlink rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
