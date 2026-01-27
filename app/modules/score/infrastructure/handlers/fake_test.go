package scorehandlers

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
)

// ------------------------
// Fake Score Service
// ------------------------

// FakeScoreService provides a programmable stub for the scoreservice.Service interface.
// It allows you to inject custom behavior for each method and track calls via Trace.
type FakeScoreService struct {
	trace []string

	ProcessRoundScoresFunc      func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error)
	CorrectScoreFunc            func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, score sharedtypes.Score, tagNumber *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error)
	ProcessScoresForStorageFunc func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) ([]sharedtypes.ScoreInfo, error)
	GetScoresForRoundFunc       func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error)
}

// NewFakeScoreService initializes a new FakeScoreService.
func NewFakeScoreService() *FakeScoreService {
	return &FakeScoreService{
		trace: []string{},
	}
}

func (f *FakeScoreService) record(step string) {
	f.trace = append(f.trace, step)
}

// Trace returns the sequence of service methods called.
func (f *FakeScoreService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// --- Service Interface Implementation ---

func (f *FakeScoreService) ProcessRoundScores(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
	f.record("ProcessRoundScores")
	if f.ProcessRoundScoresFunc != nil {
		return f.ProcessRoundScoresFunc(ctx, guildID, roundID, scores, overwrite)
	}
	return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{}, nil
}

func (f *FakeScoreService) CorrectScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, score sharedtypes.Score, tagNumber *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
	f.record("CorrectScore")
	if f.CorrectScoreFunc != nil {
		return f.CorrectScoreFunc(ctx, guildID, roundID, userID, score, tagNumber)
	}
	return scoreservice.ScoreOperationResult{}, nil
}

func (f *FakeScoreService) ProcessScoresForStorage(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) ([]sharedtypes.ScoreInfo, error) {
	f.record("ProcessScoresForStorage")
	if f.ProcessScoresForStorageFunc != nil {
		return f.ProcessScoresForStorageFunc(ctx, guildID, roundID, scores)
	}
	return nil, nil
}

func (f *FakeScoreService) GetScoresForRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	f.record("GetScoresForRound")
	if f.GetScoresForRoundFunc != nil {
		return f.GetScoresForRoundFunc(ctx, guildID, roundID)
	}
	return nil, nil
}

// Ensure the fake satisfies the Service interface
var _ scoreservice.Service = (*FakeScoreService)(nil)
