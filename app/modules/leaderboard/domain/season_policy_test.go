package leaderboarddomain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveSeasonForRound(t *testing.T) {
	activeSeason := &SeasonState{
		SeasonID: "season-1",
		IsActive: true,
	}
	inactiveSeason := &SeasonState{
		SeasonID: "season-1",
		IsActive: false,
	}

	tests := []struct {
		name             string
		rollbackSeasonID string
		activeSeason     *SeasonState
		expectedID       string
		expectedActive   bool
	}{
		{
			name:             "Explicit rollback season ID takes precedence",
			rollbackSeasonID: "season-old",
			activeSeason:     activeSeason,
			expectedID:       "season-old",
			expectedActive:   true,
		},
		{
			name:             "Use active season if no rollback ID",
			rollbackSeasonID: "",
			activeSeason:     activeSeason,
			expectedID:       "season-1",
			expectedActive:   true,
		},
		{
			name:             "No active season returns empty info",
			rollbackSeasonID: "",
			activeSeason:     nil,
			expectedID:       "",
			expectedActive:   false,
		},
		{
			name:             "Inactive season returns empty info",
			rollbackSeasonID: "",
			activeSeason:     inactiveSeason,
			expectedID:       "",
			expectedActive:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveSeasonForRound(tt.rollbackSeasonID, tt.activeSeason)
			assert.Equal(t, tt.expectedID, result.SeasonID)
			assert.Equal(t, tt.expectedActive, result.IsActive)
		})
	}
}

func TestShouldAwardPoints(t *testing.T) {
	tests := []struct {
		name     string
		season   ResolvedSeason
		expected bool
	}{
		{
			name:     "Should award points if season ID is present",
			season:   ResolvedSeason{SeasonID: "s1", IsActive: true},
			expected: true,
		},
		{
			name:     "Should not award points if season ID is empty",
			season:   ResolvedSeason{SeasonID: "", IsActive: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ShouldAwardPoints(tt.season))
		})
	}
}

func TestValidateSeasonStart(t *testing.T) {
	tests := []struct {
		name       string
		seasonID   string
		seasonName string
		wantError  bool
	}{
		{
			name:       "Valid input",
			seasonID:   "s1",
			seasonName: "Spring 2024",
			wantError:  false,
		},
		{
			name:       "Missing season ID",
			seasonID:   "",
			seasonName: "Spring 2024",
			wantError:  true,
		},
		{
			name:       "Missing season name",
			seasonID:   "s1",
			seasonName: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ValidateSeasonStart(tt.seasonID, tt.seasonName)
			if tt.wantError {
				assert.NotEmpty(t, msg)
			} else {
				assert.Empty(t, msg)
			}
		})
	}
}
