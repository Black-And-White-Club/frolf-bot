package userservice

import (
	"context"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// MapDBUserWithMembershipToDomain converts the DB join result to our clean application struct.
func MapDBUserWithMembershipToDomain(dbUser *userdb.UserWithMembership) *UserWithMembership {
	if dbUser == nil {
		return nil
	}
	return &UserWithMembership{
		UserData: usertypes.UserData{
			UserID: dbUser.User.UserID,
			Role:   dbUser.Role,
		},
		IsMember: true, // If the repo found a UserWithMembership, they have a record
	}
}

// GetUser retrieves user data and maps it to a clean domain type.
func (s *UserService) GetUser(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (UserWithMembershipResult, error) {

	getUserOp := func(ctx context.Context, db bun.IDB) (UserWithMembershipResult, error) {
		return s.executeGetUser(ctx, db, guildID, userID)
	}

	result, err := withTelemetry(s, ctx, "GetUser", userID, func(ctx context.Context) (UserWithMembershipResult, error) {
		return getUserOp(ctx, s.db)
	})

	if err != nil {
		return UserWithMembershipResult{}, fmt.Errorf("GetUser failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeGetUser(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (UserWithMembershipResult, error) {

	if userID == "" {
		return results.FailureResult[*UserWithMembership](ErrInvalidDiscordID), nil
	}

	user, err := s.repo.GetUserByUserID(ctx, db, userID, guildID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[*UserWithMembership](userdb.ErrNotFound), nil
		}
		return UserWithMembershipResult{}, fmt.Errorf("failed to get user: %w", err)
	}

	// MAP TO DOMAIN HERE
	return results.SuccessResult[*UserWithMembership, error](MapDBUserWithMembershipToDomain(user)), nil
}

// GetUserRole remains simple because UserRoleEnum is already a shared type
func (s *UserService) GetUserRole(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {

	getUserRoleOp := func(ctx context.Context, db bun.IDB) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
		return s.executeGetUserRole(ctx, db, guildID, userID)
	}

	result, err := withTelemetry(s, ctx, "GetUserRole", userID, func(ctx context.Context) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
		return getUserRoleOp(ctx, s.db)
	})

	if err != nil {
		return results.OperationResult[sharedtypes.UserRoleEnum, error]{}, fmt.Errorf("GetUserRole failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeGetUserRole(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {

	role, err := s.repo.GetUserRole(ctx, db, userID, guildID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[sharedtypes.UserRoleEnum](userdb.ErrNotFound), nil
		}
		return results.OperationResult[sharedtypes.UserRoleEnum, error]{}, fmt.Errorf("failed to get user role: %w", err)
	}

	if !role.IsValid() {
		return results.FailureResult[sharedtypes.UserRoleEnum](fmt.Errorf("invalid role: %s", role)), nil
	}

	return results.SuccessResult[sharedtypes.UserRoleEnum, error](role), nil
}
