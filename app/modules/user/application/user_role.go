package userservice

import (
	"context"
	"errors"
	"fmt"

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
		return results.FailureResult[bool](errors.New("invalid role")), nil
	}

	// 2. Repository Call
	err := s.repo.UpdateUserRole(ctx, db, userID, guildID, newRole)
	if err != nil {
		// If user not found, return domain failure (nil error)
		if errors.Is(err, userdb.ErrNoRowsAffected) || errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[bool](userdb.ErrNotFound), nil
		}
		// Technical error
		return results.OperationResult[bool, error]{}, fmt.Errorf("failed to update user role: %w", err)
	}

	// 3. Success
	return results.SuccessResult[bool, error](true), nil
}
