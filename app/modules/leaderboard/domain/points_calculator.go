package leaderboarddomain

import (
	"cmp"
	"slices"
)

// RoundParticipant holds the data needed for points calculation after tag allocation.
type RoundParticipant struct {
	MemberID     string
	TagNumber    int // tag after allocation
	RoundsPlayed int
	BestTag      int // season-best tag (lowest achieved)
	CurrentTier  Tier
}

// PointAward represents the points earned by a single participant in a round.
type PointAward struct {
	MemberID        string
	Points          int
	OpponentsBeaten int
	Tier            Tier
}

// CalculateRoundPoints computes points for all participants using the opponents-defeated matrix.
//
// Participants are sorted by tag number ascending (best tag = rank 1).
// Each participant earns points for every participant they outrank (have a lower tag number).
// Tier bonuses apply per matchup according to CalculateMatchup rules.
func CalculateRoundPoints(participants []RoundParticipant) []PointAward {
	if len(participants) == 0 {
		return nil
	}

	// Sort by tag number ascending with deterministic tie-break on member ID
	sorted := make([]RoundParticipant, len(participants))
	copy(sorted, participants)

	slices.SortFunc(sorted, func(a, b RoundParticipant) int {
		if c := cmp.Compare(a.TagNumber, b.TagNumber); c != 0 {
			return c
		}
		return cmp.Compare(a.MemberID, b.MemberID)
	})

	awards := make([]PointAward, len(sorted))

	for i := 0; i < len(sorted); i++ {
		winner := PlayerContext{
			ID:           sorted[i].MemberID,
			RoundsPlayed: sorted[i].RoundsPlayed,
			BestTag:      sorted[i].BestTag,
			CurrentTier:  sorted[i].CurrentTier,
		}

		totalPoints := 0
		opponentsBeaten := 0

		// Winner beats everyone ranked below them
		for j := i + 1; j < len(sorted); j++ {
			loser := PlayerContext{
				ID:           sorted[j].MemberID,
				RoundsPlayed: sorted[j].RoundsPlayed,
				BestTag:      sorted[j].BestTag,
				CurrentTier:  sorted[j].CurrentTier,
			}

			matchPoints := CalculateMatchup(winner, loser)
			totalPoints += int(matchPoints)
			opponentsBeaten++
		}

		awards[i] = PointAward{
			MemberID:        sorted[i].MemberID,
			Points:          totalPoints,
			OpponentsBeaten: opponentsBeaten,
			Tier:            sorted[i].CurrentTier,
		}
	}

	return awards
}

// UpdateBestTag returns the better (lower) of the current best and the new tag.
// A value of 0 means "no tag yet", so any positive tag beats it.
func UpdateBestTag(currentBest, newTag int) int {
	if currentBest <= 0 {
		return newTag
	}
	if newTag <= 0 {
		return currentBest
	}
	if newTag < currentBest {
		return newTag
	}
	return currentBest
}
