// models/user.go
package models

// User represents a user in the system.
type User struct {
	ID        int64    `bun:"id,pk,autoincrement" json:"id"`
	Name      string   `bun:"name,notnull" json:"name"`
	DiscordID string   `bun:"discord_id,notnull" json:"discord_id"`
	Role      UserRole `bun:"role,notnull" json:"role"`
}

// UserRole represents the possible roles for a user.
type UserRole string

// Define the possible user role values as constants.
const (
	UserRoleRattler UserRole = "RATTLER"
	UserRoleAdmin   UserRole = "ADMIN"
	UserRoleEditor  UserRole = "EDITOR"
)
