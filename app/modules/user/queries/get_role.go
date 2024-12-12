package userqueries

import (
	"context"
	"errors"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
)

type GetUserRole struct {
	DiscordID string
}

type GetUserRoleHandler interface {
	Handle(ctx context.Context, query GetUserRole) (userdb.UserRole, error)
}

type getUserRoleHandler struct {
	userDB userdb.UserDB
}

func NewGetUserRoleHandler(userDB userdb.UserDB) *getUserRoleHandler {
	return &getUserRoleHandler{userDB: userDB}
}

func (h *getUserRoleHandler) Handle(ctx context.Context, query GetUserRole) (userdb.UserRole, error) {
	user, err := h.userDB.GetUserByDiscordID(ctx, query.DiscordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	return user.Role, nil
}
