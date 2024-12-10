// user/db/models.go
package userdb // Update the package name

import (
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/uptrace/bun"
)

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64                  `bun:"id,pk,autoincrement" json:"id"`
	Name          string                 `bun:"name" json:"name"`
	DiscordID     string                 `bun:"discord_id,notnull" json:"discord_id"`
	Role          userapimodels.UserRole `json:"role"` // Use the UserRole from api_models
}
