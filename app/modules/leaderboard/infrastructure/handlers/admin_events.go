package leaderboardhandlers

import (
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Admin event constants - defined locally since the shared events package doesn't have these yet.
// These follow the same naming convention as existing shared events.
const (
	LeaderboardPointHistoryRequestedV1 = "leaderboard.point.history.requested.v1"
	LeaderboardPointHistoryResponseV1  = "leaderboard.point.history.response.v1"
	LeaderboardPointHistoryFailedV1    = "leaderboard.point.history.failed.v1"

	LeaderboardManualPointAdjustmentV1        = "leaderboard.manual.point.adjustment.v1"
	LeaderboardManualPointAdjustmentSuccessV1 = "leaderboard.manual.point.adjustment.success.v1"
	LeaderboardManualPointAdjustmentFailedV1  = "leaderboard.manual.point.adjustment.failed.v1"

	LeaderboardRecalculateRoundV1        = "leaderboard.recalculate.round.v1"
	LeaderboardRecalculateRoundSuccessV1 = "leaderboard.recalculate.round.success.v1"
	LeaderboardRecalculateRoundFailedV1  = "leaderboard.recalculate.round.failed.v1"

	LeaderboardStartNewSeasonV1        = "leaderboard.start.new.season.v1"
	LeaderboardStartNewSeasonSuccessV1 = "leaderboard.start.new.season.success.v1"
	LeaderboardStartNewSeasonFailedV1  = "leaderboard.start.new.season.failed.v1"

	LeaderboardGetSeasonStandingsV1         = "leaderboard.get.season.standings.v1"
	LeaderboardGetSeasonStandingsResponseV1 = "leaderboard.get.season.standings.response.v1"
	LeaderboardGetSeasonStandingsFailedV1   = "leaderboard.get.season.standings.failed.v1"
)

// --- Admin Event Payload Types ---

// PointHistoryRequestedPayloadV1 requests point history for a member.
type PointHistoryRequestedPayloadV1 struct {
	GuildID  sharedtypes.GuildID   `json:"guild_id"`
	MemberID sharedtypes.DiscordID `json:"member_id"`
	Limit    int                   `json:"limit,omitempty"`
}

// PointHistoryResponsePayloadV1 contains the point history response.
type PointHistoryResponsePayloadV1 struct {
	GuildID  sharedtypes.GuildID   `json:"guild_id"`
	MemberID sharedtypes.DiscordID `json:"member_id"`
	History  []PointHistoryItemV1  `json:"history"`
}

// PointHistoryItemV1 represents a single point history entry.
type PointHistoryItemV1 struct {
	RoundID   sharedtypes.RoundID `json:"round_id"`
	SeasonID  string              `json:"season_id"`
	Points    int                 `json:"points"`
	Reason    string              `json:"reason"`
	Tier      string              `json:"tier"`
	Opponents int                 `json:"opponents"`
	CreatedAt string              `json:"created_at"`
}

// ManualPointAdjustmentPayloadV1 requests a manual point adjustment.
type ManualPointAdjustmentPayloadV1 struct {
	GuildID     sharedtypes.GuildID   `json:"guild_id"`
	MemberID    sharedtypes.DiscordID `json:"member_id"`
	PointsDelta int                   `json:"points_delta"`
	Reason      string                `json:"reason"`
	AdminID     sharedtypes.DiscordID `json:"admin_id"`
}

// ManualPointAdjustmentSuccessPayloadV1 confirms a point adjustment.
type ManualPointAdjustmentSuccessPayloadV1 struct {
	GuildID     sharedtypes.GuildID   `json:"guild_id"`
	MemberID    sharedtypes.DiscordID `json:"member_id"`
	PointsDelta int                   `json:"points_delta"`
	Reason      string                `json:"reason"`
}

// RecalculateRoundPayloadV1 requests recalculation of a round.
type RecalculateRoundPayloadV1 struct {
	GuildID sharedtypes.GuildID `json:"guild_id"`
	RoundID sharedtypes.RoundID `json:"round_id"`
}

// RecalculateRoundSuccessPayloadV1 confirms round recalculation.
type RecalculateRoundSuccessPayloadV1 struct {
	GuildID       sharedtypes.GuildID           `json:"guild_id"`
	RoundID       sharedtypes.RoundID           `json:"round_id"`
	PointsAwarded map[sharedtypes.DiscordID]int `json:"points_awarded"`
}

// StartNewSeasonPayloadV1 requests a new season to be started.
type StartNewSeasonPayloadV1 struct {
	GuildID    sharedtypes.GuildID `json:"guild_id"`
	SeasonID   string              `json:"season_id"`
	SeasonName string              `json:"season_name"`
}

// StartNewSeasonSuccessPayloadV1 confirms a new season was started.
type StartNewSeasonSuccessPayloadV1 struct {
	GuildID    sharedtypes.GuildID `json:"guild_id"`
	SeasonID   string              `json:"season_id"`
	SeasonName string              `json:"season_name"`
}

// GetSeasonStandingsPayloadV1 requests standings for a season.
type GetSeasonStandingsPayloadV1 struct {
	GuildID  sharedtypes.GuildID `json:"guild_id"`
	SeasonID string              `json:"season_id"`
}

// GetSeasonStandingsResponsePayloadV1 contains the standings response.
type GetSeasonStandingsResponsePayloadV1 struct {
	GuildID   sharedtypes.GuildID    `json:"guild_id"`
	SeasonID  string                 `json:"season_id"`
	Standings []SeasonStandingItemV1 `json:"standings"`
}

// SeasonStandingItemV1 represents a single season standing entry.
type SeasonStandingItemV1 struct {
	MemberID      sharedtypes.DiscordID `json:"member_id"`
	TotalPoints   int                   `json:"total_points"`
	CurrentTier   string                `json:"current_tier"`
	SeasonBestTag int                   `json:"season_best_tag"`
	RoundsPlayed  int                   `json:"rounds_played"`
}

// FailedPayloadV1 is a generic failure payload for admin operations.
type FailedPayloadV1 struct {
	GuildID sharedtypes.GuildID `json:"guild_id"`
	Reason  string              `json:"reason"`
}
