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

// MapDBUserToUserData converts internal DB model to the shared domain model.
func MapDBUserToUserData(dbUser *userdb.User) *usertypes.UserData {
	if dbUser == nil {
		return nil
	}
	return &usertypes.UserData{
		UserID: dbUser.UserID,
		Role:   sharedtypes.UserRoleUser,
	}
}

// CreateUser is the public entry point with telemetry and transaction handling.
func (s *UserService) CreateUser(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	tag *sharedtypes.TagNumber,
	udiscUsername *string,
	udiscName *string,
) (UserResult, error) {

	// Named transaction function
	createUserTx := func(ctx context.Context, db bun.IDB) (UserResult, error) {
		return s.executeCreateUser(ctx, db, guildID, userID, tag, udiscUsername, udiscName)
	}

	// Wrap with telemetry & transaction helper
	result, err := withTelemetry(s, ctx, "CreateUser", userID, func(ctx context.Context) (UserResult, error) {
		return runInTx(s, ctx, createUserTx)
	})

	if err != nil {
		return UserResult{}, fmt.Errorf("CreateUser failed: %w", err)
	}

	return result, nil
}

// executeCreateUser contains the actual business logic.
func (s *UserService) executeCreateUser(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	tag *sharedtypes.TagNumber,
	udiscUsername *string,
	udiscName *string,
) (UserResult, error) {

	// 1. Domain Validations
	if userID == "" {
		return results.FailureResult[*usertypes.UserData](ErrInvalidDiscordID), nil
	}
	if tag != nil && *tag < 0 {
		return results.FailureResult[*usertypes.UserData](ErrNegativeTagNumber), nil
	}

	// 2. Check if user exists globally
	user, err := s.repo.GetUserGlobal(ctx, db, userID)
	if err != nil && !errors.Is(err, userdb.ErrNotFound) {
		return UserResult{}, fmt.Errorf("failed to get global user: %w", err)
	}

	if user != nil {
		membership, err := s.repo.GetGuildMembership(ctx, db, userID, guildID)
		if err != nil && !errors.Is(err, userdb.ErrNotFound) {
			return UserResult{}, fmt.Errorf("failed to check guild membership: %w", err)
		}

		if membership != nil {
			return results.FailureResult[*usertypes.UserData](ErrUserAlreadyExists), nil
		}

		if err := s.repo.CreateGuildMembership(ctx, db, &userdb.GuildMembership{
			UserID:  userID,
			GuildID: guildID,
			Role:    sharedtypes.UserRoleUser,
		}); err != nil {
			return UserResult{}, fmt.Errorf("failed to create guild membership: %w", err)
		}

		return results.SuccessResult[*usertypes.UserData, error](MapDBUserToUserData(user)), nil
	}

	// 3. New user flow
	newUser := &userdb.User{
		UserID:        userID,
		UDiscUsername: normalizeStringPointer(udiscUsername),
		UDiscName:     normalizeStringPointer(udiscName),
	}

	if err := s.repo.SaveGlobalUser(ctx, db, newUser); err != nil {
		return UserResult{}, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.repo.CreateGuildMembership(ctx, db, &userdb.GuildMembership{
		UserID:  userID,
		GuildID: guildID,
		Role:    sharedtypes.UserRoleUser,
	}); err != nil {
		return UserResult{}, fmt.Errorf("failed to create guild membership: %w", err)
	}

	return results.SuccessResult[*usertypes.UserData, error](MapDBUserToUserData(newUser)), nil
}
