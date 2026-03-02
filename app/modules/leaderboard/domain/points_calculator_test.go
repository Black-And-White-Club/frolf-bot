package leaderboarddomain

import "testing"

func TestCalculateRoundPoints(t *testing.T) {
	participants := []RoundParticipant{
		{MemberID: "alice", TagNumber: 1, RoundsPlayed: 10, BestTag: 1, CurrentTier: TierGold},
		{MemberID: "bob", TagNumber: 2, RoundsPlayed: 10, BestTag: 2, CurrentTier: TierSilver},
		{MemberID: "charlie", TagNumber: 3, RoundsPlayed: 10, BestTag: 3, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 3 {
		t.Fatalf("expected 3 awards, got %d", len(awards))
	}

	if awards[0].MemberID != "alice" || awards[0].Points != 200 || awards[0].OpponentsBeaten != 2 {
		t.Fatalf("unexpected award[0]: %+v", awards[0])
	}
	if awards[1].MemberID != "bob" || awards[1].Points != 100 || awards[1].OpponentsBeaten != 1 {
		t.Fatalf("unexpected award[1]: %+v", awards[1])
	}
	if awards[2].MemberID != "charlie" || awards[2].Points != 0 || awards[2].OpponentsBeaten != 0 {
		t.Fatalf("unexpected award[2]: %+v", awards[2])
	}
}

func TestCalculateRoundPointsDeterministicTieBreak(t *testing.T) {
	participants := []RoundParticipant{
		{MemberID: "z-user", TagNumber: 1, RoundsPlayed: 10, BestTag: 1, CurrentTier: TierBronze},
		{MemberID: "a-user", TagNumber: 1, RoundsPlayed: 10, BestTag: 1, CurrentTier: TierBronze},
		{MemberID: "b-user", TagNumber: 2, RoundsPlayed: 10, BestTag: 2, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 3 {
		t.Fatalf("expected 3 awards, got %d", len(awards))
	}

	// Same tag ties are resolved by member ID.
	if awards[0].MemberID != "a-user" {
		t.Fatalf("expected first winner to be a-user, got %s", awards[0].MemberID)
	}
}

func TestCalculateRoundPointsUntaggedNotCountedAsOpponent(t *testing.T) {
	participants := []RoundParticipant{
		{MemberID: "untagged", TagNumber: 0, RoundsPlayed: 5, BestTag: 0, CurrentTier: TierBronze},
		{MemberID: "tagged", TagNumber: 10, RoundsPlayed: 5, BestTag: 10, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 1 {
		t.Fatalf("expected 1 award, got %d", len(awards))
	}

	// Tagged player sorts first, but earns 0 points: untagged players are not counted as opponents.
	if awards[0].MemberID != "tagged" {
		t.Errorf("expected tagged player to be first, got %s", awards[0].MemberID)
	}
	if awards[0].Points != 0 {
		t.Errorf("expected tagged player to earn 0 points (no tagged opponents), got %d", awards[0].Points)
	}
	if awards[0].OpponentsBeaten != 0 {
		t.Errorf("expected 0 opponents beaten, got %d", awards[0].OpponentsBeaten)
	}
}

// TestCalculateRoundPoints_TiedFinishRank verifies the core tie-handling property:
// two players sharing a FinishRank are not counted as each other's opponents. With
// identical matchup modifier context, they end up with equal points.
func TestCalculateRoundPoints_TiedFinishRank(t *testing.T) {
	// alice (tag 1) and bob (tag 2) tied at rank 1; carol (tag 3) finishes 3rd.
	// alice and bob should each beat only carol, not each other.
	participants := []RoundParticipant{
		{MemberID: "alice", TagNumber: 1, FinishRank: 1, RoundsPlayed: 10, BestTag: 1, CurrentTier: TierBronze},
		{MemberID: "bob", TagNumber: 2, FinishRank: 1, RoundsPlayed: 10, BestTag: 2, CurrentTier: TierBronze},
		{MemberID: "carol", TagNumber: 3, FinishRank: 3, RoundsPlayed: 10, BestTag: 3, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 3 {
		t.Fatalf("expected 3 awards, got %d", len(awards))
	}

	byMember := make(map[string]PointAward, len(awards))
	for _, a := range awards {
		byMember[a.MemberID] = a
	}

	// Both tied players should beat only carol (1 opponent each)
	if byMember["alice"].OpponentsBeaten != 1 {
		t.Errorf("alice: expected 1 opponent beaten, got %d", byMember["alice"].OpponentsBeaten)
	}
	if byMember["bob"].OpponentsBeaten != 1 {
		t.Errorf("bob: expected 1 opponent beaten, got %d", byMember["bob"].OpponentsBeaten)
	}
	// Same tie rank + same bonus context -> equal points, and both must be > 0.
	// The > 0 guard ensures a regression in CalculateMatchup returning 0 doesn't
	// silently pass (alice.Points == bob.Points == 0 would satisfy equality alone).
	if byMember["alice"].Points != byMember["bob"].Points {
		t.Errorf("tied players should have equal points: alice=%d bob=%d",
			byMember["alice"].Points, byMember["bob"].Points)
	}
	if byMember["alice"].Points <= 0 {
		t.Errorf("tied players should earn positive points, got alice=%d", byMember["alice"].Points)
	}
	// carol is last: no opponents beaten
	if byMember["carol"].OpponentsBeaten != 0 {
		t.Errorf("carol: expected 0 opponents beaten, got %d", byMember["carol"].OpponentsBeaten)
	}
	if byMember["carol"].Points != 0 {
		t.Errorf("carol: expected 0 points, got %d", byMember["carol"].Points)
	}
}

// TestCalculateRoundPoints_LargeFieldWithTiedGroup verifies tie handling in a
// realistic field: 5 players where two are tied for 2nd, ensuring the tied pair
// skips each other but still beat the two players behind them.
func TestCalculateRoundPoints_LargeFieldWithTiedGroup(t *testing.T) {
	// rank 1: alice (tag 1)
	// rank 2: bob (tag 2), carol (tag 3) — tied
	// rank 4: dan (tag 4), eve (tag 5)
	participants := []RoundParticipant{
		{MemberID: "alice", TagNumber: 1, FinishRank: 1, RoundsPlayed: 5, BestTag: 1, CurrentTier: TierBronze},
		{MemberID: "bob", TagNumber: 2, FinishRank: 2, RoundsPlayed: 5, BestTag: 2, CurrentTier: TierBronze},
		{MemberID: "carol", TagNumber: 3, FinishRank: 2, RoundsPlayed: 5, BestTag: 3, CurrentTier: TierBronze},
		{MemberID: "dan", TagNumber: 4, FinishRank: 4, RoundsPlayed: 5, BestTag: 4, CurrentTier: TierBronze},
		{MemberID: "eve", TagNumber: 5, FinishRank: 4, RoundsPlayed: 5, BestTag: 5, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 5 {
		t.Fatalf("expected 5 awards, got %d", len(awards))
	}

	byMember := make(map[string]PointAward, len(awards))
	for _, a := range awards {
		byMember[a.MemberID] = a
	}

	// alice beats bob, carol, dan, eve → 4 opponents
	if byMember["alice"].OpponentsBeaten != 4 {
		t.Errorf("alice: expected 4 opponents beaten, got %d", byMember["alice"].OpponentsBeaten)
	}
	// bob and carol are tied — each beats dan and eve (2 opponents), not each other
	if byMember["bob"].OpponentsBeaten != 2 {
		t.Errorf("bob: expected 2 opponents beaten, got %d", byMember["bob"].OpponentsBeaten)
	}
	if byMember["carol"].OpponentsBeaten != 2 {
		t.Errorf("carol: expected 2 opponents beaten, got %d", byMember["carol"].OpponentsBeaten)
	}
	if byMember["bob"].Points != byMember["carol"].Points {
		t.Errorf("same-context tied players should have equal points: bob=%d carol=%d",
			byMember["bob"].Points, byMember["carol"].Points)
	}
	if byMember["bob"].Points <= 0 {
		t.Errorf("tied players should earn positive points, got bob=%d", byMember["bob"].Points)
	}
	// dan and eve are tied at rank 4 — neither beats the other, beat nobody below
	if byMember["dan"].OpponentsBeaten != 0 {
		t.Errorf("dan: expected 0 opponents beaten, got %d", byMember["dan"].OpponentsBeaten)
	}
	if byMember["eve"].OpponentsBeaten != 0 {
		t.Errorf("eve: expected 0 opponents beaten, got %d", byMember["eve"].OpponentsBeaten)
	}
}

// TestCalculateRoundPoints_TiedFinishRank_MixedTierRewardsLowerTier verifies that
// tied finishers can earn different points when tier modifiers differ.
func TestCalculateRoundPoints_TiedFinishRank_MixedTierRewardsLowerTier(t *testing.T) {
	participants := []RoundParticipant{
		// Tied for 1st.
		{MemberID: "gold", TagNumber: 1, FinishRank: 1, RoundsPlayed: 10, BestTag: 1, CurrentTier: TierGold},
		{MemberID: "bronze", TagNumber: 2, FinishRank: 1, RoundsPlayed: 10, BestTag: 2, CurrentTier: TierBronze},
		// Shared lower finisher both tied leaders beat.
		{MemberID: "silver", TagNumber: 3, FinishRank: 3, RoundsPlayed: 10, BestTag: 3, CurrentTier: TierSilver},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 3 {
		t.Fatalf("expected 3 awards, got %d", len(awards))
	}

	byMember := make(map[string]PointAward, len(awards))
	for _, a := range awards {
		byMember[a.MemberID] = a
	}

	if byMember["gold"].OpponentsBeaten != 1 {
		t.Errorf("gold: expected 1 opponent beaten, got %d", byMember["gold"].OpponentsBeaten)
	}
	if byMember["bronze"].OpponentsBeaten != 1 {
		t.Errorf("bronze: expected 1 opponent beaten, got %d", byMember["bronze"].OpponentsBeaten)
	}

	// Bronze gets an upset bonus vs Silver; Gold gets base points only.
	if byMember["bronze"].Points <= byMember["gold"].Points {
		t.Errorf("expected bronze tied finisher to outscore gold tied finisher: bronze=%d gold=%d",
			byMember["bronze"].Points, byMember["gold"].Points)
	}
}

// TestCalculateRoundPoints_ZeroFinishRank_LegacyBehavior checks that when
// FinishRank is 0 (not set by caller), players are not skipped as opponents,
// preserving backwards-compatible behaviour for legacy callers.
func TestCalculateRoundPoints_ZeroFinishRank_LegacyBehavior(t *testing.T) {
	// All FinishRanks zero: falls back to sequential opponent-counting.
	participants := []RoundParticipant{
		{MemberID: "alice", TagNumber: 1, FinishRank: 0, RoundsPlayed: 5, BestTag: 1, CurrentTier: TierBronze},
		{MemberID: "bob", TagNumber: 2, FinishRank: 0, RoundsPlayed: 5, BestTag: 2, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)
	if len(awards) != 2 {
		t.Fatalf("expected 2 awards, got %d", len(awards))
	}

	byMember := make(map[string]PointAward, len(awards))
	for _, a := range awards {
		byMember[a.MemberID] = a
	}

	// With FinishRank=0, alice beats bob (the skip-same-rank guard requires both > 0)
	if byMember["alice"].OpponentsBeaten != 1 {
		t.Errorf("alice: expected 1 opponent beaten (legacy path), got %d", byMember["alice"].OpponentsBeaten)
	}
	if byMember["bob"].OpponentsBeaten != 0 {
		t.Errorf("bob: expected 0 opponents beaten, got %d", byMember["bob"].OpponentsBeaten)
	}
}

func TestUpdateBestTag(t *testing.T) {
	tests := []struct {
		name        string
		currentBest int
		newTag      int
		want        int
	}{
		{name: "initial best", currentBest: 0, newTag: 4, want: 4},
		{name: "ignore zero new tag", currentBest: 2, newTag: 0, want: 2},
		{name: "improved best", currentBest: 4, newTag: 2, want: 2},
		{name: "worse tag", currentBest: 2, newTag: 4, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateBestTag(tt.currentBest, tt.newTag)
			if got != tt.want {
				t.Fatalf("UpdateBestTag() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestCalculateRoundPoints_SingleParticipant verifies that a solo round with an
// explicit FinishRank produces a valid (zero-point) award entry and does not panic
// or skip the participant.
func TestCalculateRoundPoints_SingleParticipant(t *testing.T) {
	participants := []RoundParticipant{
		{MemberID: "solo", TagNumber: 7, FinishRank: 1, RoundsPlayed: 5, BestTag: 7, CurrentTier: TierBronze},
	}

	awards := CalculateRoundPoints(participants)

	if len(awards) != 1 {
		t.Fatalf("expected 1 award for single participant, got %d", len(awards))
	}
	if awards[0].MemberID != "solo" {
		t.Errorf("expected award for 'solo', got %q", awards[0].MemberID)
	}
	if awards[0].Points != 0 {
		t.Errorf("solo participant with no opponents should earn 0 points, got %d", awards[0].Points)
	}
	if awards[0].OpponentsBeaten != 0 {
		t.Errorf("solo participant should have 0 opponents beaten, got %d", awards[0].OpponentsBeaten)
	}
}
