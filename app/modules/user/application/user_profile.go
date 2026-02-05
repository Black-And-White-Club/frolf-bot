package userservice

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// UpdateUserProfile updates user's display name and avatar hash.
func (s *UserService) UpdateUserProfile(
	ctx context.Context,
	userID sharedtypes.DiscordID,
	guildID sharedtypes.GuildID,
	displayName string,
	avatarHash string,
) error {

	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		// Update global profile
		if err := s.repo.UpdateProfile(ctx, db, userID, displayName, avatarHash); err != nil {
			return results.OperationResult[bool, error]{}, err
		}

		// Update club membership if guildID is provided
		if guildID != "" {
			userUUID, err := s.repo.GetUUIDByDiscordID(ctx, db, userID)
			if err != nil {
				return results.OperationResult[bool, error]{}, err
			}
			clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
			if err != nil {
				return results.OperationResult[bool, error]{}, err
			}

			// Upsert club membership with Discord as source
			avatarURL := ""
			if avatarHash != "" {
				avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", string(userID), avatarHash)
			}

			now := time.Now()
			membership := &userdb.ClubMembership{
				UserUUID:    userUUID,
				ClubUUID:    clubUUID,
				DisplayName: &displayName,
				AvatarURL:   &avatarURL,
				Source:      "discord",
				ExternalID:  pointer(string(userID)),
				SyncedAt:    &now, // Mark when this sync occurred
			}

			// Get existing to preserve role
			existing, err := s.repo.GetClubMembership(ctx, db, userUUID, clubUUID)
			if err == nil && existing != nil {
				membership.Role = existing.Role
			} else {
				membership.Role = sharedtypes.UserRoleUser
			}

			if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
				return results.OperationResult[bool, error]{}, err
			}
		}

		return results.SuccessResult[bool, error](true), nil
	}

	_, err := withTelemetry(s, ctx, "UpdateUserProfile", userID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, op)
	})

	return err
}

// LookupProfiles retrieves user profiles for display.
func (s *UserService) LookupProfiles(
	ctx context.Context,
	userIDs []sharedtypes.DiscordID,
) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
	userIdForTelemetry := sharedtypes.DiscordID("bulk")
	if len(userIDs) > 0 {
		userIdForTelemetry = userIDs[0] + "..."
	}

	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
		return s.executeLookupProfiles(ctx, db, userIDs)
	}

	return withTelemetry(s, ctx, "LookupProfiles", userIdForTelemetry, func(ctx context.Context) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
		return runInTx(s, ctx, op)
	})
}

func (s *UserService) executeLookupProfiles(
	ctx context.Context,
	db bun.IDB,
	userIDs []sharedtypes.DiscordID,
) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
	if len(userIDs) == 0 {
		return results.SuccessResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error](make(map[sharedtypes.DiscordID]*usertypes.UserProfile)), nil
	}

	users, err := s.repo.GetByUserIDs(ctx, db, userIDs)
	if err != nil {
		return results.FailureResult[map[sharedtypes.DiscordID]*usertypes.UserProfile](err), nil
	}

	result := make(map[sharedtypes.DiscordID]*usertypes.UserProfile, len(users))
	for _, u := range users {
		result[u.GetUserID()] = &usertypes.UserProfile{
			UserID:        u.GetUserID(),
			DisplayName:   u.GetDisplayName(),
			AvatarURL:     u.AvatarURL(64), // 64px for list views
			UDiscUsername: u.UDiscUsername,
			UDiscName:     u.UDiscName,
		}
	}

	// For users not in DB, generate default profiles
	for _, id := range userIDs {
		if _, exists := result[id]; !exists {
			result[id] = s.generateDefaultProfile(id)
		}
	}

	return results.SuccessResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error](result), nil
}

func (s *UserService) generateDefaultProfile(userID sharedtypes.DiscordID) *usertypes.UserProfile {
	idStr := string(userID)
	displayName := "User"
	if len(idStr) > 6 {
		displayName = "User ..." + idStr[len(idStr)-6:]
	}

	// Default avatar
	userIDInt, _ := strconv.ParseUint(idStr, 10, 64)
	index := (userIDInt >> 22) % 6
	avatarURL := fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)

	return &usertypes.UserProfile{
		UserID:      userID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}
