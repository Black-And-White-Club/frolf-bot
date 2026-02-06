package roundservice

import (
	"context"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake User Lookup
// ------------------------

type FakeUserLookup struct {
	FindByUsernameFn func(name string) sharedtypes.DiscordID
	FindByDisplayFn  func(name string) sharedtypes.DiscordID
	FindByPartialFn  func(name string) []*UserIdentity
}

func (f *FakeUserLookup) FindByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, n string) (*UserIdentity, error) {
	if f.FindByUsernameFn != nil {
		if id := f.FindByUsernameFn(n); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
		if id := f.FindByUsernameFn(strings.Title(n)); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
	}
	return nil, nil
}

func (f *FakeUserLookup) FindByNormalizedUDiscDisplayName(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, n string) (*UserIdentity, error) {
	if f.FindByDisplayFn != nil {
		if id := f.FindByDisplayFn(n); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
		if id := f.FindByDisplayFn(strings.Title(n)); id != "" {
			return &UserIdentity{UserID: id}, nil
		}
	}
	return nil, nil
}

func (f *FakeUserLookup) FindByPartialUDiscName(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, n string) ([]*UserIdentity, error) {
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

	CreateRoundFunc                    func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r *roundtypes.Round) error
	GetRoundFunc                       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	UpdateRoundFunc                    func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error)
	DeleteRoundFunc                    func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) error
	UpdateParticipantFunc              func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error)
	UpdateRoundStateFunc               func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, state roundtypes.RoundState) error
	GetUpcomingRoundsFunc              func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) ([]*roundtypes.Round, error)
	UpdateParticipantScoreFunc         func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, score sharedtypes.Score) error
	GetParticipantsWithResponsesFunc   func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, res ...string) ([]roundtypes.Participant, error)
	GetRoundStateFunc                  func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (roundtypes.RoundState, error)
	GetParticipantsFunc                func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateEventMessageIDFunc           func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)
	UpdateDiscordEventIDFunc           func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error)
	GetRoundByDiscordEventIDFunc       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error)
	GetParticipantFunc                 func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipantFunc              func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) ([]roundtypes.Participant, error)
	GetEventMessageIDFunc              func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (string, error)
	UpdateRoundsAndParticipantsFunc    func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error
	GetUpcomingRoundsByParticipantFunc func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, uID sharedtypes.DiscordID) ([]*roundtypes.Round, error)
	UpdateImportStatusFunc             func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error
	CreateRoundGroupsFunc              func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error
	RoundHasGroupsFunc                 func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error)
	GetRoundsByGuildIDFunc             func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error)
}

func NewFakeRepo() *FakeRepo {
	return &FakeRepo{
		trace: []string{},
	}
}

func (f *FakeRepo) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeRepo) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeRepo) CreateRound(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r *roundtypes.Round) error {
	f.record("CreateRound")
	if f.CreateRoundFunc != nil {
		return f.CreateRoundFunc(ctx, db, g, r)
	}
	return nil
}

func (f *FakeRepo) GetRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	f.record("GetRound")
	if f.GetRoundFunc != nil {
		return f.GetRoundFunc(ctx, db, guildID, roundID)
	}
	return &roundtypes.Round{
		ID:      roundID,
		GuildID: guildID,
	}, nil
}

func (f *FakeRepo) UpdateRound(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
	f.record("UpdateRound")
	if f.UpdateRoundFunc != nil {
		return f.UpdateRoundFunc(ctx, db, g, rID, r)
	}
	return r, nil
}

func (f *FakeRepo) DeleteRound(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) error {
	f.record("DeleteRound")
	if f.DeleteRoundFunc != nil {
		return f.DeleteRoundFunc(ctx, db, g, r)
	}
	return nil
}

func (f *FakeRepo) UpdateParticipant(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, p roundtypes.Participant) ([]roundtypes.Participant, error) {
	f.record("UpdateParticipant")
	if f.UpdateParticipantFunc != nil {
		return f.UpdateParticipantFunc(ctx, db, g, rID, p)
	}
	return nil, nil
}

func (f *FakeRepo) UpdateRoundState(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID, state roundtypes.RoundState) error {
	f.record("UpdateRoundState")
	if f.UpdateRoundStateFunc != nil {
		return f.UpdateRoundStateFunc(ctx, db, g, r, state)
	}
	return nil
}

func (f *FakeRepo) GetUpcomingRounds(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	f.record("GetUpcomingRounds")
	if f.GetUpcomingRoundsFunc != nil {
		return f.GetUpcomingRoundsFunc(ctx, db, g)
	}
	return nil, nil
}

func (f *FakeRepo) UpdateParticipantScore(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, score sharedtypes.Score) error {
	f.record("UpdateParticipantScore")
	if f.UpdateParticipantScoreFunc != nil {
		return f.UpdateParticipantScoreFunc(ctx, db, g, rID, uID, score)
	}
	return nil
}

func (f *FakeRepo) GetParticipantsWithResponses(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, res ...string) ([]roundtypes.Participant, error) {
	f.record("GetParticipantsWithResponses")
	if f.GetParticipantsWithResponsesFunc != nil {
		return f.GetParticipantsWithResponsesFunc(ctx, db, g, rID, res...)
	}
	return nil, nil
}

func (f *FakeRepo) GetRoundState(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (roundtypes.RoundState, error) {
	f.record("GetRoundState")
	if f.GetRoundStateFunc != nil {
		return f.GetRoundStateFunc(ctx, db, g, rID)
	}
	return "", nil
}

func (f *FakeRepo) GetParticipants(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
	f.record("GetParticipants")
	if f.GetParticipantsFunc != nil {
		return f.GetParticipantsFunc(ctx, db, g, rID)
	}
	return []roundtypes.Participant{}, nil
}

func (f *FakeRepo) UpdateEventMessageID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
	f.record("UpdateEventMessageID")
	if f.UpdateEventMessageIDFunc != nil {
		return f.UpdateEventMessageIDFunc(ctx, db, guildID, roundID, discordMessageID)
	}
	return &roundtypes.Round{}, nil
}

func (f *FakeRepo) UpdateDiscordEventID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error) {
	f.record("UpdateDiscordEventID")
	if f.UpdateDiscordEventIDFunc != nil {
		return f.UpdateDiscordEventIDFunc(ctx, db, guildID, roundID, discordEventID)
	}
	return &roundtypes.Round{}, nil
}

func (f *FakeRepo) GetRoundByDiscordEventID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error) {
	f.record("GetRoundByDiscordEventID")
	if f.GetRoundByDiscordEventIDFunc != nil {
		return f.GetRoundByDiscordEventIDFunc(ctx, db, guildID, discordEventID)
	}
	return &roundtypes.Round{}, nil
}

func (f *FakeRepo) GetParticipant(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) (*roundtypes.Participant, error) {
	f.record("GetParticipant")
	if f.GetParticipantFunc != nil {
		return f.GetParticipantFunc(ctx, db, g, rID, uID)
	}
	return nil, nil
}

func (f *FakeRepo) RemoveParticipant(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID) ([]roundtypes.Participant, error) {
	f.record("RemoveParticipant")
	if f.RemoveParticipantFunc != nil {
		return f.RemoveParticipantFunc(ctx, db, g, rID, uID)
	}
	return nil, nil
}

func (f *FakeRepo) GetEventMessageID(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (string, error) {
	f.record("GetEventMessageID")
	if f.GetEventMessageIDFunc != nil {
		return f.GetEventMessageIDFunc(ctx, db, g, rID)
	}
	return "", nil
}

func (f *FakeRepo) UpdateRoundsAndParticipants(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
	f.record("UpdateRoundsAndParticipants")
	if f.UpdateRoundsAndParticipantsFunc != nil {
		return f.UpdateRoundsAndParticipantsFunc(ctx, db, guildID, updates)
	}
	return nil
}

func (f *FakeRepo) GetUpcomingRoundsByParticipant(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, uID sharedtypes.DiscordID) ([]*roundtypes.Round, error) {
	f.record("GetUpcomingRoundsByParticipant")
	if f.GetUpcomingRoundsByParticipantFunc != nil {
		return f.GetUpcomingRoundsByParticipantFunc(ctx, db, g, uID)
	}
	return nil, nil
}

func (f *FakeRepo) UpdateImportStatus(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error {
	f.record("UpdateImportStatus")
	if f.UpdateImportStatusFunc != nil {
		return f.UpdateImportStatusFunc(ctx, db, guildID, roundID, importID, status, errorMessage, errorCode)
	}
	return nil
}

func (f *FakeRepo) CreateRoundGroups(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
	f.record("CreateRoundGroups")
	if f.CreateRoundGroupsFunc != nil {
		return f.CreateRoundGroupsFunc(ctx, db, roundID, participants)
	}
	return nil
}

func (f *FakeRepo) RoundHasGroups(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
	f.record("RoundHasGroups")
	if f.RoundHasGroupsFunc != nil {
		return f.RoundHasGroupsFunc(ctx, db, roundID)
	}
	return false, nil
}

func (f *FakeRepo) GetRoundsByGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error) {
	f.record("GetRoundsByGuildID")
	if f.GetRoundsByGuildIDFunc != nil {
		return f.GetRoundsByGuildIDFunc(ctx, db, guildID, states...)
	}
	return []*roundtypes.Round{}, nil
}

// ------------------------
// Fake Queue Service
// ------------------------

type FakeQueueService struct {
	trace []string

	ScheduleRoundStartFunc    func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayloadV1) error
	ScheduleRoundReminderFunc func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayloadV1) error
	CancelRoundJobsFunc       func(ctx context.Context, roundID sharedtypes.RoundID) error
	GetScheduledJobsFunc      func(ctx context.Context, roundID sharedtypes.RoundID) ([]roundqueue.JobInfo, error)
	HealthCheckFunc           func(ctx context.Context) error
	StartFunc                 func(ctx context.Context) error
	StopFunc                  func(ctx context.Context) error
}

func NewFakeQueueService() *FakeQueueService {
	return &FakeQueueService{
		trace: []string{},
	}
}

func (f *FakeQueueService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeQueueService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

func (f *FakeQueueService) ScheduleRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayloadV1) error {
	f.record("ScheduleRoundStart")
	if f.ScheduleRoundStartFunc != nil {
		return f.ScheduleRoundStartFunc(ctx, guildID, roundID, startTime, payload)
	}
	return nil
}

func (f *FakeQueueService) ScheduleRoundReminder(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayloadV1) error {
	f.record("ScheduleRoundReminder")
	if f.ScheduleRoundReminderFunc != nil {
		return f.ScheduleRoundReminderFunc(ctx, guildID, roundID, reminderTime, payload)
	}
	return nil
}

func (f *FakeQueueService) CancelRoundJobs(ctx context.Context, roundID sharedtypes.RoundID) error {
	f.record("CancelRoundJobs")
	if f.CancelRoundJobsFunc != nil {
		return f.CancelRoundJobsFunc(ctx, roundID)
	}
	return nil
}

func (f *FakeQueueService) GetScheduledJobs(ctx context.Context, roundID sharedtypes.RoundID) ([]roundqueue.JobInfo, error) {
	f.record("GetScheduledJobs")
	if f.GetScheduledJobsFunc != nil {
		return f.GetScheduledJobsFunc(ctx, roundID)
	}
	return nil, nil
}

func (f *FakeQueueService) HealthCheck(ctx context.Context) error {
	f.record("HealthCheck")
	if f.HealthCheckFunc != nil {
		return f.HealthCheckFunc(ctx)
	}
	return nil
}

func (f *FakeQueueService) Start(ctx context.Context) error {
	f.record("Start")
	if f.StartFunc != nil {
		return f.StartFunc(ctx)
	}
	return nil
}

func (f *FakeQueueService) Stop(ctx context.Context) error {
	f.record("Stop")
	if f.StopFunc != nil {
		return f.StopFunc(ctx)
	}
	return nil
}

// ------------------------
// Fake EventBus
// ------------------------

type FakeEventBus struct {
	PublishFunc           func(topic string, messages ...*message.Message) error
	SubscribeFunc         func(ctx context.Context, topic string) (<-chan *message.Message, error)
	CloseFunc             func() error
	GetNATSConnectionFunc func() *nc.Conn
	GetJetStreamFunc      func() jetstream.JetStream
	GetHealthCheckersFunc func() []eventbus.HealthChecker
	CreateStreamFunc      func(ctx context.Context, streamName string) error
	SubscribeForTestFunc  func(ctx context.Context, topic string) (<-chan *message.Message, error)
}

func (f *FakeEventBus) Publish(topic string, messages ...*message.Message) error {
	if f.PublishFunc != nil {
		return f.PublishFunc(topic, messages...)
	}
	return nil
}

func (f *FakeEventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if f.SubscribeFunc != nil {
		return f.SubscribeFunc(ctx, topic)
	}
	return nil, nil
}

func (f *FakeEventBus) Close() error {
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}
	return nil
}

func (f *FakeEventBus) GetNATSConnection() *nc.Conn {
	if f.GetNATSConnectionFunc != nil {
		return f.GetNATSConnectionFunc()
	}
	return nil
}

func (f *FakeEventBus) GetJetStream() jetstream.JetStream {
	if f.GetJetStreamFunc != nil {
		return f.GetJetStreamFunc()
	}
	return nil
}

func (f *FakeEventBus) GetHealthCheckers() []eventbus.HealthChecker {
	if f.GetHealthCheckersFunc != nil {
		return f.GetHealthCheckersFunc()
	}
	return nil
}

func (f *FakeEventBus) CreateStream(ctx context.Context, streamName string) error {
	if f.CreateStreamFunc != nil {
		return f.CreateStreamFunc(ctx, streamName)
	}
	return nil
}

func (f *FakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if f.SubscribeForTestFunc != nil {
		return f.SubscribeForTestFunc(ctx, topic)
	}
	return nil, nil
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
// Fake Clock
// ------------------------

// FakeClock has been moved to roundutil.FakeClock in app/modules/round/utils/fake_clock.go

// ------------------------
// Fake Guild Config Provider
// ------------------------

type FakeGuildConfigProvider struct {
	GetConfigFunc func(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
}

func (f *FakeGuildConfigProvider) GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	if f.GetConfigFunc != nil {
		return f.GetConfigFunc(ctx, guildID)
	}
	return nil, nil
}

// ------------------------
// Interface assertions
// ------------------------
var _ rounddb.Repository = (*FakeRepo)(nil)
var _ roundqueue.QueueService = (*FakeQueueService)(nil)
var _ roundutil.RoundValidator = (*FakeRoundValidator)(nil)
var _ roundtime.TimeParserInterface = (*FakeTimeParser)(nil)
var _ UserLookup = (*FakeUserLookup)(nil)
var _ eventbus.EventBus = (*FakeEventBus)(nil)
var _ GuildConfigProvider = (*FakeGuildConfigProvider)(nil)
