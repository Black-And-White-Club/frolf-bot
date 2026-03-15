package bettingservice

import (
	"context"
	"errors"
	"testing"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// TestPlaceBet
// ---------------------------------------------------------------------------

func TestPlaceBet(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	userUUID := uuid.New()
	roundID := sharedtypes.RoundID(uuid.New())
	marketID := int64(99)
	futureStart := time.Now().Add(24 * time.Hour)

	openMarket := bettingdb.Market{
		ID:         marketID,
		ClubUUID:   clubUUID,
		SeasonID:   "2026-spring",
		RoundID:    roundID.UUID(),
		MarketType: winnerMarketType,
		Status:     openMarketStatus,
		LocksAt:    futureStart,
	}
	validOptions := []bettingdb.MarketOption{
		{MarketID: marketID, OptionKey: "player-a", Label: "Player A", ProbabilityBps: 5000, DecimalOddsCents: 200},
		{MarketID: marketID, OptionKey: "player-b", Label: "Player B", ProbabilityBps: 5000, DecimalOddsCents: 200},
	}

	// minimal round stubs for eligibility resolution
	twoPlayerRound := &roundtypes.Round{
		ID:      roundID,
		GuildID: "guild-1",
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "player-a", Response: roundtypes.ResponseAccept},
			{UserID: "player-b", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	baseSetup := func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
		userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
			return memberMembership(userUUID, clubUUID), nil
		}
		userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
			return "guild-1", nil
		}
		guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
			return enabledEntitlements(), nil
		}
		lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
			return &leaderboarddb.Season{ID: "2026-spring"}, nil
		}
		roundRepo.GetUpcomingRoundsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) ([]*roundtypes.Round, error) {
			return []*roundtypes.Round{twoPlayerRound}, nil
		}
		roundRepo.GetRoundFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID, _ sharedtypes.RoundID) (*roundtypes.Round, error) {
			return twoPlayerRound, nil
		}
		userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, id sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		userRepo.GetUserByUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (*userdb.User, error) {
			discordID := sharedtypes.DiscordID("player-test")
			return &userdb.User{UserID: &discordID}, nil
		}
	}

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository, *FakeLeaderboardRepository, *FakeRoundRepository)
		verify func(t *testing.T, result *BetTicket, err error)
	}{
		{
			name: "happy path: creates bet and stake reserve entry",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				baseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo)
				// Season-point deltas are now in the wallet journal; provide wallet balance directly.
				repo.AcquireWalletBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (*bettingdb.WalletBalance, error) {
					return &bettingdb.WalletBalance{Balance: 1000, Reserved: 0}, nil
				}
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return &openMarket, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return validOptions, nil
				}
				repo.ListBetsForUserAndMarketFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ int64) ([]bettingdb.Bet, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, result *BetTicket, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil BetTicket")
				}
				if result.SelectionKey != "player-a" {
					t.Errorf("SelectionKey: want player-a, got %s", result.SelectionKey)
				}
				if result.Status != acceptedBetStatus {
					t.Errorf("Status: want %s, got %s", acceptedBetStatus, result.Status)
				}
			},
		},
		{
			name: "insufficient balance returns ErrInsufficientBalance",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				baseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo)
				// Wallet has 5 already reserved — available = 0 - 5 = -5 < stake(100).
				repo.AcquireWalletBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (*bettingdb.WalletBalance, error) {
					return &bettingdb.WalletBalance{Balance: 0, Reserved: 5}, nil
				}
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return &openMarket, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return validOptions, nil
				}
				repo.ListBetsForUserAndMarketFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ int64) ([]bettingdb.Bet, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, result *BetTicket, err error) {
				if !errors.Is(err, ErrInsufficientBalance) {
					t.Errorf("expected ErrInsufficientBalance, got %v", err)
				}
			},
		},
		{
			name: "invalid selection key returns ErrSelectionInvalid",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				baseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo)
				repo.AcquireWalletBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (*bettingdb.WalletBalance, error) {
					return &bettingdb.WalletBalance{Balance: 1000, Reserved: 0}, nil
				}
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return &openMarket, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return validOptions, nil
				}
				repo.ListBetsForUserAndMarketFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ int64) ([]bettingdb.Bet, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, result *BetTicket, err error) {
				if !errors.Is(err, ErrSelectionInvalid) {
					t.Errorf("expected ErrSelectionInvalid, got %v", err)
				}
			},
		},
		{
			name: "locked market returns ErrMarketLocked",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				baseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo)
				repo.AcquireWalletBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (*bettingdb.WalletBalance, error) {
					return &bettingdb.WalletBalance{Balance: 1000, Reserved: 0}, nil
				}
				pastTime := time.Now().Add(-1 * time.Hour)
				lockedMarket := bettingdb.Market{
					ID:         marketID,
					ClubUUID:   clubUUID,
					SeasonID:   "2026-spring",
					RoundID:    roundID.UUID(),
					MarketType: winnerMarketType,
					Status:     openMarketStatus,
					LocksAt:    pastTime,
				}
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return &lockedMarket, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return validOptions, nil
				}
				repo.ListBetsForUserAndMarketFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ int64) ([]bettingdb.Bet, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, result *BetTicket, err error) {
				if !errors.Is(err, ErrMarketLocked) {
					t.Errorf("expected ErrMarketLocked, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			guildRepo := NewFakeGuildRepository()
			lbRepo := NewFakeLeaderboardRepository()
			roundRepo := NewFakeRoundRepository()
			tt.setup(repo, userRepo, guildRepo, lbRepo, roundRepo)
			svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
			selectionKey := "player-a"
			if tt.name == "invalid selection key returns ErrSelectionInvalid" {
				selectionKey = "nonexistent-player"
			}
			result, err := svc.PlaceBet(context.Background(), PlaceBetRequest{
				ClubUUID:     clubUUID,
				UserUUID:     userUUID,
				RoundID:      roundID,
				SelectionKey: selectionKey,
				Stake:        100,
			})
			tt.verify(t, result, err)
		})
	}
}

// ---------------------------------------------------------------------------
// TestPlaceBet_MarketTypeDispatchAndSelfBet
// Tests market type dispatch and self-bet prevention with a full request struct.
// ---------------------------------------------------------------------------

func TestPlaceBet_MarketTypeDispatchAndSelfBet(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	userUUID := uuid.New()
	roundID := sharedtypes.RoundID(uuid.New())
	marketID := int64(77)
	futureStart := time.Now().Add(24 * time.Hour)

	// A round with 3 players — enough for placement_2nd and placement_last markets.
	threePlayerRound := &roundtypes.Round{
		ID:      roundID,
		GuildID: "guild-1",
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "player-a", Response: roundtypes.ResponseAccept},
			{UserID: "player-b", Response: roundtypes.ResponseAccept},
			{UserID: "player-c", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	// A round with 2 players — not enough for placement_2nd.
	twoPlayerRound := &roundtypes.Round{
		ID:      roundID,
		GuildID: "guild-1",
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "player-a", Response: roundtypes.ResponseAccept},
			{UserID: "player-b", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	// Options for placement markets: option key = player Discord ID.
	placementOptions := []bettingdb.MarketOption{
		{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a", ProbabilityBps: 3333, DecimalOddsCents: 300},
		{MarketID: marketID, OptionKey: "player-b", ParticipantMemberID: "player-b", ProbabilityBps: 3333, DecimalOddsCents: 300},
		{MarketID: marketID, OptionKey: "player-c", ParticipantMemberID: "player-c", ProbabilityBps: 3334, DecimalOddsCents: 300},
	}

	// Options for O/U market: option key = {id}_over / {id}_under.
	ouOptions := []bettingdb.MarketOption{
		{MarketID: marketID, OptionKey: "player-a_over", ParticipantMemberID: "player-a", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":65}`},
		{MarketID: marketID, OptionKey: "player-a_under", ParticipantMemberID: "player-a", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":65}`},
		{MarketID: marketID, OptionKey: "player-b_over", ParticipantMemberID: "player-b", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":70}`},
		{MarketID: marketID, OptionKey: "player-b_under", ParticipantMemberID: "player-b", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":70}`},
		{MarketID: marketID, OptionKey: "player-c_over", ParticipantMemberID: "player-c", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":75}`},
		{MarketID: marketID, OptionKey: "player-c_under", ParticipantMemberID: "player-c", ProbabilityBps: 5000, DecimalOddsCents: 200, Metadata: `{"line":75}`},
	}

	// baseSetup wires the common fakes for all cases.
	baseSetup := func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository,
		lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository, round *roundtypes.Round) {
		userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
			return memberMembership(userUUID, clubUUID), nil
		}
		userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
			return "guild-1", nil
		}
		guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
			return enabledEntitlements(), nil
		}
		lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
			return &leaderboarddb.Season{ID: "2026-spring"}, nil
		}
		roundRepo.GetUpcomingRoundsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) ([]*roundtypes.Round, error) {
			return []*roundtypes.Round{round}, nil
		}
		roundRepo.GetRoundFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID, _ sharedtypes.RoundID) (*roundtypes.Round, error) {
			return round, nil
		}
		userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		repo.AcquireWalletBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (*bettingdb.WalletBalance, error) {
			return &bettingdb.WalletBalance{Balance: 1000, Reserved: 0}, nil
		}
		repo.ListBetsForUserAndMarketFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ int64) ([]bettingdb.Bet, error) {
			return nil, nil
		}
	}

	// makeMarket returns an open market of the given type.
	makeMarket := func(marketType string) *bettingdb.Market {
		return &bettingdb.Market{
			ID:         marketID,
			ClubUUID:   clubUUID,
			SeasonID:   "2026-spring",
			RoundID:    roundID.UUID(),
			MarketType: marketType,
			Status:     openMarketStatus,
			LocksAt:    futureStart,
		}
	}

	tests := []struct {
		name         string
		round        *roundtypes.Round
		marketType   string
		selectionKey string
		bettorID     string // Discord ID returned by GetUserByUUID
		options      []bettingdb.MarketOption
		wantErr      error
	}{
		{
			name:         "self-bet blocked on placement_2nd",
			round:        threePlayerRound,
			marketType:   placement2ndMarketType,
			selectionKey: "player-a",
			bettorID:     "player-a", // matches ParticipantMemberID
			options:      placementOptions,
			wantErr:      ErrSelfBetProhibited,
		},
		{
			name: "self-bet blocked on placement_3rd",
			round: func() *roundtypes.Round {
				// Need 4 players for placement_3rd.
				r := *threePlayerRound
				r.Participants = append(r.Participants, roundtypes.Participant{UserID: "player-d", Response: roundtypes.ResponseAccept})
				opts4 := make([]roundtypes.Participant, len(r.Participants))
				copy(opts4, r.Participants)
				r.Participants = opts4
				return &r
			}(),
			marketType:   placement3rdMarketType,
			selectionKey: "player-a",
			bettorID:     "player-a",
			options: []bettingdb.MarketOption{
				{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a", ProbabilityBps: 2500, DecimalOddsCents: 400},
				{MarketID: marketID, OptionKey: "player-b", ParticipantMemberID: "player-b", ProbabilityBps: 2500, DecimalOddsCents: 400},
				{MarketID: marketID, OptionKey: "player-c", ParticipantMemberID: "player-c", ProbabilityBps: 2500, DecimalOddsCents: 400},
				{MarketID: marketID, OptionKey: "player-d", ParticipantMemberID: "player-d", ProbabilityBps: 2500, DecimalOddsCents: 400},
			},
			wantErr: ErrSelfBetProhibited,
		},
		{
			name:         "self-bet blocked on placement_last",
			round:        threePlayerRound,
			marketType:   placementLastMarketType,
			selectionKey: "player-b",
			bettorID:     "player-b",
			options:      placementOptions,
			wantErr:      ErrSelfBetProhibited,
		},
		{
			name:         "self-bet blocked on over_under (over option)",
			round:        threePlayerRound,
			marketType:   overUnderMarketType,
			selectionKey: "player-a_over",
			bettorID:     "player-a",
			options:      ouOptions,
			wantErr:      ErrSelfBetProhibited,
		},
		{
			name:         "self-bet blocked on over_under (under option)",
			round:        threePlayerRound,
			marketType:   overUnderMarketType,
			selectionKey: "player-a_under",
			bettorID:     "player-a",
			options:      ouOptions,
			wantErr:      ErrSelfBetProhibited,
		},
		{
			name:         "self-bet allowed on round_winner",
			round:        twoPlayerRound,
			marketType:   winnerMarketType,
			selectionKey: "player-a",
			bettorID:     "player-a", // same as selection — allowed for winner market
			options: []bettingdb.MarketOption{
				{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a", ProbabilityBps: 5000, DecimalOddsCents: 200},
				{MarketID: marketID, OptionKey: "player-b", ParticipantMemberID: "player-b", ProbabilityBps: 5000, DecimalOddsCents: 200},
			},
			wantErr: nil, // no error — self-bet allowed on winner
		},
		{
			name:         "unknown market type returns ErrInvalidMarketType",
			round:        twoPlayerRound,
			marketType:   "bogus_market",
			selectionKey: "player-a",
			bettorID:     "player-a",
			options:      nil,
			wantErr:      ErrInvalidMarketType,
		},
		{
			name:         "placement_2nd with 2-player round returns ErrNoEligibleRound",
			round:        twoPlayerRound,
			marketType:   placement2ndMarketType,
			selectionKey: "player-a",
			bettorID:     "player-a",
			options:      nil,
			wantErr:      ErrNoEligibleRound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			guildRepo := NewFakeGuildRepository()
			lbRepo := NewFakeLeaderboardRepository()
			roundRepo := NewFakeRoundRepository()

			baseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo, tt.round)

			// Wire GetUserByUUID to return a user whose Discord ID is tt.bettorID.
			bettorDiscordID := sharedtypes.DiscordID(tt.bettorID)
			userRepo.GetUserByUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (*userdb.User, error) {
				return &userdb.User{UserID: &bettorDiscordID}, nil
			}

			if tt.options != nil {
				// Wire market lookup to return an existing market of the requested type.
				market := makeMarket(tt.marketType)
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return market, nil
				}
				opts := tt.options
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return opts, nil
				}
			}

			svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
			_, err := svc.PlaceBet(context.Background(), PlaceBetRequest{
				ClubUUID:     clubUUID,
				UserUUID:     userUUID,
				RoundID:      roundID,
				SelectionKey: tt.selectionKey,
				Stake:        100,
				MarketType:   tt.marketType,
			})

			if tt.wantErr == nil {
				// Happy path for self-bet allowed on winner — just check no error
				// (or only errors not related to self-bet/dispatch).
				if errors.Is(err, ErrSelfBetProhibited) {
					t.Errorf("expected no ErrSelfBetProhibited, got %v", err)
				}
				if errors.Is(err, ErrInvalidMarketType) {
					t.Errorf("expected no ErrInvalidMarketType, got %v", err)
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
