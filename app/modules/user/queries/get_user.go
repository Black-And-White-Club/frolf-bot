package userqueries

import (
	"context"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
)

// GetUserByDiscordID represents the query to get a user by their Discord ID.
type GetUserByDiscordID struct {
	DiscordID string
}

// GetUserByDiscordIDHandler is an interface for handlers that process GetUserByDiscordID queries.
type GetUserByDiscordIDHandler interface {
	Handle(ctx context.Context, query GetUserByDiscordID) (*userdb.User, error)
}

type getUserByDiscordIDHandler struct {
	userDB userdb.UserDB
}

func NewGetUserByDiscordIDHandler(userDB userdb.UserDB) *getUserByDiscordIDHandler {
	return &getUserByDiscordIDHandler{userDB: userDB}
}

func (h *getUserByDiscordIDHandler) Handle(ctx context.Context, query GetUserByDiscordID) (*userdb.User, error) {
	user, err := h.userDB.GetUserByDiscordID(ctx, query.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil // Removed the "user not found" check
}
