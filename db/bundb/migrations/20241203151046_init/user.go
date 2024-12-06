package models

import (
	"github.com/Black-And-White-Club/tcr-bot/api/structs"
	"github.com/uptrace/bun"
)

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int64            `bun:"id,pk,autoincrement"`
	Name      string           `bun:"name,notnull"`
	DiscordID string           `bun:"discord_id,notnull"`
	Role      structs.UserRole `bun:"role,notnull"`
}
