package models

import (
	"github.com/uptrace/bun"
)

// UserRole represents the possible roles for a user.
type UserRole string

// Define the possible user role values as constants.
const (
	UserRoleRattler UserRole = "RATTLER"
	UserRoleAdmin   UserRole = "ADMIN"
	UserRoleEditor  UserRole = "EDITOR"
)

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int64    `bun:"id,pk,autoincrement"`
	Name      string   `bun:"name,notnull"`
	DiscordID string   `bun:"discord_id,notnull"`
	Role      UserRole `bun:"role,notnull"`
}
