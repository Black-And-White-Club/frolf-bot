package bettinghandlers

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// FakeBettingService
// ---------------------------------------------------------------------------

type FakeBettingService struct {
	trace []string

	GetOverviewFunc               func(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.Overview, error)
	GetNextRoundMarketFunc        func(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.NextRoundMarket, error)
	GetAdminMarketsFunc           func(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.AdminMarketBoard, error)
	UpdateSettingsFunc            func(ctx context.Context, req bettingservice.UpdateSettingsRequest) (*bettingservice.MemberSettings, error)
	AdjustWalletFunc              func(ctx context.Context, req bettingservice.AdjustWalletRequest) (*bettingservice.WalletJournal, error)
	PlaceBetFunc                  func(ctx context.Context, req bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error)
	AdminMarketActionFunc         func(ctx context.Context, req bettingservice.AdminMarketActionRequest) (*bettingservice.AdminMarketActionResult, error)
	SettleRoundFunc               func(ctx context.Context, guildID sharedtypes.GuildID, round *bettingservice.BettingSettlementRound, source string, actorUUID *uuid.UUID, reason string) ([]bettingservice.MarketSettlementResult, error)
	VoidRoundMarketsFunc          func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, source string, actorUUID *uuid.UUID, reason string) ([]bettingservice.MarketVoidResult, error)
	EnsureMarketsForGuildFunc     func(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketGeneratedResult, error)
	LockDueMarketsFunc            func(ctx context.Context) ([]bettingservice.MarketLockResult, error)
	SuspendOpenMarketsForClubFunc func(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error)
	GetMarketSnapshotFunc         func(ctx context.Context, clubUUID uuid.UUID) (*bettingservice.MarketSnapshot, error)
	MirrorPointsToWalletFunc      func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, points map[sharedtypes.DiscordID]int) error
}

// compile-time check
var _ bettingservice.Service = (*FakeBettingService)(nil)

func (f *FakeBettingService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeBettingService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeBettingService) GetOverview(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.Overview, error) {
	f.record("GetOverview")
	if f.GetOverviewFunc != nil {
		return f.GetOverviewFunc(ctx, clubUUID, userUUID)
	}
	return nil, nil
}

func (f *FakeBettingService) GetNextRoundMarket(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.NextRoundMarket, error) {
	f.record("GetNextRoundMarket")
	if f.GetNextRoundMarketFunc != nil {
		return f.GetNextRoundMarketFunc(ctx, clubUUID, userUUID)
	}
	return nil, nil
}

func (f *FakeBettingService) GetAdminMarkets(ctx context.Context, clubUUID, userUUID uuid.UUID) (*bettingservice.AdminMarketBoard, error) {
	f.record("GetAdminMarkets")
	if f.GetAdminMarketsFunc != nil {
		return f.GetAdminMarketsFunc(ctx, clubUUID, userUUID)
	}
	return nil, nil
}

func (f *FakeBettingService) UpdateSettings(ctx context.Context, req bettingservice.UpdateSettingsRequest) (*bettingservice.MemberSettings, error) {
	f.record("UpdateSettings")
	if f.UpdateSettingsFunc != nil {
		return f.UpdateSettingsFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeBettingService) AdjustWallet(ctx context.Context, req bettingservice.AdjustWalletRequest) (*bettingservice.WalletJournal, error) {
	f.record("AdjustWallet")
	if f.AdjustWalletFunc != nil {
		return f.AdjustWalletFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeBettingService) PlaceBet(ctx context.Context, req bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
	f.record("PlaceBet")
	if f.PlaceBetFunc != nil {
		return f.PlaceBetFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeBettingService) AdminMarketAction(ctx context.Context, req bettingservice.AdminMarketActionRequest) (*bettingservice.AdminMarketActionResult, error) {
	f.record("AdminMarketAction")
	if f.AdminMarketActionFunc != nil {
		return f.AdminMarketActionFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeBettingService) SettleRound(ctx context.Context, guildID sharedtypes.GuildID, round *bettingservice.BettingSettlementRound, source string, actorUUID *uuid.UUID, reason string) ([]bettingservice.MarketSettlementResult, error) {
	f.record("SettleRound")
	if f.SettleRoundFunc != nil {
		return f.SettleRoundFunc(ctx, guildID, round, source, actorUUID, reason)
	}
	return nil, nil
}

func (f *FakeBettingService) VoidRoundMarkets(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, source string, actorUUID *uuid.UUID, reason string) ([]bettingservice.MarketVoidResult, error) {
	f.record("VoidRoundMarkets")
	if f.VoidRoundMarketsFunc != nil {
		return f.VoidRoundMarketsFunc(ctx, guildID, roundID, source, actorUUID, reason)
	}
	return nil, nil
}

func (f *FakeBettingService) EnsureMarketsForGuild(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketGeneratedResult, error) {
	f.record("EnsureMarketsForGuild")
	if f.EnsureMarketsForGuildFunc != nil {
		return f.EnsureMarketsForGuildFunc(ctx, guildID)
	}
	return nil, nil
}

func (f *FakeBettingService) LockDueMarkets(ctx context.Context) ([]bettingservice.MarketLockResult, error) {
	f.record("LockDueMarkets")
	if f.LockDueMarketsFunc != nil {
		return f.LockDueMarketsFunc(ctx)
	}
	return nil, nil
}

func (f *FakeBettingService) SuspendOpenMarketsForClub(ctx context.Context, guildID sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error) {
	f.record("SuspendOpenMarketsForClub")
	if f.SuspendOpenMarketsForClubFunc != nil {
		return f.SuspendOpenMarketsForClubFunc(ctx, guildID)
	}
	return nil, nil
}

func (f *FakeBettingService) GetMarketSnapshot(ctx context.Context, clubUUID uuid.UUID) (*bettingservice.MarketSnapshot, error) {
	f.record("GetMarketSnapshot")
	if f.GetMarketSnapshotFunc != nil {
		return f.GetMarketSnapshotFunc(ctx, clubUUID)
	}
	return nil, nil
}

func (f *FakeBettingService) MirrorPointsToWallet(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, points map[sharedtypes.DiscordID]int) error {
	f.record("MirrorPointsToWallet")
	if f.MirrorPointsToWalletFunc != nil {
		return f.MirrorPointsToWalletFunc(ctx, guildID, roundID, points)
	}
	return nil
}
