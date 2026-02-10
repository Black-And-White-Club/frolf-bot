package leaderboarddomain

import (
	"math"
)

// Points uses a custom type to prevent floating-point errors.
type Points int

// Tier represents a player's skill tier.
type Tier string

const (
	TierGold   Tier = "Gold"
	TierSilver Tier = "Silver"
	TierBronze Tier = "Bronze"

	BaseWin          Points = 100
	BonusStandard    Points = 50
	BonusGiantSlayer Points = 75
)

// PlayerContext contains the necessary information for a player to calculate ratings.
type PlayerContext struct {
	ID           string
	RoundsPlayed int
	BestTag      int // 0 means no tag or unranked
	CurrentTier  Tier
}

// CalculateMatchup calculates the points earned by the winner of a matchup.
// It returns winnerPoints.
func CalculateMatchup(winner, loser PlayerContext) Points {
	// Base win points
	winnerPoints := BaseWin

	// Provisional players (played < 3 rounds) never earn bonuses.
	if winner.RoundsPlayed < 3 {
		return winnerPoints
	}

	// Gold tier players never earn bonuses.
	if winner.CurrentTier == TierGold {
		return winnerPoints
	}

	// Calculate bonuses based on tier gap if lower rank beats higher rank.
	// Rank order: Gold > Silver > Bronze.

	// Silver beats Gold -> Standard Bonus
	if winner.CurrentTier == TierSilver && loser.CurrentTier == TierGold {
		winnerPoints += BonusStandard
	}

	// Bronze beats Silver -> Standard Bonus
	if winner.CurrentTier == TierBronze && loser.CurrentTier == TierSilver {
		winnerPoints += BonusStandard
	}

	// Bronze beats Gold -> Giant Slayer Bonus
	if winner.CurrentTier == TierBronze && loser.CurrentTier == TierGold {
		winnerPoints += BonusGiantSlayer
	}

	return winnerPoints
}

// DetermineTier calculates the tier based on the best tag and total members.
// Top 10% = Gold, Next 30% = Silver, Rest = Bronze.
func DetermineTier(bestTag int, totalMembers int) Tier {
	if totalMembers <= 0 || bestTag <= 0 {
		return TierBronze
	}

	// Calculate rank thresholds
	// Gold threshold: 10%
	goldCount := int(math.Ceil(float64(totalMembers) * 0.10))
	if bestTag <= goldCount {
		return TierGold
	}

	// Silver threshold: Next 30% (so cumulative 40%)
	silverCount := int(math.Ceil(float64(totalMembers) * 0.40))
	if bestTag <= silverCount {
		return TierSilver
	}

	return TierBronze
}
