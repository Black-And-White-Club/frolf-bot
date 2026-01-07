package adapters

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// UserLookupAdapter adapts the user repository to the round service UserLookup port.
type UserLookupAdapter struct {
	userDB userdb.UserDB
}

// NewUserLookupAdapter constructs a new adapter.
func NewUserLookupAdapter(db userdb.UserDB) *UserLookupAdapter {
	return &UserLookupAdapter{userDB: db}
}

func (a *UserLookupAdapter) FindByNormalizedUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, normalizedUsername string) (*roundservice.UserIdentity, error) {
	user, err := a.userDB.FindByUDiscUsername(ctx, guildID, normalizedUsername)
	if err != nil {
		if err == userdb.ErrUserNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &roundservice.UserIdentity{UserID: user.User.UserID}, nil
}

func (a *UserLookupAdapter) FindByNormalizedUDiscDisplayName(ctx context.Context, guildID sharedtypes.GuildID, normalizedDisplayName string) (*roundservice.UserIdentity, error) {
	user, err := a.userDB.FindByUDiscName(ctx, guildID, normalizedDisplayName)
	if err != nil {
		if err == userdb.ErrUserNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &roundservice.UserIdentity{UserID: user.User.UserID}, nil
}
