package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// ------------------------
// Fake Service
// ------------------------

type FakeService struct {
	trace []string

	// Create Round
	ValidateAndProcessRoundWithClockFn func(ctx context.Context, payload roundevents.CreateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (results.OperationResult, error)
	StoreRoundFn                       func(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundEntityCreatedPayloadV1) (results.OperationResult, error)
	UpdateRoundMessageIDFn             func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error)

	// Update Round
	ValidateAndProcessRoundUpdateWithClockFn func(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (results.OperationResult, error)
	ValidateAndProcessRoundUpdateFn          func(ctx context.Context, payload roundevents.UpdateRoundRequestedPayloadV1, timeParser roundtime.TimeParserInterface) (results.OperationResult, error)
	UpdateRoundEntityFn                      func(ctx context.Context, payload roundevents.RoundUpdateValidatedPayloadV1) (results.OperationResult, error)
	UpdateScheduledRoundEventsFn             func(ctx context.Context, payload roundevents.RoundScheduleUpdatePayloadV1) (results.OperationResult, error)

	// Delete Round
	ValidateRoundDeleteRequestFn func(ctx context.Context, payload roundevents.RoundDeleteRequestPayloadV1) (results.OperationResult, error)
	DeleteRoundFn                func(ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayloadV1) (results.OperationResult, error)

	// Start Round
	ProcessRoundStartFn func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult, error)

	// Join Round
	ValidateParticipantJoinRequestFn func(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)
	UpdateParticipantStatusFn        func(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)
	ParticipantRemovalFn             func(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayloadV1) (results.OperationResult, error)
	CheckParticipantStatusFn         func(ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error)

	// Score Round
	ValidateScoreUpdateRequestFn  func(ctx context.Context, payload roundevents.ScoreUpdateRequestPayloadV1) (results.OperationResult, error)
	UpdateParticipantScoreFn      func(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayloadV1) (results.OperationResult, error)
	UpdateParticipantScoresBulkFn func(ctx context.Context, payload roundevents.ScoreBulkUpdateRequestPayloadV1) (results.OperationResult, error)
	CheckAllScoresSubmittedFn     func(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayloadV1) (results.OperationResult, error)

	// Finalize Round
	FinalizeRoundFn     func(ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) (results.OperationResult, error)
	NotifyScoreModuleFn func(ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) (results.OperationResult, error)

	// Round Reminder
	ProcessRoundReminderFn func(ctx context.Context, payload roundevents.DiscordReminderPayloadV1) (results.OperationResult, error)

	// Retrieve Round
	GetRoundFn func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult, error)

	// Schedule Round Events
	ScheduleRoundEventsFn func(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundScheduledPayloadV1, discordMessageID string) (results.OperationResult, error)

	// Update Participant Tags
	UpdateScheduledRoundsWithNewTagsFn func(ctx context.Context, guildID sharedtypes.GuildID, changedTags map[sharedtypes.DiscordID]sharedtypes.TagNumber) (results.OperationResult, error)

	// Scorecard Import
	ScorecardURLRequestedFn     func(ctx context.Context, payload roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error)
	CreateImportJobFn           func(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error)
	ParseScorecardFn            func(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1, fileData []byte) (results.OperationResult, error)
	NormalizeParsedScorecardFn  func(ctx context.Context, parsed *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult, error)
	IngestNormalizedScorecardFn func(ctx context.Context, payload roundevents.ScorecardNormalizedPayloadV1) (results.OperationResult, error)
	ApplyImportedScoresFn       func(ctx context.Context, payload roundevents.ImportCompletedPayloadV1) (results.OperationResult, error)
}

func NewFakeService() *FakeService {
	return &FakeService{trace: []string{}}
}

func (f *FakeService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeService) Trace() []string {
	return f.trace
}

// --- Implementation ---

func (f *FakeService) ValidateAndProcessRoundWithClock(ctx context.Context, p roundevents.CreateRoundRequestedPayloadV1, tp roundtime.TimeParserInterface, c roundutil.Clock) (results.OperationResult, error) {
	f.record("ValidateAndProcessRoundWithClock")
	if f.ValidateAndProcessRoundWithClockFn != nil {
		return f.ValidateAndProcessRoundWithClockFn(ctx, p, tp, c)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) StoreRound(ctx context.Context, g sharedtypes.GuildID, p roundevents.RoundEntityCreatedPayloadV1) (results.OperationResult, error) {
	f.record("StoreRound")
	if f.StoreRoundFn != nil {
		return f.StoreRoundFn(ctx, g, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateRoundMessageID(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID, m string) (*roundtypes.Round, error) {
	f.record("UpdateRoundMessageID")
	if f.UpdateRoundMessageIDFn != nil {
		return f.UpdateRoundMessageIDFn(ctx, g, r, m)
	}
	return &roundtypes.Round{}, nil
}

func (f *FakeService) ValidateAndProcessRoundUpdateWithClock(ctx context.Context, p roundevents.UpdateRoundRequestedPayloadV1, tp roundtime.TimeParserInterface, c roundutil.Clock) (results.OperationResult, error) {
	f.record("ValidateAndProcessRoundUpdateWithClock")
	if f.ValidateAndProcessRoundUpdateWithClockFn != nil {
		return f.ValidateAndProcessRoundUpdateWithClockFn(ctx, p, tp, c)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ValidateAndProcessRoundUpdate(ctx context.Context, p roundevents.UpdateRoundRequestedPayloadV1, tp roundtime.TimeParserInterface) (results.OperationResult, error) {
	f.record("ValidateAndProcessRoundUpdate")
	if f.ValidateAndProcessRoundUpdateFn != nil {
		return f.ValidateAndProcessRoundUpdateFn(ctx, p, tp)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateRoundEntity(ctx context.Context, p roundevents.RoundUpdateValidatedPayloadV1) (results.OperationResult, error) {
	f.record("UpdateRoundEntity")
	if f.UpdateRoundEntityFn != nil {
		return f.UpdateRoundEntityFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateScheduledRoundEvents(ctx context.Context, p roundevents.RoundScheduleUpdatePayloadV1) (results.OperationResult, error) {
	f.record("UpdateScheduledRoundEvents")
	if f.UpdateScheduledRoundEventsFn != nil {
		return f.UpdateScheduledRoundEventsFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ValidateRoundDeleteRequest(ctx context.Context, p roundevents.RoundDeleteRequestPayloadV1) (results.OperationResult, error) {
	f.record("ValidateRoundDeleteRequest")
	if f.ValidateRoundDeleteRequestFn != nil {
		return f.ValidateRoundDeleteRequestFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) DeleteRound(ctx context.Context, p roundevents.RoundDeleteAuthorizedPayloadV1) (results.OperationResult, error) {
	f.record("DeleteRound")
	if f.DeleteRoundFn != nil {
		return f.DeleteRoundFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ProcessRoundStart(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (results.OperationResult, error) {
	f.record("ProcessRoundStart")
	if f.ProcessRoundStartFn != nil {
		return f.ProcessRoundStartFn(ctx, g, r)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ValidateParticipantJoinRequest(ctx context.Context, p roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error) {
	f.record("ValidateParticipantJoinRequest")
	if f.ValidateParticipantJoinRequestFn != nil {
		return f.ValidateParticipantJoinRequestFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateParticipantStatus(ctx context.Context, p roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error) {
	f.record("UpdateParticipantStatus")
	if f.UpdateParticipantStatusFn != nil {
		return f.UpdateParticipantStatusFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ParticipantRemoval(ctx context.Context, p roundevents.ParticipantRemovalRequestPayloadV1) (results.OperationResult, error) {
	f.record("ParticipantRemoval")
	if f.ParticipantRemovalFn != nil {
		return f.ParticipantRemovalFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) CheckParticipantStatus(ctx context.Context, p roundevents.ParticipantJoinRequestPayloadV1) (results.OperationResult, error) {
	f.record("CheckParticipantStatus")
	if f.CheckParticipantStatusFn != nil {
		return f.CheckParticipantStatusFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ValidateScoreUpdateRequest(ctx context.Context, p roundevents.ScoreUpdateRequestPayloadV1) (results.OperationResult, error) {
	f.record("ValidateScoreUpdateRequest")
	if f.ValidateScoreUpdateRequestFn != nil {
		return f.ValidateScoreUpdateRequestFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateParticipantScore(ctx context.Context, p roundevents.ScoreUpdateValidatedPayloadV1) (results.OperationResult, error) {
	f.record("UpdateParticipantScore")
	if f.UpdateParticipantScoreFn != nil {
		return f.UpdateParticipantScoreFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateParticipantScoresBulk(ctx context.Context, p roundevents.ScoreBulkUpdateRequestPayloadV1) (results.OperationResult, error) {
	f.record("UpdateParticipantScoresBulk")
	if f.UpdateParticipantScoresBulkFn != nil {
		return f.UpdateParticipantScoresBulkFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) CheckAllScoresSubmitted(ctx context.Context, p roundevents.ParticipantScoreUpdatedPayloadV1) (results.OperationResult, error) {
	f.record("CheckAllScoresSubmitted")
	if f.CheckAllScoresSubmittedFn != nil {
		return f.CheckAllScoresSubmittedFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) FinalizeRound(ctx context.Context, p roundevents.AllScoresSubmittedPayloadV1) (results.OperationResult, error) {
	f.record("FinalizeRound")
	if f.FinalizeRoundFn != nil {
		return f.FinalizeRoundFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) NotifyScoreModule(ctx context.Context, p roundevents.RoundFinalizedPayloadV1) (results.OperationResult, error) {
	f.record("NotifyScoreModule")
	if f.NotifyScoreModuleFn != nil {
		return f.NotifyScoreModuleFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ProcessRoundReminder(ctx context.Context, p roundevents.DiscordReminderPayloadV1) (results.OperationResult, error) {
	f.record("ProcessRoundReminder")
	if f.ProcessRoundReminderFn != nil {
		return f.ProcessRoundReminderFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) GetRound(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (results.OperationResult, error) {
	f.record("GetRound")
	if f.GetRoundFn != nil {
		return f.GetRoundFn(ctx, g, r)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ScheduleRoundEvents(ctx context.Context, g sharedtypes.GuildID, p roundevents.RoundScheduledPayloadV1, m string) (results.OperationResult, error) {
	f.record("ScheduleRoundEvents")
	if f.ScheduleRoundEventsFn != nil {
		return f.ScheduleRoundEventsFn(ctx, g, p, m)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) UpdateScheduledRoundsWithNewTags(ctx context.Context, g sharedtypes.GuildID, ct map[sharedtypes.DiscordID]sharedtypes.TagNumber) (results.OperationResult, error) {
	f.record("UpdateScheduledRoundsWithNewTags")
	if f.UpdateScheduledRoundsWithNewTagsFn != nil {
		return f.UpdateScheduledRoundsWithNewTagsFn(ctx, g, ct)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ScorecardURLRequested(ctx context.Context, p roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error) {
	f.record("ScorecardURLRequested")
	if f.ScorecardURLRequestedFn != nil {
		return f.ScorecardURLRequestedFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) CreateImportJob(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error) {
	f.record("CreateImportJob")
	if f.CreateImportJobFn != nil {
		return f.CreateImportJobFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ParseScorecard(ctx context.Context, p roundevents.ScorecardUploadedPayloadV1, fd []byte) (results.OperationResult, error) {
	f.record("ParseScorecard")
	if f.ParseScorecardFn != nil {
		return f.ParseScorecardFn(ctx, p, fd)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) NormalizeParsedScorecard(ctx context.Context, pr *roundtypes.ParsedScorecard, m roundtypes.Metadata) (results.OperationResult, error) {
	f.record("NormalizeParsedScorecard")
	if f.NormalizeParsedScorecardFn != nil {
		return f.NormalizeParsedScorecardFn(ctx, pr, m)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) IngestNormalizedScorecard(ctx context.Context, p roundevents.ScorecardNormalizedPayloadV1) (results.OperationResult, error) {
	f.record("IngestNormalizedScorecard")
	if f.IngestNormalizedScorecardFn != nil {
		return f.IngestNormalizedScorecardFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

func (f *FakeService) ApplyImportedScores(ctx context.Context, p roundevents.ImportCompletedPayloadV1) (results.OperationResult, error) {
	f.record("ApplyImportedScores")
	if f.ApplyImportedScoresFn != nil {
		return f.ApplyImportedScoresFn(ctx, p)
	}
	return results.OperationResult{}, nil
}

// Interface assertion
var _ roundservice.Service = (*FakeService)(nil)
