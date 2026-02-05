package userservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// UpdateUDiscIdentity sets UDisc username/name for a user globally.
func (s *UserService) UpdateUDiscIdentity(
	ctx context.Context,
	userID sharedtypes.DiscordID,
	username *string,
	name *string,
) (results.OperationResult[bool, error], error) {

	updateTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		return s.executeUpdateUDiscIdentity(ctx, db, userID, username, name)
	}

	result, err := withTelemetry(s, ctx, "UpdateUDiscIdentity", userID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, updateTx)
	})

	if err != nil {
		return results.FailureResult[bool](err), fmt.Errorf("UpdateUDiscIdentity failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeUpdateUDiscIdentity(
	ctx context.Context,
	db bun.IDB,
	userID sharedtypes.DiscordID,
	username *string,
	name *string,
) (results.OperationResult[bool, error], error) {

	if userID == "" {
		return results.FailureResult[bool](ErrInvalidDiscordID), nil
	}

	updates := &userdb.UserUpdateFields{
		UDiscUsername: normalizeStringPointer(username),
		UDiscName:     normalizeStringPointer(name),
	}

	if updates.IsEmpty() {
		return results.SuccessResult[bool, error](true), nil
	}

	if err := s.repo.UpdateGlobalUser(ctx, db, userID, updates); err != nil {
		if errors.Is(err, userdb.ErrNoRowsAffected) {
			return results.FailureResult[bool](userdb.ErrNotFound), nil
		}
		return results.OperationResult[bool, error]{}, fmt.Errorf("failed to update global user: %w", err)
	}

	return results.SuccessResult[bool, error](true), nil
}

// UpdateClubMembership updates a user's club membership.
func (s *UserService) UpdateClubMembership(
	ctx context.Context,
	clubUUID uuid.UUID,
	userUUID uuid.UUID,
	displayName *string,
	avatarURL *string,
	role *sharedtypes.UserRoleEnum,
) (results.OperationResult[bool, error], error) {
	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		// Get existing to determine role if not provided
		existing, err := s.repo.GetClubMembership(ctx, db, userUUID, clubUUID)
		if err != nil && !errors.Is(err, userdb.ErrNotFound) {
			return results.OperationResult[bool, error]{}, err
		}

		membership := &userdb.ClubMembership{
			UserUUID:    userUUID,
			ClubUUID:    clubUUID,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
		}

		if role != nil {
			membership.Role = *role
		} else if existing != nil {
			membership.Role = existing.Role
		} else {
			membership.Role = sharedtypes.UserRoleUser
		}

		if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
			return results.OperationResult[bool, error]{}, err
		}
		return results.SuccessResult[bool, error](true), nil
	}

	return withTelemetry(s, ctx, "UpdateClubMembership", "", func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, op)
	})
}

// GetUUIDByDiscordID resolves a Discord ID to internal UUID.
func (s *UserService) GetUUIDByDiscordID(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	return s.repo.GetUUIDByDiscordID(ctx, s.db, discordID)
}

// GetClubUUIDByDiscordGuildID resolves a Discord guild ID to internal club UUID.
func (s *UserService) GetClubUUIDByDiscordGuildID(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	return s.repo.GetClubUUIDByDiscordGuildID(ctx, s.db, guildID)
}

func normalizeStringPointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}

func pointer[T any](v T) *T {
	return &v
}
