// user/db/models.go
package userdb // Update the package name

import (
	"github.com/uptrace/bun"
)

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64    `bun:"id,pk,autoincrement" json:"id"`
	Name          string   `bun:"name" json:"name"`
	DiscordID     string   `bun:"discord_id,notnull" json:"discord_id"`
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
