package roundhandlers

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

const (
	adminPwaImportSource      = "admin_pwa_upload"
	discordUploadImportSource = "discord_upload"
	discordURLImportSource    = "discord_url"
)

// HandleAdminScorecardUploadRequested accepts admin-originated uploads from PWA.
// It enforces admin role and forces guest fallback + overwrite semantics.
func (h *RoundHandlers) HandleAdminScorecardUploadRequested(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Admin scorecard upload requested",
		attr.String("source", adminPwaImportSource),
		attr.String("import_id", payload.ImportID),
		attr.String("round_id", payload.RoundID.String()),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("user_id", string(payload.UserID)),
	)

	if err := h.ensureAdminRole(ctx, payload.GuildID, payload.UserID); err != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.MessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          err.Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
	}

	req := roundtypes.ImportCreateJobInput{
		ImportID:  payload.ImportID,
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		Source:    adminPwaImportSource,
		UserID:    payload.UserID,
		ChannelID: payload.ChannelID,
		FileName:  payload.FileName,
		FileData:  payload.FileData,
		UDiscURL:  payload.UDiscURL,
		Notes:     payload.Notes,
		// Forced for admin upload path.
		AllowGuestPlayers:       true,
		OverwriteExistingScores: true,
	}

	result, err := h.service.CreateImportJob(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.MessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          (*result.Failure).Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
	}
	parsePayload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID:                payload.ImportID,
		Source:                  adminPwaImportSource,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		MessageID:               payload.MessageID,
		FileData:                payload.FileData,
		FileURL:                 payload.FileURL,
		FileName:                payload.FileName,
		UDiscURL:                payload.UDiscURL,
		Notes:                   payload.Notes,
		AllowGuestPlayers:       true,
		OverwriteExistingScores: true,
		Timestamp:               time.Now().UTC(),
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParseRequestedV1, Payload: parsePayload}}, nil
}

// HandleScorecardUploaded transforms a scorecard upload into an import job request.
func (h *RoundHandlers) HandleScorecardUploaded(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	source := payload.Source
	if source == "" {
		source = discordUploadImportSource
	}

	req := roundtypes.ImportCreateJobInput{
		ImportID:  payload.ImportID,
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		Source:    source,
		UserID:    payload.UserID,
		ChannelID: payload.ChannelID,
		FileName:  payload.FileName,
		FileData:  payload.FileData,
		UDiscURL:  payload.UDiscURL,
		Notes:     payload.Notes,
		// Non-admin path keeps standard matching semantics.
		AllowGuestPlayers:       false,
		OverwriteExistingScores: false,
	}

	result, err := h.service.CreateImportJob(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.MessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          (*result.Failure).Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
	}
	parsePayload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID:                payload.ImportID,
		Source:                  source,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		MessageID:               payload.MessageID,
		FileData:                payload.FileData,
		FileURL:                 payload.FileURL,
		FileName:                payload.FileName,
		UDiscURL:                payload.UDiscURL,
		Notes:                   payload.Notes,
		AllowGuestPlayers:       false,
		OverwriteExistingScores: false,
		Timestamp:               time.Now().UTC(),
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParseRequestedV1, Payload: parsePayload}}, nil
}

// HandleScorecardURLRequested transforms a URL request into an import job.
func (h *RoundHandlers) HandleScorecardURLRequested(ctx context.Context, payload *roundevents.ScorecardURLRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req := roundtypes.ImportCreateJobInput{
		ImportID:                payload.ImportID,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		Source:                  discordURLImportSource,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		UDiscURL:                payload.UDiscURL,
		Notes:                   payload.Notes,
		AllowGuestPlayers:       false,
		OverwriteExistingScores: false,
	}

	result, err := h.service.ScorecardURLRequested(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.MessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          (*result.Failure).Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
	}
	normalizedURL := payload.UDiscURL
	if result.Success.Job != nil && result.Success.Job.UDiscURL != "" {
		normalizedURL = result.Success.Job.UDiscURL
	}
	parsePayload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID:                payload.ImportID,
		Source:                  discordURLImportSource,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		MessageID:               payload.MessageID,
		UDiscURL:                normalizedURL,
		Notes:                   payload.Notes,
		AllowGuestPlayers:       false,
		OverwriteExistingScores: false,
		Timestamp:               time.Now().UTC(),
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParseRequestedV1, Payload: parsePayload}}, nil
}

// HandleParseScorecardRequest handles the actual parsing of file data (CSV or URL download).
func (h *RoundHandlers) HandleParseScorecardRequest(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.DebugContext(ctx, "parsing scorecard",
		attr.String("import_id", payload.ImportID),
		attr.String("source", payload.Source),
	)

	req := roundtypes.ImportParseScorecardInput{
		ImportID: payload.ImportID,
		GuildID:  payload.GuildID,
		RoundID:  payload.RoundID,
		Source:   payload.Source,
		FileName: payload.FileName,
		FileData: payload.FileData,
		FileURL:  payload.FileURL,
	}
	if req.FileURL == "" {
		req.FileURL = payload.UDiscURL
	}

	result, err := h.service.ParseScorecard(ctx, &req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.MessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          (*result.Failure).Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
	}

	payloadOut := &roundevents.ParsedScorecardPayloadV1{
		ImportID:                payload.ImportID,
		Source:                  payload.Source,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		EventMessageID:          payload.MessageID,
		ParsedData:              result.Success,
		AllowGuestPlayers:       payload.AllowGuestPlayers,
		OverwriteExistingScores: payload.OverwriteExistingScores,
	}

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardParsedForNormalizationV1, Payload: payloadOut}}, nil
}

// HandleScorecardParsedForNormalization takes raw parsed data and structures it.
func (h *RoundHandlers) HandleScorecardParsedForNormalization(ctx context.Context, payload *roundevents.ParsedScorecardPayloadV1) ([]handlerwrapper.Result, error) {
	meta := roundtypes.Metadata{
		ImportID:       payload.ImportID,
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		Source:         payload.Source,
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
			attr.Any("failure", *result.Failure),
			attr.String("error_msg", (*result.Failure).Error()),
		)
	}

	return mapOperationResult(result.Map(
		func(s *roundtypes.NormalizedScorecard) any {
			return &roundevents.ScorecardNormalizedPayloadV1{
				ImportID:                payload.ImportID,
				Source:                  payload.Source,
				GuildID:                 payload.GuildID,
				RoundID:                 payload.RoundID,
				UserID:                  payload.UserID,
				ChannelID:               payload.ChannelID,
				EventMessageID:          payload.EventMessageID,
				Normalized:              *s,
				AllowGuestPlayers:       payload.AllowGuestPlayers,
				OverwriteExistingScores: payload.OverwriteExistingScores,
				Timestamp:               time.Now().UTC(),
			}
		},
		func(f error) any {
			return &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.EventMessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          f.Error(),
				Timestamp:      time.Now().UTC(),
			}
		},
	), roundevents.ScorecardNormalizedV1, roundevents.ImportFailedV1), nil
}

// HandleScorecardNormalized handles the ingestion/matching of names.
func (h *RoundHandlers) HandleScorecardNormalized(ctx context.Context, payload *roundevents.ScorecardNormalizedPayloadV1) ([]handlerwrapper.Result, error) {
	req := roundtypes.ImportIngestScorecardInput{
		ImportID:                payload.ImportID,
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		Source:                  payload.Source,
		UserID:                  payload.UserID,
		ChannelID:               payload.ChannelID,
		EventMessageID:          payload.EventMessageID,
		NormalizedData:          payload.Normalized,
		AllowGuestPlayers:       payload.AllowGuestPlayers,
		OverwriteExistingScores: payload.OverwriteExistingScores,
	}

	result, err := h.service.IngestNormalizedScorecard(ctx, req)
	if err != nil {
		return nil, err
	}
	return mapOperationResult(result.Map(
		func(s *roundtypes.IngestScorecardResult) any {
			return &roundevents.ImportCompletedPayloadV1{
				ImportID:                s.ImportID,
				Source:                  payload.Source,
				GuildID:                 s.GuildID,
				RoundID:                 s.RoundID,
				Scores:                  s.Scores,
				EventMessageID:          s.EventMessageID,
				AllowGuestPlayers:       payload.AllowGuestPlayers,
				OverwriteExistingScores: payload.OverwriteExistingScores,
				ParScores:               s.ParScores,
				Timestamp:               s.Timestamp,
			}
		},
		func(f error) any {
			return &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.EventMessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          f.Error(),
				Timestamp:      time.Now().UTC(),
			}
		},
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
			UserID:     s.UserID,
			RawName:    s.RawName,
			Score:      int(s.Score),
			TeamID:     s.TeamID,
			HoleScores: s.HoleScores,
			IsDNF:      s.IsDNF,
		}
	}

	req := roundtypes.ImportApplyScoresInput{
		GuildID:                 payload.GuildID,
		RoundID:                 payload.RoundID,
		ImportID:                payload.ImportID,
		Source:                  payload.Source,
		Scores:                  scores,
		AllowGuestPlayers:       payload.AllowGuestPlayers,
		OverwriteExistingScores: payload.OverwriteExistingScores,
		ParScores:               payload.ParScores,
	}

	res, err := h.service.ApplyImportedScores(ctx, req)
	if err != nil {
		return nil, err
	}

	if res.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: roundevents.ImportFailedV1,
			Payload: &roundevents.ImportFailedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				EventMessageID: payload.EventMessageID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				Error:          (*res.Failure).Error(),
				Timestamp:      time.Now().UTC(),
			},
		}}, nil
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

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundParticipantScoreUpdatedV1, success.GuildID)

	return results, nil
}

func (h *RoundHandlers) ensureAdminRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) error {
	if userID == "" {
		return fmt.Errorf("admin scorecard upload requires a valid user_id")
	}

	if err := h.requireAdminRole(ctx, guildID, userID); err == nil {
		return nil
	}

	parsedUUID, parseErr := uuid.Parse(string(userID))
	if parseErr != nil {
		return fmt.Errorf("failed to verify admin role: user %q is not a discord ID and is not a valid UUID", userID)
	}

	discordID, resolveErr := h.userService.GetDiscordIDByUUID(ctx, parsedUUID)
	if resolveErr != nil {
		return fmt.Errorf("failed to verify admin role: could not resolve UUID %q to discord user: %w", userID, resolveErr)
	}

	if err := h.requireAdminRole(ctx, guildID, discordID); err != nil {
		return err
	}

	return nil
}

func (h *RoundHandlers) requireAdminRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) error {
	roleResult, err := h.userService.GetUserRole(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to verify admin role: %w", err)
	}
	if roleResult.Failure != nil {
		return fmt.Errorf("failed to verify admin role: %w", *roleResult.Failure)
	}
	if roleResult.Success == nil || *roleResult.Success != sharedtypes.UserRoleAdmin {
		return fmt.Errorf("admin role required for manual scorecard upload")
	}

	return nil
}
