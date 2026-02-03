package permissions

import (
	"fmt"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
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

// ForRole builds permissions based on the user's claims.
func (b *Builder) ForRole(claims *authdomain.Claims) *Permissions {
	clubUUID := claims.ActiveClubUUID.String()
	userUUID := claims.UserUUID.String()
	guildID := claims.GuildID
	userID := claims.UserID

	switch claims.Role {
	case authdomain.RoleViewer:
		return b.viewerPermissions(clubUUID, guildID)
	case authdomain.RolePlayer:
		return b.playerPermissions(clubUUID, userUUID, guildID, userID)
	case authdomain.RoleEditor:
		return b.editorPermissions(clubUUID, userUUID, guildID, userID)
	default:
		return b.viewerPermissions(clubUUID, guildID)
	}
}

// viewerPermissions returns read-only permissions for a club/guild.
func (b *Builder) viewerPermissions(clubUUID, guildID string) *Permissions {
	allow := []string{
		fmt.Sprintf("round.*.%s", clubUUID),
		fmt.Sprintf("leaderboard.*.%s", clubUUID),
		fmt.Sprintf("guild.*.%s", clubUUID),
	}
	if guildID != "" {
		allow = append(allow,
			fmt.Sprintf("round.*.%s", guildID),
			fmt.Sprintf("leaderboard.*.%s", guildID),
			fmt.Sprintf("guild.*.%s", guildID),
		)
	}
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: allow,
		},
		Publish: PermissionSet{
			Allow: []string{},
		},
	}
}

// playerPermissions returns permissions for players who can participate.
func (b *Builder) playerPermissions(clubUUID, userUUID, guildID, userID string) *Permissions {
	subAllow := []string{
		fmt.Sprintf("round.*.%s", clubUUID),
		fmt.Sprintf("leaderboard.*.%s", clubUUID),
		fmt.Sprintf("guild.*.%s", clubUUID),
		fmt.Sprintf("score.*.%s", userUUID),
		fmt.Sprintf("user.*.%s", userUUID),
	}
	pubAllow := []string{
		fmt.Sprintf("round.participant.join.%s", clubUUID),
		fmt.Sprintf("round.participant.leave.%s", clubUUID),
	}

	if guildID != "" {
		subAllow = append(subAllow,
			fmt.Sprintf("round.*.%s", guildID),
			fmt.Sprintf("leaderboard.*.%s", guildID),
			fmt.Sprintf("guild.*.%s", guildID),
		)
		pubAllow = append(pubAllow,
			fmt.Sprintf("round.participant.join.%s", guildID),
			fmt.Sprintf("round.participant.leave.%s", guildID),
		)
	}
	if userID != "" {
		subAllow = append(subAllow,
			fmt.Sprintf("score.*.%s", userID),
			fmt.Sprintf("user.*.%s", userID),
		)
	}

	return &Permissions{
		Subscribe: PermissionSet{
			Allow: subAllow,
		},
		Publish: PermissionSet{
			Allow: pubAllow,
		},
	}
}

// editorPermissions returns full permissions for editors.
func (b *Builder) editorPermissions(clubUUID, userUUID, guildID, userID string) *Permissions {
	subAllow := []string{
		fmt.Sprintf("round.*.%s", clubUUID),
		fmt.Sprintf("leaderboard.*.%s", clubUUID),
		fmt.Sprintf("guild.*.%s", clubUUID),
		fmt.Sprintf("score.*.%s", clubUUID),
		fmt.Sprintf("user.*.%s", userUUID),
	}
	pubAllow := []string{
		fmt.Sprintf("round.create.%s", clubUUID),
		fmt.Sprintf("round.update.%s", clubUUID),
		fmt.Sprintf("round.delete.%s", clubUUID),
		fmt.Sprintf("round.participant.*.%s", clubUUID),
		fmt.Sprintf("score.submit.%s", clubUUID),
	}

	if guildID != "" {
		subAllow = append(subAllow,
			fmt.Sprintf("round.*.%s", guildID),
			fmt.Sprintf("leaderboard.*.%s", guildID),
			fmt.Sprintf("guild.*.%s", guildID),
			fmt.Sprintf("score.*.%s", guildID),
		)
		pubAllow = append(pubAllow,
			fmt.Sprintf("round.create.%s", guildID),
			fmt.Sprintf("round.update.%s", guildID),
			fmt.Sprintf("round.delete.%s", guildID),
			fmt.Sprintf("round.participant.*.%s", guildID),
			fmt.Sprintf("score.submit.%s", guildID),
		)
	}
	if userID != "" {
		subAllow = append(subAllow,
			fmt.Sprintf("user.*.%s", userID),
		)
	}

	return &Permissions{
		Subscribe: PermissionSet{
			Allow: subAllow,
		},
		Publish: PermissionSet{
			Allow: pubAllow,
		},
	}
}
