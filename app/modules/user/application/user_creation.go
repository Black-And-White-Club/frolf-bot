package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// MapDBUserToUserData converts internal DB model to the application response model.
func MapDBUserToUserData(dbUser *userdb.User, tag *sharedtypes.TagNumber, returning bool) *CreateUserResponse {
	if dbUser == nil {
		return nil
	}
	return &CreateUserResponse{
		UserData: usertypes.UserData{
			ID:     dbUser.ID,
			UserID: dbUser.GetUserID(),
			Role:   sharedtypes.UserRoleUser,
		},
		TagNumber:       tag,
		IsReturningUser: returning,
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
	if ctx == nil {
		return results.FailureResult[*CreateUserResponse](errors.New("context cannot be nil")), errors.New("context cannot be nil")
	}

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
		return results.FailureResult[*CreateUserResponse](ErrInvalidDiscordID), nil
	}
	if tag != nil && *tag < 0 {
		return results.FailureResult[*CreateUserResponse](ErrNegativeTagNumber), nil
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

		// Update global user data if provided (e.g. UDisc Name/Username)
		updates := &userdb.UserUpdateFields{
			UDiscUsername: normalizeStringPointer(udiscUsername),
			UDiscName:     normalizeStringPointer(udiscName),
		}
		if !updates.IsEmpty() {
			if err := s.repo.UpdateGlobalUser(ctx, db, userID, updates); err != nil {
				s.logger.WarnContext(ctx, "Failed to update global user data during signup (existing user)",
					attr.String("user_id", string(userID)),
					attr.Error(err))
			}
		}

		if membership != nil {
			// User exists and is in the guild.
			// Ensure club membership exists (backfill if needed due to previous failures)
			clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
			if err != nil {
				s.logger.WarnContext(ctx, "Failed to resolve Club UUID for membership check (returning user)",
					attr.String("guild_id", string(guildID)),
					attr.Error(err))
			} else {
				// Upsert club membership to ensure it exists
				extID := string(userID)
				// We don't overwrite role or display name for existing members to preserve their state
				// but we ensure the record exists.
				// Use a dedicated Upsert that respects existing values if needed,
				// or just rely on the existing UpsertClubMembership logic which is an UPSERT.
				emptyName := ""
				cmParams := &userdb.ClubMembership{
					ClubUUID:    clubUUID,
					UserUUID:    user.UUID,
					Role:        "player",
					JoinedAt:    membership.JoinedAt, // Use guild join time
					ExternalID:  &extID,
					DisplayName: &emptyName,
				}

				// Fetch existing first to preserve role?
				// UpsertClubMembership in repo: "Set role = EXCLUDED.role".
				// So if we pass "player", we might overwrite "admin".
				// We should fetch existing club membership first.
				existingCM, err := s.repo.GetClubMembership(ctx, db, user.UUID, clubUUID)
				if err == nil && existingCM != nil {
					cmParams.Role = existingCM.Role
				}

				if err := s.repo.UpsertClubMembership(ctx, db, cmParams); err != nil {
					s.logger.ErrorContext(ctx, "Failed to ensure club membership (returning user)",
						attr.Error(err))
				}
			}

			return results.SuccessResult[*CreateUserResponse, error](MapDBUserToUserData(user, tag, true)), nil
		}

		if err := s.repo.CreateGuildMembership(ctx, db, &userdb.GuildMembership{
			UserID:  userID,
			GuildID: guildID,
			Role:    sharedtypes.UserRoleUser,
		}); err != nil {
			return UserResult{}, fmt.Errorf("failed to create guild membership: %w", err)
		}

		// (No changes needed below here as the flow continues for new guild members)

		// New Club Membership for existing user
		clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to resolve Club UUID for membership creation (existing user)",
				attr.String("guild_id", string(guildID)),
				attr.Error(err))
		} else {
			extID := string(userID)
			emptyName := ""
			membership := &userdb.ClubMembership{
				ClubUUID:    clubUUID,
				UserUUID:    user.UUID,
				Role:        "player",
				JoinedAt:    time.Now(),
				ExternalID:  &extID,
				DisplayName: &emptyName,
			}
			if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
				s.logger.ErrorContext(ctx, "Failed to create club membership (existing user)",
					attr.String("user_uuid", user.UUID.String()),
					attr.String("club_uuid", clubUUID.String()),
					attr.Error(err))
			}
		}

		return results.SuccessResult[*CreateUserResponse, error](MapDBUserToUserData(user, tag, true)), nil
	}

	// 3. New user flow
	newUser := &userdb.User{
		UserID:        &userID,
		UDiscUsername: normalizeStringPointer(udiscUsername),
		UDiscName:     normalizeStringPointer(udiscName),
	}

	if err := s.repo.SaveGlobalUser(ctx, db, newUser); err != nil {
		return UserResult{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Legacy Guild Membership
	if err := s.repo.CreateGuildMembership(ctx, db, &userdb.GuildMembership{
		UserID:  userID,
		GuildID: guildID,
		Role:    sharedtypes.UserRoleUser,
	}); err != nil {
		return UserResult{}, fmt.Errorf("failed to create guild membership: %w", err)
	}

	// New Club Membership
	// Try to resolve Club UUID from Guild ID. If it fails (e.g. race condition), log and continue.
	// The system should eventually converge or require a manual sync/setup.
	clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to resolve Club UUID for membership creation",
			attr.String("guild_id", string(guildID)),
			attr.Error(err))
	} else {
		extID := string(userID)
		emptyName := ""
		membership := &userdb.ClubMembership{
			ClubUUID:    clubUUID,
			UserUUID:    newUser.UUID,
			Role:        "player", // Default role
			JoinedAt:    newUser.CreatedAt,
			ExternalID:  &extID,
			DisplayName: &emptyName, // Will be backfilled/updated later
		}
		if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
			s.logger.ErrorContext(ctx, "Failed to create club membership",
				attr.String("user_uuid", newUser.UUID.String()),
				attr.String("club_uuid", clubUUID.String()),
				attr.Error(err))
			// Don't fail the whole request, as legacy auth still works
		}
	}

	return results.SuccessResult[*CreateUserResponse, error](MapDBUserToUserData(newUser, tag, false)), nil
}
