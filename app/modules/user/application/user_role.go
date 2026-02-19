package userservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// UpdateUserRoleInDatabase updates a user's role in the database.
func (s *UserService) UpdateUserRoleInDatabase(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	newRole sharedtypes.UserRoleEnum,
) (results.OperationResult[bool, error], error) {

	// Named transaction function for logic execution
	updateRoleTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		return s.executeUpdateUserRole(ctx, db, guildID, userID, newRole)
	}

	// Wrap with telemetry & transaction helper
	result, err := withTelemetry(s, ctx, "UpdateUserRole", userID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, updateRoleTx)
	})

	if err != nil {
		// Infrastructure failure
		return results.OperationResult[bool, error]{}, fmt.Errorf("UpdateUserRoleInDatabase failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeUpdateUserRole(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	newRole sharedtypes.UserRoleEnum,
) (results.OperationResult[bool, error], error) {

	// 1. Domain Validations
	if userID == "" {
		return results.FailureResult[bool](ErrInvalidDiscordID), nil
	}
	if !newRole.IsValid() {
		return results.FailureResult[bool](ErrInvalidRole), nil
	}

	// 2. Update legacy guild_memberships table.
	err := s.repo.UpdateUserRole(ctx, db, userID, guildID, newRole)
	if err != nil {
		// If user not found, return domain failure (nil error)
		if errors.Is(err, userdb.ErrNoRowsAffected) || errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[bool](ErrUserNotFound), nil
		}
		// Technical error
		return results.OperationResult[bool, error]{}, fmt.Errorf("failed to update user role: %w", err)
	}

	// 3. Also update club_memberships (new identity abstraction) so JWT role claims stay in sync.
	// Best-effort: ErrNoRowsAffected means no club membership exists yet (backfill pending).
	if clubErr := s.repo.UpdateClubMembershipRoleByDiscordIDs(ctx, db, userID, guildID, newRole); clubErr != nil {
		if errors.Is(clubErr, userdb.ErrNoRowsAffected) {
			s.logger.WarnContext(ctx, "No club membership found when updating role; backfill may not have run yet",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
		} else {
			s.logger.WarnContext(ctx, "Failed to update club membership role; JWT role may be stale until next profile sync",
				attr.Error(clubErr),
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
		}
	}

	// 4. Success
	return results.SuccessResult[bool, error](true), nil
}
