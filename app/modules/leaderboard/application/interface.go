package leaderboardservice

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
)

// PlayerResult represents a player's performance in a round.
type PlayerResult struct {
	PlayerID  sharedtypes.DiscordID
	TagNumber int
}

// ProcessRoundResult wraps the leaderboard data and the per-player points awarded.
type ProcessRoundResult struct {
	LeaderboardData leaderboardtypes.LeaderboardData
	PointsAwarded   map[sharedtypes.DiscordID]int
}

// Service defines the contract for leaderboard operations.
// All state mutations flow through ExecuteBatchTagAssignment (The Funnel).
type Service interface {
	// --- MUTATIONS (The Funnel) ---
	// ProcessRound calculates and persists ratings and tag updates for a round.
	ProcessRound(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		roundID sharedtypes.RoundID,
		playerResults []PlayerResult,
		source sharedtypes.ServiceUpdateSource,
	) (results.OperationResult[ProcessRoundResult, error], error)

	// ExecuteBatchTagAssignment is the consolidated funnel.
	// All other mutation methods eventually call this internally.
	ExecuteBatchTagAssignment(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		requests []sharedtypes.TagAssignmentRequest,
		updateID sharedtypes.RoundID,
		source sharedtypes.ServiceUpdateSource,
	) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)

	// TagSwapRequested attempts an assignment and returns the updated leaderboard data or an error.
	TagSwapRequested(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		targetTag sharedtypes.TagNumber,
	) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error)

	// --- READS ---

	// GetLeaderboard returns the active leaderboard entries as domain types.
	GetLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error)

	// GetTagByUserID returns the tag for a user or an error.
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error)

	// RoundGetTagByUserID provides tag lookup with round-specific metadata.
	RoundGetTagByUserID(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
	) (results.OperationResult[sharedtypes.TagNumber, error], error)

	// CheckTagAvailability validates whether a tag can be assigned to a user.
	CheckTagAvailability(
		ctx context.Context,
		guildID sharedtypes.GuildID,
		userID sharedtypes.DiscordID,
		tagNumber sharedtypes.TagNumber,
	) (results.OperationResult[TagAvailabilityResult, error], error)

	// --- ADMIN OPERATIONS ---

	// GetPointHistoryForMember returns the point history for a given member.
	GetPointHistoryForMember(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, limit int) (results.OperationResult[[]PointHistoryEntry, error], error)

	// AdjustPoints manually adjusts a member's points with a reason.
	// Does NOT recalculate tiers (tiers only change on round processing).
	AdjustPoints(ctx context.Context, guildID sharedtypes.GuildID, memberID sharedtypes.DiscordID, pointsDelta int, reason string) (results.OperationResult[bool, error], error)

	// StartNewSeason creates a new season record, deactivates the old one.
	StartNewSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string, seasonName string) (results.OperationResult[bool, error], error)

	// GetSeasonStandings retrieves standings for a specific season.
	GetSeasonStandingsForSeason(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]SeasonStandingEntry, error], error)

	// ListSeasons returns all seasons for a guild, ordered by active first then start_date desc.
	ListSeasons(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[[]SeasonInfo, error], error)

	// GetSeasonName retrieves the display name for a specific season.
	GetSeasonName(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (string, error)

	// --- COMMAND-STYLE OPERATIONS (transport-agnostic) ---

	// ProcessRoundCommand runs the normalized command flow for round processing.
	ProcessRoundCommand(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error)

	// ResetTagsFromQualifyingRound clears and reassigns tags based on finish order.
	ResetTagsFromQualifyingRound(ctx context.Context, guildID sharedtypes.GuildID, finishOrder []sharedtypes.DiscordID) ([]leaderboarddomain.TagChange, error)

	// EndSeason ends the active season for a guild.
	EndSeason(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error)

	// --- TAG HISTORY ---

	// GetTagHistory returns tag history for a member or all members.
	GetTagHistory(ctx context.Context, guildID sharedtypes.GuildID, memberID string, limit int) ([]TagHistoryView, error)

	// GetTagList returns the master tag list for a guild.
	GetTagList(ctx context.Context, guildID sharedtypes.GuildID) ([]TaggedMemberView, error)

	// GenerateTagGraphPNG generates a PNG chart of a member's tag history.
	GenerateTagGraphPNG(ctx context.Context, guildID sharedtypes.GuildID, memberID string) ([]byte, error)

	// --- INFRASTRUCTURE ---
	EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error)
}

// PointHistoryEntry is a read model for point history.
type PointHistoryEntry struct {
	MemberID  sharedtypes.DiscordID
	RoundID   sharedtypes.RoundID
	SeasonID  string
	Points    int
	Reason    string
	Tier      string
	Opponents int
	CreatedAt string // ISO 8601
}

// SeasonStandingEntry is a read model for season standings.
type SeasonStandingEntry struct {
	MemberID      sharedtypes.DiscordID
	SeasonID      string
	TotalPoints   int
	CurrentTier   string
	SeasonBestTag int
	RoundsPlayed  int
}

// SeasonInfo is a read model for season listing.
type SeasonInfo struct {
	ID        string
	Name      string
	IsActive  bool
	StartDate string
	EndDate   *string
}
