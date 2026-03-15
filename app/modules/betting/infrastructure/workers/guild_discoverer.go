package bettingworkers

import (
	"context"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// roundRepository is the subset of the round repository needed by the
// guild discoverer.
type roundRepository interface {
	GetAllUpcomingRoundsInWindow(ctx context.Context, db bun.IDB, lookahead time.Duration) ([]*roundtypes.Round, error)
}

// RoundRepoGuildDiscoverer implements guildDiscoverer by querying the round
// repository for upcoming rounds and deduplicating their guild IDs.
type RoundRepoGuildDiscoverer struct {
	repo roundRepository
}

func NewRoundRepoGuildDiscoverer(repo roundRepository) *RoundRepoGuildDiscoverer {
	return &RoundRepoGuildDiscoverer{repo: repo}
}

func (d *RoundRepoGuildDiscoverer) DiscoverGuildsWithUpcomingRounds(ctx context.Context, lookahead time.Duration) ([]sharedtypes.GuildID, error) {
	rounds, err := d.repo.GetAllUpcomingRoundsInWindow(ctx, nil, lookahead)
	if err != nil {
		return nil, err
	}

	seen := make(map[sharedtypes.GuildID]struct{}, len(rounds))
	out := make([]sharedtypes.GuildID, 0, len(rounds))
	for _, r := range rounds {
		if r.GuildID == "" {
			continue
		}
		if _, ok := seen[r.GuildID]; !ok {
			seen[r.GuildID] = struct{}{}
			out = append(out, r.GuildID)
		}
	}
	return out, nil
}
