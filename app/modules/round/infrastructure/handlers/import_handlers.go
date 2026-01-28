package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
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

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParsedForNormalizationV1, Payload: result.Success}}, nil
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
	return mapOperationResult(result, roundevents.ScorecardNormalizedV1, roundevents.ImportFailedV1), nil
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
	return mapOperationResult(result, roundevents.ImportCompletedV1, roundevents.ImportFailedV1), nil
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

	applied := &roundevents.ImportScoresAppliedPayloadV1{
		GuildID:        success.GuildID,
		RoundID:        success.RoundID,
		ImportID:       success.ImportID,
		Participants:   success.Participants,
		EventMessageID: success.EventMessageID,
		Timestamp:      success.Timestamp,
	}

	// --- LOGGING FOR VISIBILITY ---
	userIDs := make([]string, 0, len(applied.Participants))
	for _, p := range applied.Participants {
		if p.Score != nil {
			userIDs = append(userIDs, string(p.UserID))
		}
	}

	h.logger.InfoContext(ctx, "Import scores applied successfully",
		attr.String("round_id", applied.RoundID.String()),
		attr.Int("participant_count", len(userIDs)),
		attr.Any("imported_user_ids", userIDs),
	)

	results := []handlerwrapper.Result{
		{
			Topic: roundevents.RoundParticipantScoreUpdatedV1,
			Payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        applied.GuildID,
				RoundID:        applied.RoundID,
				EventMessageID: applied.EventMessageID,
				Participants:   applied.Participants,
			},
		},
	}

	// Add guild-scoped version for PWA permission scoping
	results = addGuildScopedResult(results, roundevents.RoundParticipantScoreUpdatedV1, applied.GuildID)

	return results, nil
}
