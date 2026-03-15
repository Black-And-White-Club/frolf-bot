package guilddb

import (
	"testing"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
)

func TestPickBestAccess(t *testing.T) {
	t.Parallel()

	now := time.Now()
	expired := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	enabled := func(expiry *time.Time) *guildtypes.ClubFeatureAccess {
		return &guildtypes.ClubFeatureAccess{
			Key:       guildtypes.ClubFeatureBetting,
			State:     guildtypes.FeatureAccessStateEnabled,
			ExpiresAt: expiry,
		}
	}
	frozen := func(expiry *time.Time) *guildtypes.ClubFeatureAccess {
		return &guildtypes.ClubFeatureAccess{
			Key:       guildtypes.ClubFeatureBetting,
			State:     guildtypes.FeatureAccessStateFrozen,
			ExpiresAt: expiry,
		}
	}
	disabled := func() *guildtypes.ClubFeatureAccess {
		return &guildtypes.ClubFeatureAccess{
			Key:   guildtypes.ClubFeatureBetting,
			State: guildtypes.FeatureAccessStateDisabled,
		}
	}

	tests := []struct {
		name      string
		a, b      *guildtypes.ClubFeatureAccess
		wantState guildtypes.FeatureAccessState
	}{
		{
			name:      "trial active + sub active => enabled",
			a:         enabled(&future),
			b:         enabled(&future),
			wantState: guildtypes.FeatureAccessStateEnabled,
		},
		{
			name:      "trial expired + sub active => enabled (subscription wins)",
			a:         frozen(&expired),
			b:         enabled(&future),
			wantState: guildtypes.FeatureAccessStateEnabled,
		},
		{
			name:      "trial active + sub expired => enabled (trial wins)",
			a:         enabled(&future),
			b:         frozen(&expired),
			wantState: guildtypes.FeatureAccessStateEnabled,
		},
		{
			name:      "trial expired + sub expired => frozen (latest expiry wins, state is frozen)",
			a:         frozen(&expired),
			b:         frozen(&expired),
			wantState: guildtypes.FeatureAccessStateFrozen,
		},
		{
			name:      "neither trial nor sub => disabled",
			a:         disabled(),
			b:         disabled(),
			wantState: guildtypes.FeatureAccessStateDisabled,
		},
		{
			name:      "one nil => returns non-nil",
			a:         nil,
			b:         enabled(&future),
			wantState: guildtypes.FeatureAccessStateEnabled,
		},
		{
			name:      "both nil => returns nil (caller defaults to disabled)",
			a:         nil,
			b:         nil,
			wantState: "", // handled by nil check in caller
		},
		{
			name:      "enabled no expiry beats enabled with expiry",
			a:         enabled(nil),
			b:         enabled(&future),
			wantState: guildtypes.FeatureAccessStateEnabled,
		},
		{
			name:      "frozen no expiry beats frozen with expiry",
			a:         frozen(nil),
			b:         frozen(&future),
			wantState: guildtypes.FeatureAccessStateFrozen,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pickBestAccess(tt.a, tt.b)
			if tt.a == nil && tt.b == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.State != tt.wantState {
				t.Errorf("state: want %s, got %s", tt.wantState, got.State)
			}
		})
	}
}
