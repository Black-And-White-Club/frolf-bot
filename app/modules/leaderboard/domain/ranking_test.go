package leaderboarddomain

import (
	"testing"
)

func TestCalculateMatchup(t *testing.T) {
	tests := []struct {
		name       string
		winner     PlayerContext
		loser      PlayerContext
		wantWinner Points
	}{
		{
			name: "Provisional player wins (no bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 2,
				CurrentTier:  TierBronze,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierGold,
			},
			wantWinner: BaseWin,
		},
		{
			name: "Gold player wins (no bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 10,
				CurrentTier:  TierGold,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierBronze,
			},
			wantWinner: BaseWin,
		},
		{
			name: "Same tier match (no bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 10,
				CurrentTier:  TierSilver,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierSilver,
			},
			wantWinner: BaseWin,
		},
		{
			name: "Bronze beats Silver (Standard Bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 10,
				CurrentTier:  TierBronze,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierSilver,
			},
			wantWinner: BaseWin + BonusStandard,
		},
		{
			name: "Silver beats Gold (Standard Bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 10,
				CurrentTier:  TierSilver,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierGold,
			},
			wantWinner: BaseWin + BonusStandard,
		},
		{
			name: "Bronze beats Gold (Giant Slayer Bonus)",
			winner: PlayerContext{
				ID:           "winner",
				RoundsPlayed: 10,
				CurrentTier:  TierBronze,
			},
			loser: PlayerContext{
				ID:           "loser",
				RoundsPlayed: 10,
				CurrentTier:  TierGold,
			},
			wantWinner: BaseWin + BonusGiantSlayer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWinner := CalculateMatchup(tt.winner, tt.loser)
			if gotWinner != tt.wantWinner {
				t.Errorf("CalculateMatchup() winner = %v, want %v", gotWinner, tt.wantWinner)
			}
		})
	}
}

func TestDetermineTier(t *testing.T) {
	tests := []struct {
		name         string
		bestTag      int
		totalMembers int
		want         Tier
	}{
		{
			name:         "Empty league",
			bestTag:      0,
			totalMembers: 0,
			want:         TierBronze,
		},
		{
			name:         "Small league (10 members) - Rank 1 is Gold",
			bestTag:      1,
			totalMembers: 10,
			want:         TierGold,
		},
		{
			name:         "Small league (10 members) - Rank 2 is Silver",
			bestTag:      2,
			totalMembers: 10,
			want:         TierSilver,
		},
		{
			name:         "Small league (10 members) - Rank 4 is Silver",
			bestTag:      4,
			totalMembers: 10,
			want:         TierSilver,
		},
		{
			name:         "Small league (10 members) - Rank 5 is Bronze",
			bestTag:      5,
			totalMembers: 10,
			want:         TierBronze,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineTier(tt.bestTag, tt.totalMembers); got != tt.want {
				t.Errorf("DetermineTier() = %v, want %v", got, tt.want)
			}
		})
	}
}
