package userqueries

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
)

// userQueryService implements the QueryService interface.
type userQueryService struct {
	getUserByDiscordIDHandler GetUserByDiscordIDHandler
	getUserRoleHandler        GetUserRoleHandler
}

// NewUserQueryService creates a new QueryService instance.
func NewUserQueryService(getUserByDiscordIDHandler GetUserByDiscordIDHandler, getUserRoleHandler GetUserRoleHandler) QueryService {
	return &userQueryService{
		getUserByDiscordIDHandler: getUserByDiscordIDHandler,
		getUserRoleHandler:        getUserRoleHandler,
	}
}

func (s *userQueryService) GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error) {
	return s.getUserByDiscordIDHandler.Handle(ctx, GetUserByDiscordID{DiscordID: discordID})
}

func (s *userQueryService) GetUserRole(ctx context.Context, discordID string) (userdb.UserRole, error) {
	return s.getUserRoleHandler.Handle(ctx, GetUserRole{DiscordID: discordID})
}
