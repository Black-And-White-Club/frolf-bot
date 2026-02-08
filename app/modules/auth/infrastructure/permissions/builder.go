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

// unique returns a slice with unique strings.
func unique(strings []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strings {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// ForRole builds permissions based on the user's claims.
func (b *Builder) ForRole(claims *authdomain.Claims) *Permissions {
	userUUID := claims.UserUUID.String()
	userID := claims.UserID

	perms := &Permissions{
		Subscribe: PermissionSet{Allow: []string{}},
		Publish:   PermissionSet{Allow: []string{}},
	}

	// Add permissions for each club membership
	for _, membership := range claims.Clubs {
		clubUUID := membership.ClubUUID.String()

		// For now, we use a simple mapping for the legacy guild ID if it matches the active club.
		// If we had the actual guild IDs for all clubs in the claims, we'd use them here.
		// Since we don't, we primarily rely on the UUIDs which is the new standard.
		var guildID string
		if membership.ClubUUID == claims.ActiveClubUUID {
			guildID = claims.GuildID
		}

		var clubPerms *Permissions
		switch membership.Role {
		case authdomain.RoleViewer:
			clubPerms = b.viewerPermissions(clubUUID, guildID)
		case authdomain.RolePlayer, authdomain.Role("User"):
			clubPerms = b.playerPermissions(clubUUID, userUUID, guildID, userID)
		case authdomain.RoleEditor, authdomain.Role("Editor"), authdomain.RoleAdmin:
			clubPerms = b.editorPermissions(clubUUID, userUUID, guildID, userID)
		default:
			clubPerms = b.viewerPermissions(clubUUID, guildID)
		}

		// Merge permissions
		perms.Subscribe.Allow = append(perms.Subscribe.Allow, clubPerms.Subscribe.Allow...)
		perms.Publish.Allow = append(perms.Publish.Allow, clubPerms.Publish.Allow...)
	}

	// Add base permissions
	b.ensureBasePermissions(perms)

	// Ensure unique entries
	perms.Subscribe.Allow = unique(perms.Subscribe.Allow)
	perms.Subscribe.Deny = unique(perms.Subscribe.Deny)
	perms.Publish.Allow = unique(perms.Publish.Allow)
	perms.Publish.Deny = unique(perms.Publish.Deny)

	return perms
}

// ensureBasePermissions adds standard permissions required by all users (e.g. Inboxes)
func (b *Builder) ensureBasePermissions(perms *Permissions) {
	// Allow subscribing to own inbox for request-reply patterns
	perms.Subscribe.Allow = append(perms.Subscribe.Allow, "_INBOX.>")
}

// viewerPermissions returns read-only permissions for a club/guild.
func (b *Builder) viewerPermissions(clubUUID, guildID string) *Permissions {
	allow := subscribePatterns(clubUUID, guildID, "", "")
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: allow,
		},
		Publish: PermissionSet{
			Allow: []string{},
		},
	}
}

// subscribePatterns generates subscribe patterns for round/leaderboard/guild events.
// Subjects follow pattern: {domain}.{action}[.{sub}...].v1.{id}
// Examples: round.created.v1.{id}, round.participant.joined.v1.{id}, round.participant.score.updated.v1.{id}
func subscribePatterns(clubUUID, guildID, userUUID, userID string) []string {
	patterns := []string{}

	for _, id := range []string{clubUUID, guildID} {
		if id == "" {
			continue
		}
		// round.*.v1.{id} - matches round.created.v1.{id}, round.updated.v1.{id}, etc. (4 tokens)
		// round.*.*.v1.{id} - matches round.participant.joined.v1.{id}, round.list.request.v1.{id} (5 tokens)
		// round.*.*.*.v1.{id} - matches round.participant.score.updated.v1.{id} (6 tokens)
		patterns = append(patterns,
			fmt.Sprintf("round.*.v1.%s", id),
			fmt.Sprintf("round.*.*.v1.%s", id),
			fmt.Sprintf("round.*.*.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.*.*.v1.%s", id),
			fmt.Sprintf("guild.*.v1.%s", id),
			fmt.Sprintf("guild.*.*.v1.%s", id),
		)
	}

	// User-specific patterns
	for _, id := range []string{userUUID, userID} {
		if id == "" {
			continue
		}
		patterns = append(patterns,
			fmt.Sprintf("score.*.v1.%s", id),
			fmt.Sprintf("score.*.*.v1.%s", id),
			fmt.Sprintf("user.*.v1.%s", id),
			fmt.Sprintf("user.*.*.v1.%s", id),
		)
	}

	return patterns
}

// publishPatterns generates publish patterns for request-reply calls.
func publishPatterns(clubUUID, guildID string, includeParticipant bool) []string {
	patterns := []string{}

	for _, id := range []string{clubUUID, guildID} {
		if id == "" {
			continue
		}
		// Request-reply patterns for fetching data
		patterns = append(patterns,
			fmt.Sprintf("round.list.request.v1.%s", id),
			fmt.Sprintf("leaderboard.snapshot.request.v1.%s", id),
		)
		if includeParticipant {
			// Participant actions
			patterns = append(patterns,
				fmt.Sprintf("round.participant.join.v1.%s", id),
				fmt.Sprintf("round.participant.leave.v1.%s", id),
			)
		}
	}

	// Club info request - scoped to the user's active club
	if clubUUID != "" {
		patterns = append(patterns, fmt.Sprintf("club.info.request.v1.%s", clubUUID))
	}

	return patterns
}

// playerPermissions returns permissions for players who can participate.
func (b *Builder) playerPermissions(clubUUID, userUUID, guildID, userID string) *Permissions {
	return &Permissions{
		Subscribe: PermissionSet{
			Allow: subscribePatterns(clubUUID, guildID, userUUID, userID),
		},
		Publish: PermissionSet{
			Allow: publishPatterns(clubUUID, guildID, true),
		},
	}
}

// editorPermissions returns full permissions for editors.
func (b *Builder) editorPermissions(clubUUID, userUUID, guildID, userID string) *Permissions {
	pubAllow := publishPatterns(clubUUID, guildID, true)

	// Editors can also create/update/delete rounds and submit scores
	for _, id := range []string{clubUUID, guildID} {
		if id == "" {
			continue
		}
		pubAllow = append(pubAllow,
			fmt.Sprintf("round.create.v1.%s", id),
			fmt.Sprintf("round.update.v1.%s", id),
			fmt.Sprintf("round.delete.v1.%s", id),
			fmt.Sprintf("score.submit.v1.%s", id),
		)
	}

	return &Permissions{
		Subscribe: PermissionSet{
			Allow: subscribePatterns(clubUUID, guildID, userUUID, userID),
		},
		Publish: PermissionSet{
			Allow: pubAllow,
		},
	}
}
