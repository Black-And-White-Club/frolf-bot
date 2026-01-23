package roundservice

import (
	"context"
	"strings"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// ------------------------
// Fake User Lookup
// ------------------------

type FakeUserLookup struct {
	FindByUsernameFn func(name string) sharedtypes.DiscordID
	FindByDisplayFn  func(name string) sharedtypes.DiscordID
	FindByPartialFn  func(name string) []*UserIdentity
}

func (f *FakeUserLookup) FindByNormalizedUDiscUsername(ctx context.Context, g sharedtypes.GuildID, n string) (*UserIdentity, error) {
	if f.FindByUsernameFn != nil {
		// Try the normalized form first, then a best-effort title-cased form to
		// accommodate test stubs that expect capitalized names (e.g., "Alice").
		if id := f.FindByUsernameFn(n); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
		if id := f.FindByUsernameFn(strings.Title(n)); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
	}
	return nil, nil
}

func (f *FakeUserLookup) FindByNormalizedUDiscDisplayName(ctx context.Context, g sharedtypes.GuildID, n string) (*UserIdentity, error) {
	if f.FindByDisplayFn != nil {
		// Try the normalized form first, then a title-cased form to match test
		// stubs which often provide capitalized display names.
		if id := f.FindByDisplayFn(n); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
		if id := f.FindByDisplayFn(strings.Title(n)); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
	}
	return nil, nil
}

func (f *FakeUserLookup) FindByPartialUDiscName(ctx context.Context, g sharedtypes.GuildID, n string) ([]*UserIdentity, error) {
	if f.FindByPartialFn != nil {
		return f.FindByPartialFn(n), nil
	}
	return []*UserIdentity{}, nil
}

// ------------------------
// Fake Repo
// ------------------------

type FakeRepo struct {
	trace []string

	CreateRoundFunc                 func(ctx context.Context, g sharedtypes.GuildID, r *roundtypes.Round) error
	RoundHasGroupsFunc              func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error)
	CreateRoundGroupsFunc           func(roundID sharedtypes.RoundID, participants []roundtypes.Participant) error
	UpdateImportStatusFunc          func(guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string) error
	GetRoundFunc                    func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	GetParticipantsFunc             func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateRoundStateFunc            func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID, state roundtypes.RoundState) error
	UpdateEventMessageIDFunc        func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)
	UpdateParticipantScoreFunc      func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, score sharedtypes.Score) error
	UpdateRoundsAndParticipantsFunc func(ctx context.Context, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error
	UpdateRoundFunc                 func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error)
}

// NewFakeRepo creates a lightweight fake repo for unit tests
func NewFakeRepo() *FakeRepo {
	return &FakeRepo{
		trace: []string{},
	}
}

// record appends a trace entry
func (f *FakeRepo) record(step string) {
	f.trace = append(f.trace, step)
}

// ---------------------- Stub Parser & Factory ----------------------
type StubParser struct{}

func (p *StubParser) Parse(_ []byte) (*roundtypes.ParsedScorecard, error) {
	return &roundtypes.ParsedScorecard{
		PlayerScores: []roundtypes.PlayerScoreRow{
			{PlayerName: "John Doe", Total: 50},
		},
	}, nil
}

type StubFactory struct{}

func (f *StubFactory) GetParser(filename string) (parsers.Parser, error) {
	return &StubParser{}, nil
}

// --- Methods under test ---

func (f *FakeRepo) UpdateRoundState(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID, s roundtypes.RoundState) error {
	f.record("UpdateRoundState")
	if f.UpdateRoundStateFunc != nil {
		return f.UpdateRoundStateFunc(ctx, g, r, s)
	}
	return nil
}

func (f *FakeRepo) CreateRound(ctx context.Context, g sharedtypes.GuildID, r *roundtypes.Round) error {
	f.record("CreateRound")
	if f.CreateRoundFunc != nil {
		return f.CreateRoundFunc(ctx, g, r)
	}
	return nil
}

func (f *FakeRepo) RoundHasGroups(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) {
	f.record("RoundHasGroups")
	if f.RoundHasGroupsFunc != nil {
		return f.RoundHasGroupsFunc(ctx, roundID)
	}
	return false, nil
}

func (f *FakeRepo) CreateRoundGroups(ctx context.Context, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
	f.record("CreateRoundGroups")
	if f.CreateRoundGroupsFunc != nil {
		return f.CreateRoundGroupsFunc(roundID, participants)
	}
	return nil
}

func (f *FakeRepo) UpdateImportStatus(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error {
	f.record("UpdateImportStatus")
	if f.UpdateImportStatusFunc != nil {
		return f.UpdateImportStatusFunc(guildID, roundID, importID, status)
	}
	return nil
}

func (f *FakeRepo) UpdateEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
	f.record("UpdateEventMessageID")
	if f.UpdateEventMessageIDFunc != nil {
		return f.UpdateEventMessageIDFunc(ctx, guildID, roundID, discordMessageID)
	}
	return &roundtypes.Round{}, nil
}

func (f *FakeRepo) UpdateParticipantScore(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, score sharedtypes.Score) error {
	f.record("UpdateParticipantScore")
	if f.UpdateParticipantScoreFunc != nil {
		return f.UpdateParticipantScoreFunc(ctx, g, rID, uID, score)
	}
	return nil
}

func (f *FakeRepo) UpdateRound(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
	f.record("UpdateRound")
	if f.UpdateRoundFunc != nil {
		return f.UpdateRoundFunc(ctx, g, rID, r)
	}
	return r, nil // Default: just return the round passed in
}

func (f *FakeRepo) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	f.record("GetRound")
	if f.GetRoundFunc != nil {
		return f.GetRoundFunc(ctx, guildID, roundID)
	}
	return &roundtypes.Round{
		ID:      roundID,
		GuildID: guildID,
	}, nil
}

func (f *FakeRepo) GetParticipants(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
	f.record("GetParticipants")
	if f.GetParticipantsFunc != nil {
		return f.GetParticipantsFunc(ctx, g, rID)
	}
	return []roundtypes.Participant{}, nil
}

func (f *FakeRepo) UpdateRoundsAndParticipants(ctx context.Context, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
	f.record("UpdateRoundsAndParticipants")
	if f.UpdateRoundsAndParticipantsFunc != nil {
		return f.UpdateRoundsAndParticipantsFunc(ctx, guildID, updates)
	}
	return nil
}

// --- Accessors for assertions ---

func (f *FakeRepo) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// --- Other repo methods for interface satisfaction ---

func (f *FakeRepo) DeleteRound(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) error {
	return nil
}

func (f *FakeRepo) UpdateParticipant(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error) {
	return nil, nil
}

func (f *FakeRepo) GetUpcomingRounds(ctx context.Context, g sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	return nil, nil
}

func (f *FakeRepo) GetParticipantsWithResponses(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, res ...string) ([]roundtypes.Participant, error) {
	return nil, nil
}
func (f *FakeRepo) GetRoundState(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) (roundtypes.RoundState, error) {
	return "", nil
}

func (f *FakeRepo) GetParticipant(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) (*roundtypes.Participant, error) {
	return nil, nil
}
func (f *FakeRepo) RemoveParticipant(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
	return nil, nil
}
func (f *FakeRepo) GetEventMessageID(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID) (string, error) {
	return "", nil
}

func (f *FakeRepo) GetUpcomingRoundsByParticipant(ctx context.Context, g sharedtypes.GuildID, uID sharedtypes.DiscordID) ([]*roundtypes.Round, error) {
	return nil, nil
}

// ------------------------
// Fake Round Validator
// ------------------------

type FakeRoundValidator struct {
	ValidateInput       func(input roundtypes.CreateRoundInput) []string
	ValidateBasePayload func(input roundtypes.BaseRoundPayload) []string
}

func (f *FakeRoundValidator) ValidateRoundInput(input roundtypes.CreateRoundInput) []string {
	if f.ValidateInput != nil {
		return f.ValidateInput(input)
	}
	return roundutil.NewRoundValidator().ValidateRoundInput(input)
}

func (f *FakeRoundValidator) ValidateBaseRoundPayload(input roundtypes.BaseRoundPayload) []string {
	if f.ValidateBasePayload != nil {
		return f.ValidateBasePayload(input)
	}
	return roundutil.NewRoundValidator().ValidateBaseRoundPayload(input)
}

// ------------------------
// Fake Time Parser
// ------------------------

type FakeTimeParser struct {
	GetTimezoneInput func(input string) (string, bool)
	ParseFn          func(startTimeStr string, timezone roundtypes.Timezone, clock roundutil.Clock) (int64, error)
}

func (f *FakeTimeParser) GetTimezoneFromInput(input string) (string, bool) {
	if f.GetTimezoneInput != nil {
		return f.GetTimezoneInput(input)
	}
	return "America/Chicago", false
}

func (f *FakeTimeParser) ParseUserTimeInput(startTimeStr string, timezone roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
	if f.ParseFn != nil {
		return f.ParseFn(startTimeStr, timezone, clock)
	}
	return time.Now().Unix(), nil
}

// ------------------------
// Interface assertions
// ------------------------
var _ rounddb.Repository = (*FakeRepo)(nil)
var _ roundutil.RoundValidator = (*FakeRoundValidator)(nil)
var _ roundtime.TimeParserInterface = (*FakeTimeParser)(nil)
