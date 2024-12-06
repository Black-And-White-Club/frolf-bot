// structs/user.go
package structs

import "github.com/Black-And-White-Club/tcr-bot/models"

// User represents a user in the system.
type User struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	DiscordID string          `json:"discord_id"`
	Role      models.UserRole `json:"role"`
}
