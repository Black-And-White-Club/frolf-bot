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
