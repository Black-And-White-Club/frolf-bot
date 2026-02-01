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
			ID:     dbUser.User.ID,
			UserID: dbUser.User.UserID,
			Role:   dbUser.Role,
		},
		DisplayName:   dbUser.User.GetDisplayName(),
		AvatarHash:    dbUser.User.AvatarHash,
		UDiscUsername: dbUser.User.UDiscUsername,
		UDiscName:     dbUser.User.UDiscName,
		IsMember:      true,
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

	return result, err

}

func (s *UserService) executeGetUser(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (UserWithMembershipResult, error) {

	if userID == "" {
		return results.FailureResult[*UserWithMembership](ErrInvalidDiscordID), ErrInvalidDiscordID
	}

	user, err := s.repo.GetUserByUserID(ctx, db, userID, guildID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[*UserWithMembership](ErrUserNotFound), nil
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
) (UserRoleResult, error) {

	getUserRoleOp := func(ctx context.Context, db bun.IDB) (UserRoleResult, error) {
		return s.executeGetUserRole(ctx, db, guildID, userID)
	}

	result, err := withTelemetry(s, ctx, "GetUserRole", userID, func(ctx context.Context) (UserRoleResult, error) {
		return getUserRoleOp(ctx, s.db)
	})

	if err != nil {
		return UserRoleResult{}, fmt.Errorf("GetUserRole failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeGetUserRole(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (UserRoleResult, error) {

	role, err := s.repo.GetUserRole(ctx, db, userID, guildID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return results.FailureResult[sharedtypes.UserRoleEnum](ErrUserNotFound), nil
		}
		return UserRoleResult{}, fmt.Errorf("failed to get user role: %w", err)
	}

	if !role.IsValid() {
		return results.FailureResult[sharedtypes.UserRoleEnum](fmt.Errorf("invalid role: %s", role)), nil
	}

	return results.SuccessResult[sharedtypes.UserRoleEnum, error](role), nil
}

// FindByUDiscUsername searches for a user by their UDisc username.
func (s *UserService) FindByUDiscUsername(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	username string,
) (UserWithMembershipResult, error) {
	op := func(ctx context.Context, db bun.IDB) (UserWithMembershipResult, error) {
		user, err := s.repo.FindByUDiscUsername(ctx, db, guildID, username)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return results.FailureResult[*UserWithMembership](ErrUserNotFound), nil
			}
			return UserWithMembershipResult{}, fmt.Errorf("failed to find user by udisc username: %w", err)
		}
		return results.SuccessResult[*UserWithMembership, error](MapDBUserWithMembershipToDomain(user)), nil
	}

	result, err := withTelemetry(s, ctx, "FindByUDiscUsername", "", func(ctx context.Context) (UserWithMembershipResult, error) {
		return op(ctx, s.db)
	})

	if err != nil {
		return UserWithMembershipResult{}, fmt.Errorf("FindByUDiscUsername failed: %w", err)
	}

	return result, nil
}

// FindByUDiscName searches for a user by their UDisc name.
func (s *UserService) FindByUDiscName(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	name string,
) (UserWithMembershipResult, error) {
	op := func(ctx context.Context, db bun.IDB) (UserWithMembershipResult, error) {
		user, err := s.repo.FindByUDiscName(ctx, db, guildID, name)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return results.FailureResult[*UserWithMembership](ErrUserNotFound), nil
			}
			return UserWithMembershipResult{}, fmt.Errorf("failed to find user by udisc name: %w", err)
		}
		return results.SuccessResult[*UserWithMembership, error](MapDBUserWithMembershipToDomain(user)), nil
	}

	result, err := withTelemetry(s, ctx, "FindByUDiscName", "", func(ctx context.Context) (UserWithMembershipResult, error) {
		return op(ctx, s.db)
	})

	if err != nil {
		return UserWithMembershipResult{}, fmt.Errorf("FindByUDiscName failed: %w", err)
	}

	return result, nil
}
