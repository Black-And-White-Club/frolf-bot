package adapters

import (
	"context"

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
