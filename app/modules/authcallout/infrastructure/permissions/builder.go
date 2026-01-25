package permissions

import (
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot/pkg/jwt"
)

// Permissions defines pub/sub permissions for a user.
type Permissions struct {
	Publish   PermissionSet `json:"pub"`
	Subscribe PermissionSet `json:"sub"`
}

// PermissionSet contains allow and deny patterns.
type PermissionSet struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// Builder constructs permission sets based on roles.
type Builder struct{}

// NewBuilder creates a new permission builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// ForRole builds permissions based on the user's role, guild, and user ID.
func (b *Builder) ForRole(role jwt.Role, guildID, userID string) *Permissions {
	switch role {
	case jwt.RoleViewer:
		return b.viewerPermissions(guildID)
	case jwt.RolePlayer:
		return b.playerPermissions(guildID, userID)
	case jwt.RoleEditor:
		return b.editorPermissions(guildID, userID)
	default:
		return b.viewerPermissions(guildID)
	}
}

// viewerPermissions returns read-only permissions for a guild.
func (b *Builder) viewerPermissions(guildID string) *Permissions {
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: []string{
				fmt.Sprintf("round.*.%s", guildID),
				fmt.Sprintf("leaderboard.*.%s", guildID),
				fmt.Sprintf("guild.*.%s", guildID),
			},
		},
		Publish: PermissionSet{
			Allow: []string{},
		},
	}
}

// playerPermissions returns permissions for players who can participate.
func (b *Builder) playerPermissions(guildID, userID string) *Permissions {
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: []string{
				fmt.Sprintf("round.*.%s", guildID),
				fmt.Sprintf("leaderboard.*.%s", guildID),
				fmt.Sprintf("guild.*.%s", guildID),
				fmt.Sprintf("score.*.%s", userID),
				fmt.Sprintf("user.*.%s", userID),
			},
		},
		Publish: PermissionSet{
			Allow: []string{
				fmt.Sprintf("round.participant.join.%s", guildID),
				fmt.Sprintf("round.participant.leave.%s", guildID),
			},
		},
	}
}

// editorPermissions returns full permissions for editors.
func (b *Builder) editorPermissions(guildID, userID string) *Permissions {
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: []string{
				fmt.Sprintf("round.*.%s", guildID),
				fmt.Sprintf("leaderboard.*.%s", guildID),
				fmt.Sprintf("guild.*.%s", guildID),
				fmt.Sprintf("score.*.%s", guildID),
				fmt.Sprintf("user.*.%s", userID),
			},
		},
		Publish: PermissionSet{
			Allow: []string{
				fmt.Sprintf("round.create.%s", guildID),
				fmt.Sprintf("round.update.%s", guildID),
				fmt.Sprintf("round.delete.%s", guildID),
				fmt.Sprintf("round.participant.*.%s", guildID),
				fmt.Sprintf("score.submit.%s", guildID),
			},
		},
	}
}
