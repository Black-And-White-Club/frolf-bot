package adapters

import (
	"context"
	"strings"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// UserLookupAdapter adapts the user repository to the round service UserLookup port.
type UserLookupAdapter struct {
	userDB userdb.Repository
	db     bun.IDB
}

// NewUserLookupAdapter constructs a new adapter.
func NewUserLookupAdapter(userDB userdb.Repository, db bun.IDB) *UserLookupAdapter {
	return &UserLookupAdapter{
		userDB: userDB,
		db:     db,
	}
}

func (a *UserLookupAdapter) FindByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedUsername string) (*roundservice.UserIdentity, error) {
	if db == nil {
		db = a.db
	}
	user, err := a.userDB.FindByUDiscUsername(ctx, db, guildID, normalizedUsername)
	if err != nil {
		if err == userdb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &roundservice.UserIdentity{UserID: user.User.GetUserID()}, nil
}

func (a *UserLookupAdapter) FindGlobalByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, normalizedUsername string) (*roundservice.UserIdentity, error) {
	if db == nil {
		db = a.db
	}
	user, err := a.userDB.FindGlobalByUDiscUsername(ctx, db, normalizedUsername)
	if err != nil {
		if err == userdb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &roundservice.UserIdentity{UserID: user.GetUserID()}, nil
}

func (a *UserLookupAdapter) FindByNormalizedUDiscDisplayName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedDisplayName string) (*roundservice.UserIdentity, error) {
	if db == nil {
		db = a.db
	}
	user, err := a.userDB.FindByUDiscName(ctx, db, guildID, normalizedDisplayName)
	if err != nil {
		if err == userdb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &roundservice.UserIdentity{UserID: user.User.GetUserID()}, nil
}

func (a *UserLookupAdapter) FindByPartialUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*roundservice.UserIdentity, error) {
	if db == nil {
		db = a.db
	}
	users, err := a.userDB.FindByUDiscNameFuzzy(ctx, db, guildID, partialName)
	if err != nil {
		return nil, err
	}

	// Convert to UserIdentity slice
	identities := make([]*roundservice.UserIdentity, 0, len(users))
	for _, u := range users {
		identities = append(identities, &roundservice.UserIdentity{
			UserID: u.User.GetUserID(),
		})
	}

	return identities, nil
}

// ResolveByNormalizedNames performs batched user resolution while preserving
// the existing precedence: display name -> guild username -> global username.
func (a *UserLookupAdapter) ResolveByNormalizedNames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedNames []string) (map[string]sharedtypes.DiscordID, error) {
	if db == nil {
		db = a.db
	}

	requested := toSet(normalizedNames)
	resolved := make(map[string]sharedtypes.DiscordID, len(requested))
	if len(requested) == 0 {
		return resolved, nil
	}

	keys := setKeys(requested)

	// 1) Guild-scoped display name lookup
	byName, err := a.userDB.GetUsersByUDiscNames(ctx, db, guildID, keys)
	if err != nil {
		return nil, err
	}
	for _, user := range byName {
		if user.User == nil || user.User.UDiscName == nil {
			continue
		}
		key := normalize(*user.User.UDiscName)
		if _, ok := requested[key]; !ok {
			continue
		}
		if _, matched := resolved[key]; matched {
			continue
		}
		resolved[key] = user.User.GetUserID()
	}

	// 2) Guild-scoped username lookup (with @-aware candidate expansion)
	usernameCandidates := expandUsernameLookupValues(requested, resolved)
	byUsername, err := a.userDB.GetUsersByUDiscUsernames(ctx, db, guildID, usernameCandidates)
	if err != nil {
		return nil, err
	}
	for _, user := range byUsername {
		if user.User == nil || user.User.UDiscUsername == nil {
			continue
		}
		for _, key := range usernameResolutionKeys(*user.User.UDiscUsername) {
			if _, ok := requested[key]; !ok {
				continue
			}
			if _, matched := resolved[key]; matched {
				continue
			}
			resolved[key] = user.User.GetUserID()
		}
	}

	// 3) Global username fallback for still-unmatched keys
	globalCandidates := expandUsernameLookupValues(requested, resolved)
	globalUsers, err := a.userDB.GetGlobalUsersByUDiscUsernames(ctx, db, globalCandidates)
	if err != nil {
		return nil, err
	}
	for _, user := range globalUsers {
		if user == nil || user.UDiscUsername == nil {
			continue
		}
		for _, key := range usernameResolutionKeys(*user.UDiscUsername) {
			if _, ok := requested[key]; !ok {
				continue
			}
			if _, matched := resolved[key]; matched {
				continue
			}
			resolved[key] = user.GetUserID()
		}
	}

	return resolved, nil
}

func toSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalize(value)
		if normalized == "" {
			continue
		}
		result[normalized] = struct{}{}
	}
	return result
}

func setKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	return keys
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func expandUsernameLookupValues(requested map[string]struct{}, resolved map[string]sharedtypes.DiscordID) []string {
	candidateSet := make(map[string]struct{})
	for key := range requested {
		if _, matched := resolved[key]; matched {
			continue
		}
		base := strings.TrimPrefix(key, "@")
		if base == "" {
			continue
		}
		candidateSet[base] = struct{}{}
		candidateSet["@"+base] = struct{}{}
	}

	return setKeys(candidateSet)
}

func usernameResolutionKeys(rawUsername string) []string {
	normalized := normalize(rawUsername)
	if normalized == "" {
		return nil
	}

	base := strings.TrimPrefix(normalized, "@")
	if base == "" {
		return []string{normalized}
	}

	if normalized == base {
		return []string{base, "@" + base}
	}

	return []string{normalized, base}
}
