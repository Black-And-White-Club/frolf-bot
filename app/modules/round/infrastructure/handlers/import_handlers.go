package roundhandlers

import (
	"context"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScorecardUploaded transforms a scorecard upload into an import job request.
func (h *RoundHandlers) HandleScorecardUploaded(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	req := roundtypes.ImportCreateJobInput{
		ImportID:  payload.ImportID,
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		ChannelID: payload.ChannelID,
		FileName:  payload.FileName,
		FileData:  payload.FileData,
		UDiscURL:  payload.UDiscURL,
		Notes:     payload.Notes,
	}

	result, err := h.service.CreateImportJob(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{Topic: roundevents.ImportFailedV1, Payload: result.Failure}}, nil
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParseRequestedV1, Payload: result.Success.Job}}, nil
}

// HandleScorecardURLRequested transforms a URL request into an import job.
func (h *RoundHandlers) HandleScorecardURLRequested(ctx context.Context, payload *roundevents.ScorecardURLRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req := roundtypes.ImportCreateJobInput{
		ImportID:  payload.ImportID,
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		ChannelID: payload.ChannelID,
		UDiscURL:  payload.UDiscURL,
		Notes:     payload.Notes,
	}

	result, err := h.service.ScorecardURLRequested(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{Topic: roundevents.ImportFailedV1, Payload: result.Failure}}, nil
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParseRequestedV1, Payload: result.Success.Job}}, nil
}

// HandleParseScorecardRequest handles the actual parsing of file data (CSV or URL download).
func (h *RoundHandlers) HandleParseScorecardRequest(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.DebugContext(ctx, "parsing scorecard", attr.String("import_id", payload.ImportID))

	req := roundtypes.ImportParseScorecardInput{
		ImportID: payload.ImportID,
		GuildID:  payload.GuildID,
		RoundID:  payload.RoundID,
		FileName: payload.FileName,
		FileData: payload.FileData,
		FileURL:  payload.UDiscURL,
	}

	result, err := h.service.ParseScorecard(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{Topic: roundevents.ImportFailedV1, Payload: result.Failure}}, nil
	}

	payloadOut := &roundevents.ParsedScorecardPayloadV1{
		ImportID:       payload.ImportID,
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		UserID:         payload.UserID,
		ChannelID:      payload.ChannelID,
		EventMessageID: payload.MessageID,
		ParsedData:     result.Success,
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParsedForNormalizationV1, Payload: payloadOut}}, nil
}

// HandleScorecardParsedForNormalization takes raw parsed data and structures it.
func (h *RoundHandlers) HandleScorecardParsedForNormalization(ctx context.Context, payload *roundevents.ParsedScorecardPayloadV1) ([]handlerwrapper.Result, error) {
	meta := roundtypes.Metadata{
		ImportID:       payload.ImportID,
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		UserID:         payload.UserID,
		ChannelID:      payload.ChannelID,
		EventMessageID: payload.EventMessageID,
	}

	// 1. Pass ctx
	// 2. Service now returns results.OperationResult
	result, err := h.service.NormalizeParsedScorecard(ctx, payload.ParsedData, meta)
	if err != nil {
		return nil, err
	}

	// 3. Now mapOperationResult works because 'result' is the correct type
	if result.Failure != nil {
		h.logger.WarnContext(ctx, "NormalizeParsedScorecard returned failure",
			attr.String("import_id", payload.ImportID),
			attr.Any("failure", result.Failure),
			attr.String("error_msg", (*result.Failure).Error()),
		)
	}

	return mapOperationResult(result.Map(
		func(s *roundtypes.NormalizedScorecard) any {
			return &roundevents.ScorecardNormalizedPayloadV1{
				ImportID:       payload.ImportID,
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				EventMessageID: payload.EventMessageID,
				Normalized:     *s,
				Timestamp:      time.Now().UTC(),
			}
		},
		func(f error) any { return f },
	), roundevents.ScorecardNormalizedV1, roundevents.ImportFailedV1), nil
}

// HandleScorecardNormalized handles the ingestion/matching of names.
func (h *RoundHandlers) HandleScorecardNormalized(ctx context.Context, payload *roundevents.ScorecardNormalizedPayloadV1) ([]handlerwrapper.Result, error) {
	req := roundtypes.ImportIngestScorecardInput{
		ImportID:       payload.ImportID,
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		UserID:         payload.UserID,
		ChannelID:      payload.ChannelID,
		EventMessageID: payload.EventMessageID,
		NormalizedData: payload.Normalized,
	}

	result, err := h.service.IngestNormalizedScorecard(ctx, req)
	if err != nil {
		return nil, err
	}
	return mapOperationResult(result.Map(
		func(s *roundtypes.IngestScorecardResult) any {
			return &roundevents.ImportCompletedPayloadV1{
				ImportID:       s.ImportID,
				GuildID:        s.GuildID,
				RoundID:        s.RoundID,
				Scores:         s.Scores,
				EventMessageID: s.EventMessageID,
				Timestamp:      s.Timestamp,
			}
		},
		func(f error) any { return f },
	), roundevents.ImportCompletedV1, roundevents.ImportFailedV1), nil
}

// HandleImportCompleted routes the final scores (Singles to Leaderboard, Doubles to Score Module).
func (h *RoundHandlers) HandleImportCompleted(
	ctx context.Context,
	payload *roundevents.ImportCompletedPayloadV1,
) ([]handlerwrapper.Result, error) {
	scores := make([]roundtypes.ImportScoreData, len(payload.Scores))
	for i, s := range payload.Scores {
		scores[i] = roundtypes.ImportScoreData{
			UserID:  s.UserID,
			RawName: s.RawName,
			Score:   int(s.Score),
			TeamID:  s.TeamID,
		}
	}

	req := roundtypes.ImportApplyScoresInput{
		GuildID:  payload.GuildID,
		RoundID:  payload.RoundID,
		ImportID: payload.ImportID,
		Scores:   scores,
	}

	res, err := h.service.ApplyImportedScores(ctx, req)
	if err != nil {
		return nil, err
	}

	if res.Failure != nil {
		return []handlerwrapper.Result{{Topic: roundevents.ImportFailedV1, Payload: res.Failure}}, nil
	}

	// Map Result to Event Payload
	success := *res.Success

	// 1. Emit RoundParticipantScoreUpdatedV1 for UI/Projections
	scoreUpdatedPayload := &roundevents.ParticipantScoreUpdatedPayloadV1{
		GuildID:        success.GuildID,
		RoundID:        success.RoundID,
		Participants:   success.Participants,
		EventMessageID: success.EventMessageID,
	}

	// 2. Emit RoundAllScoresSubmittedV1 to trigger finalization
	// This is what the integration tests expect.
	allScoresPayload := &roundevents.AllScoresSubmittedPayloadV1{
		GuildID:        success.GuildID,
		RoundID:        success.RoundID,
		EventMessageID: success.EventMessageID,
		Participants:   success.Participants,
		Teams:          success.Teams,
	}
	if success.RoundData != nil {
		allScoresPayload.RoundData = *success.RoundData
		mode := sharedtypes.RoundModeSingles
		if len(success.RoundData.Teams) > 0 {
			mode = sharedtypes.RoundModeDoubles
		}
		allScoresPayload.RoundMode = mode
	}

	results := []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundParticipantScoreUpdatedV1,
			Payload: scoreUpdatedPayload,
		},
		{
			Topic:   roundevents.RoundAllScoresSubmittedV1,
			Payload: allScoresPayload,
		},
	}

	// Add guild-scoped version for PWA permission scoping
	results = addGuildScopedResult(results, roundevents.RoundParticipantScoreUpdatedV1, success.GuildID)

	return results, nil
}
