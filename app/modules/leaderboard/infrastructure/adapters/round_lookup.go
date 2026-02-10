package adapters

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// RoundLookupAdapter adapts the round repository to the leaderboard handler's RoundLookup port.
type RoundLookupAdapter struct {
	roundRepo rounddb.Repository
	db        bun.IDB
}

func NewRoundLookupAdapter(roundRepo rounddb.Repository, db bun.IDB) *RoundLookupAdapter {
	return &RoundLookupAdapter{roundRepo: roundRepo, db: db}
}

func (a *RoundLookupAdapter) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	return a.roundRepo.GetRound(ctx, a.db, guildID, roundID)
}
