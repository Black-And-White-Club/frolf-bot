package scoreservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories" // Adjust this path to your project structure
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Score Repo
// ------------------------

// FakeScoreRepository provides a programmable stub for the scoredb.Repository interface.
type FakeScoreRepository struct {
	trace []string

	LogScoresFunc         func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error
	UpdateScoreFunc       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error
	UpdateOrAddScoreFunc  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error
	GetScoresForRoundFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error)
	LastLoggedScores      []sharedtypes.ScoreInfo
}

// Trace returns the sequence of method calls made to the fake.
func (f *FakeScoreRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// NewFakeScoreRepository initializes a new FakeScoreRepository with an empty trace.
func NewFakeScoreRepository() *FakeScoreRepository {
	return &FakeScoreRepository{
		trace: []string{},
	}
}

func (f *FakeScoreRepository) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Repository Interface Implementation ---

func (f *FakeScoreRepository) LogScores(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
	f.record("LogScores")
	f.LastLoggedScores = scores
	if f.LogScoresFunc != nil {
		return f.LogScoresFunc(ctx, db, guildID, roundID, scores, source)
	}
	return nil
}

func (f *FakeScoreRepository) UpdateScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error {
	f.record("UpdateScore")
	if f.UpdateScoreFunc != nil {
		return f.UpdateScoreFunc(ctx, db, guildID, roundID, userID, newScore)
	}
	return nil
}

func (f *FakeScoreRepository) UpdateOrAddScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error {
	f.record("UpdateOrAddScore")
	if f.UpdateOrAddScoreFunc != nil {
		return f.UpdateOrAddScoreFunc(ctx, db, guildID, roundID, scoreInfo)
	}
	return nil
}

func (f *FakeScoreRepository) GetScoresForRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	f.record("GetScoresForRound")
	if f.GetScoresForRoundFunc != nil {
		return f.GetScoresForRoundFunc(ctx, db, guildID, roundID)
	}
	// Default: Return empty slice, which your service treats as "no scores exist"
	return []sharedtypes.ScoreInfo{}, nil
}

// Ensure the fake actually satisfies the interface
// Note: Replace 'scoredb' with your actual repository package name if different
var _ scoredb.Repository = (*FakeScoreRepository)(nil)
