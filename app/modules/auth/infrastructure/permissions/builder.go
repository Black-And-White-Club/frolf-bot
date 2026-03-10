package permissions

import (
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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
		case authdomain.RoleEditor, authdomain.Role("Editor"):
			clubPerms = b.editorPermissions(clubUUID, userUUID, guildID, userID)
		case authdomain.RoleAdmin:
			clubPerms = b.adminPermissions(clubUUID, userUUID, guildID, userID)
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
// Scoped subjects follow pattern: {domain}.{action}[.{sub}...].v{n}.{id}
// Round and leaderboard subjects are allowed on both v1 and v2 during the migration.
// TODO: drop the v1 patterns after all scoped producers and consumers have fully
// moved to v2 subjects. Keeping them indefinitely bloats every issued token.
func subscribePatterns(clubUUID, guildID, userUUID, userID string) []string {
	patterns := []string{}

	for _, id := range []string{clubUUID, guildID} {
		if id == "" {
			continue
		}
		// round.*.v{1,2}.{id} - matches round.created.v2.{id}, round.updated.v2.{id}, etc. (4 tokens)
		// round.*.*.v{1,2}.{id} - matches round.participant.joined.v2.{id}, round.list.request.v2.{id} (5 tokens)
		// round.*.*.*.v{1,2}.{id} - matches round.participant.score.updated.v2.{id} (6 tokens)
		// leaderboard.*... follows the same migration pattern. Guild subjects remain v1 today.
		patterns = append(patterns,
			fmt.Sprintf("round.*.v1.%s", id),
			fmt.Sprintf("round.*.v2.%s", id),
			fmt.Sprintf("round.*.*.v1.%s", id),
			fmt.Sprintf("round.*.*.v2.%s", id),
			fmt.Sprintf("round.*.*.*.v1.%s", id),
			fmt.Sprintf("round.*.*.*.v2.%s", id),
			fmt.Sprintf("leaderboard.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.v2.%s", id),
			fmt.Sprintf("leaderboard.*.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.*.v2.%s", id),
			fmt.Sprintf("leaderboard.*.*.*.v1.%s", id),
			fmt.Sprintf("leaderboard.*.*.*.v2.%s", id),
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
//
// Read/query subjects (list, snapshot, etc.) are scoped to the club or guild UUID so
// the NATS server limits a token to its own club's data.
//
// Write/action subjects (participant join, round create, etc.) are intentionally
// unscoped — no club/guild suffix. Unscoped write subjects keep multi-club scenarios
// simple and avoid subject proliferation.
//
// Trade-off: NATS alone cannot prevent an authenticated user from publishing to a
// write subject with a foreign guild's ID in the payload. At present there is no
// handler-level claim verification (context GuildID comes from message metadata, not
// from the NATS JWT). Cross-guild payload spoofing is therefore not prevented at the
// application layer today.
//
// TODO: propagate NATS JWT claims into handler context and verify payload.GuildID
// against the authenticated claim before processing any write subject.
func publishPatterns(clubUUID, guildID string, includeParticipant bool) []string {
	patterns := []string{}

	for _, id := range []string{clubUUID, guildID} {
		if id == "" {
			continue
		}
		// Request-reply patterns for fetching data
		patterns = append(patterns,
			fmt.Sprintf("round.list.request.v2.%s", id),
			fmt.Sprintf("leaderboard.snapshot.request.v2.%s", id),
			fmt.Sprintf("leaderboard.tag.list.requested.v1.%s", id),
			fmt.Sprintf("leaderboard.tag.history.requested.v1.%s", id),
			fmt.Sprintf("leaderboard.tag.graph.requested.v1.%s", id),
			fmt.Sprintf("season.list.requested.v1.%s", id),
			fmt.Sprintf("season.standings.requested.v1.%s", id),
		)
	}

	if includeParticipant {
		// Participant actions
		patterns = append(patterns,
			"round.participant.join.requested.v2",
			"round.participant.declined.v1",
			"round.participant.removal.requested.v2",
		)
	}

	// Allow authenticated users to manage their UDisc identity from the PWA.
	patterns = append(patterns, "user.udisc.identity.update.requested.v1")

	// Club info request - scoped to the user's active club
	if clubUUID != "" {
		patterns = append(patterns, fmt.Sprintf("club.info.request.v2.%s", clubUUID))
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

	// Editors can also create/update/delete rounds.
	pubAllow = append(pubAllow,
		"round.creation.requested.v2",
		"round.update.requested.v2",
		"round.delete.requested.v2",
		"round.score.update.requested.v2",
		leaderboardevents.LeaderboardPointHistoryRequestedV1,
		leaderboardevents.LeaderboardGetSeasonStandingsV1,
	)

	return &Permissions{
		Subscribe: PermissionSet{
			Allow: subscribePatterns(clubUUID, guildID, userUUID, userID),
		},
		Publish: PermissionSet{
			Allow: pubAllow,
		},
	}
}

// adminPermissions returns full permissions for admins, extending editor permissions
// with the ability to publish admin operations and subscribe to their feedback topics.
// Admin subjects do not carry a guildId suffix; guild scoping is done via payload.guild_id.
func (b *Builder) adminPermissions(clubUUID, userUUID, guildID, userID string) *Permissions {
	editor := b.editorPermissions(clubUUID, userUUID, guildID, userID)

	// Admin-only publish subjects (unscoped global topics)
	editor.Publish.Allow = append(editor.Publish.Allow,
		leaderboardevents.LeaderboardManualPointAdjustmentV2,
		leaderboardevents.LeaderboardRecalculateRoundV1,
		leaderboardevents.LeaderboardStartNewSeasonV1,
		leaderboardevents.LeaderboardEndSeasonV1,
		"leaderboard.batch.tag.assignment.requested.v2",
		"round.scorecard.admin.upload.requested.v2",
		"round.admin.backfill.check.v1",
		"round.admin.backfill.requested.v1",
	)

	// Admin-only subscribe subjects for operation feedback (unscoped global topics)
	editor.Subscribe.Allow = append(editor.Subscribe.Allow,
		"leaderboard.batch.tag.assigned.v2",
		"leaderboard.batch.tag.assignment.failed.v2",
		leaderboardevents.LeaderboardManualPointAdjustmentSuccessV2,
		leaderboardevents.LeaderboardManualPointAdjustmentFailedV2,
	)

	return editor
}
