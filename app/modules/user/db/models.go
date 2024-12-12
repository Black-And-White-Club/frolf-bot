package userdb

import "github.com/uptrace/bun"

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64    `bun:"id,pk,autoincrement" json:"id"`
	Name          string   `bun:"name" json:"name"`
	DiscordID     string   `bun:"discord_id,unique"`
	Role          UserRole `bun:"role,notnull" json:"role"`
}

// UserRole represents the role of a user.
type UserRole string

// Constants for user roles
const (
	UserRoleRattler UserRole = "Rattler"
	UserRoleEditor  UserRole = "Editor"
	UserRoleAdmin   UserRole = "Admin"
)

// IsValid checks if the given role is valid
func (ur UserRole) IsValid() bool {
	switch ur {
	case UserRoleRattler, UserRoleEditor, UserRoleAdmin:
		return true
	default:
		return false
	}
}
