package userservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	// ProfileSyncStaleness defines how long we wait before re-syncing profile from Discord
	ProfileSyncStaleness = 24 * time.Hour
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
		// Validation: Don't overwrite display name with empty string if we have an existing one.
		// This guards against sync events that might have missing nickname data.
		if displayName == "" {
			if existingUser, err := s.repo.GetUserGlobal(ctx, db, userID); err == nil && existingUser != nil && existingUser.DisplayName != nil {
				displayName = *existingUser.DisplayName
			}
		}

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

			// If the incoming display name is empty (e.g. from a partial sync),
			// avoid overwriting a valid existing display name in the club membership.
			if displayName == "" && existing != nil && existing.DisplayName != nil && *existing.DisplayName != "" {
				displayNameToUse := *existing.DisplayName
				membership.DisplayName = &displayNameToUse
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
	guildID sharedtypes.GuildID,
) (results.OperationResult[*LookupProfilesResponse, error], error) {
	userIdForTelemetry := sharedtypes.DiscordID("bulk")
	if len(userIDs) > 0 {
		userIdForTelemetry = userIDs[0] + "..."
	}

	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[*LookupProfilesResponse, error], error) {
		return s.executeLookupProfiles(ctx, db, userIDs, guildID)
	}

	return withTelemetry(s, ctx, "LookupProfiles", userIdForTelemetry, func(ctx context.Context) (results.OperationResult[*LookupProfilesResponse, error], error) {
		return runInTx(s, ctx, op)
	})
}

func (s *UserService) executeLookupProfiles(
	ctx context.Context,
	db bun.IDB,
	userIDs []sharedtypes.DiscordID,
	guildID sharedtypes.GuildID,
) (results.OperationResult[*LookupProfilesResponse, error], error) {
	if len(userIDs) == 0 {
		return results.SuccessResult[*LookupProfilesResponse, error](&LookupProfilesResponse{Profiles: make(map[sharedtypes.DiscordID]*usertypes.UserProfile)}), nil
	}

	users, err := s.repo.GetByUserIDs(ctx, db, userIDs)
	if err != nil {
		return results.FailureResult[*LookupProfilesResponse, error](err), nil
	}

	responseProfiles := make(map[sharedtypes.DiscordID]*usertypes.UserProfile, len(users))
	result := &LookupProfilesResponse{
		Profiles:     responseProfiles,
		SyncRequests: make([]*userevents.UserProfileSyncRequestPayloadV1, 0),
	}

	clubUUID := uuid.Nil
	if guildID != "" {
		// GetClubUUIDByDiscordGuildID error is intentionally ignored: if no club is
		// configured for this guild or the lookup fails transiently, club membership
		// enrichment is skipped gracefully. Contrast with UpdateUserProfile where this
		// call propagates errors.
		if resolvedClubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID); err == nil {
			clubUUID = resolvedClubUUID
		}
	}

	// Batch club membership lookup for all requested user IDs (one query replaces N).
	clubMembershipByExternalID := make(map[string]*userdb.ClubMembership)
	if clubUUID != uuid.Nil {
		allExternalIDs := make([]string, len(userIDs))
		for i, id := range userIDs {
			allExternalIDs[i] = string(id)
		}
		// GetClubMembershipsByExternalIDs error is intentionally ignored: a
		// transient failure here skips enrichment but still returns the base
		// profiles. Contrast with UpdateUserProfile where this propagates.
		memberships, err := s.repo.GetClubMembershipsByExternalIDs(ctx, db, allExternalIDs, clubUUID)
		if err != nil {
			s.logger.WarnContext(ctx, "club membership batch lookup failed, skipping enrichment", "error", err)
			memberships = nil
		}
		for _, m := range memberships {
			if m.ExternalID != nil {
				clubMembershipByExternalID[*m.ExternalID] = m
			}
		}
	}

	for _, u := range users {
		profile := &usertypes.UserProfile{
			UserID:        u.GetUserID(),
			DisplayName:   u.GetDisplayName(),
			AvatarURL:     u.AvatarURL(64), // 64px for list views
			UDiscUsername: u.UDiscUsername,
			UDiscName:     u.UDiscName,
		}
		var clubProfile *usertypes.UserProfile
		if m, ok := clubMembershipByExternalID[string(u.GetUserID())]; ok {
			clubProfile = userProfileFromClubMembership(u.GetUserID(), m)
		}
		profile = mergePreferredClubMembershipProfile(profile, clubProfile)
		responseProfiles[u.GetUserID()] = profile

		// Trigger profile sync if data is missing or stale
		if guildID != "" {
			needsSync := u.DisplayName == nil || *u.DisplayName == "" ||
				u.ProfileUpdatedAt == nil ||
				time.Since(*u.ProfileUpdatedAt) > ProfileSyncStaleness

			if needsSync && shouldSyncDiscordProfile(u.GetUserID()) {
				result.SyncRequests = append(result.SyncRequests, &userevents.UserProfileSyncRequestPayloadV1{
					UserID:  u.GetUserID(),
					GuildID: guildID,
				})
			}
		}
	}

	// For users not in DB, generate default profiles
	for _, id := range userIDs {
		if _, exists := responseProfiles[id]; !exists {
			if m, ok := clubMembershipByExternalID[string(id)]; ok {
				if profile := userProfileFromClubMembership(id, m); profile != nil {
					responseProfiles[id] = profile
					continue
				}
			}

			responseProfiles[id] = s.generateDefaultProfile(id)
			// Also request sync for new users we haven't seen before
			if guildID != "" && shouldSyncDiscordProfile(id) {
				result.SyncRequests = append(result.SyncRequests, &userevents.UserProfileSyncRequestPayloadV1{
					UserID:  id,
					GuildID: guildID,
				})
			}
		}
	}

	return results.SuccessResult[*LookupProfilesResponse, error](result), nil
}

// generateDefaultDisplayName produces the synthetic display name for an unknown user.
// isSyntheticLookupDisplayName must use this function for comparison — do not duplicate the format.
func generateDefaultDisplayName(userID sharedtypes.DiscordID) string {
	idStr := string(userID)
	if len(idStr) > 6 {
		return "User ..." + idStr[len(idStr)-6:]
	}
	return "User"
}

func (s *UserService) generateDefaultProfile(userID sharedtypes.DiscordID) *usertypes.UserProfile {
	displayName := generateDefaultDisplayName(userID)

	// Default avatar
	userIDInt, _ := strconv.ParseUint(string(userID), 10, 64)
	index := (userIDInt >> 22) % 6
	avatarURL := fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)

	return &usertypes.UserProfile{
		UserID:      userID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}

// isDiscordSnowflake reports whether userID looks like a Discord snowflake.
// Discord snowflakes are 17–19 digit integers. IsValid() only validates
// that the string is all-digit (e.g. "5" passes IsValid); the length gate
// here is the actual discriminator between short external IDs and real snowflakes.
func isDiscordSnowflake(userID sharedtypes.DiscordID) bool {
	if !userID.IsValid() {
		return false
	}
	length := len(strings.TrimSpace(string(userID)))
	return length >= 17 && length <= 19
}

// shouldSyncDiscordProfile reports whether a Discord profile sync should be
// requested for userID. Short numeric IDs (e.g. tag numbers used as external
// IDs) are not Discord users and must not trigger sync requests.
func shouldSyncDiscordProfile(userID sharedtypes.DiscordID) bool {
	return isDiscordSnowflake(userID)
}

func mergePreferredClubMembershipProfile(
	current *usertypes.UserProfile,
	clubProfile *usertypes.UserProfile,
) *usertypes.UserProfile {
	if clubProfile == nil {
		return current
	}
	if current == nil {
		return clubProfile
	}

	merged := *current
	if shouldPreferClubMembershipDisplayName(current.UserID, current.DisplayName, clubProfile.DisplayName) {
		merged.DisplayName = clubProfile.DisplayName
	}
	if strings.TrimSpace(merged.AvatarURL) == "" && strings.TrimSpace(clubProfile.AvatarURL) != "" {
		merged.AvatarURL = clubProfile.AvatarURL
	}

	return &merged
}

func shouldPreferClubMembershipDisplayName(
	userID sharedtypes.DiscordID,
	currentDisplayName string,
	clubDisplayName string,
) bool {
	if strings.TrimSpace(clubDisplayName) == "" {
		return false
	}

	if strings.TrimSpace(currentDisplayName) == "" {
		return true
	}

	if !isDiscordSnowflake(userID) {
		return true
	}

	return isSyntheticLookupDisplayName(userID, currentDisplayName) ||
		strings.Contains(strings.ToLower(strings.TrimSpace(currentDisplayName)), "placeholder")
}

func isSyntheticLookupDisplayName(userID sharedtypes.DiscordID, displayName string) bool {
	trimmed := strings.TrimSpace(displayName)
	if trimmed == "" {
		return false
	}
	return trimmed == generateDefaultDisplayName(userID)
}

func userProfileFromClubMembership(userID sharedtypes.DiscordID, membership *userdb.ClubMembership) *usertypes.UserProfile {
	if membership == nil {
		return nil
	}

	displayName := ""
	if membership.DisplayName != nil {
		displayName = strings.TrimSpace(*membership.DisplayName)
	}
	avatarURL := ""
	if membership.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*membership.AvatarURL)
	}
	if displayName == "" && avatarURL == "" {
		return nil
	}

	return &usertypes.UserProfile{
		UserID:      userID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}
