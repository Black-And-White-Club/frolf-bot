package scoreservice

import (
	"context"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundScores processes scores received from the round module using the service wrapper.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (ScoreOperationResult, error) {
	s.metrics.RecordScoreProcessingAttempt(ctx, roundID)
	roundIDAttr := attr.RoundID("round_id", roundID)

	s.logger.InfoContext(ctx, "Starting to process round scores",
		attr.ExtractCorrelationID(ctx),
		roundIDAttr,
		attr.Int("num_scores", len(scores)),
		attr.String("guild_id", string(guildID)),
		attr.Bool("overwrite", overwrite),
	)

	return s.serviceWrapper(ctx, "ProcessRoundScores", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		// Check if scores already exist for this round
		existingScores, err := s.ScoreDB.GetScoresForRound(ctx, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to check existing scores",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
				attr.Error(err),
			)
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  "failed to check existing scores",
				},
			}, nil
		}

		if len(existingScores) > 0 && !overwrite {
			s.logger.WarnContext(ctx, "Scores already exist for round, overwrite not requested",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
			)
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  "SCORES_ALREADY_EXIST",
				},
			}, nil
		}

		processedScores, err := s.ProcessScoresForStorage(ctx, guildID, roundID, scores)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to process scores for storage",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
				attr.Error(err),
			)
			s.logger.InfoContext(ctx, "Service returning: Error processing scores for storage",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
				attr.Error(err),
			)
			// Return a failure payload for business logic errors from ProcessScoresForStorage
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  err.Error(),
				},
			}, nil // Return nil error to indicate handled business failure
		}

		tagMappings := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(processedScores))

		extractStartTime := time.Now()
		for _, scoreInfo := range processedScores {
			if scoreInfo.TagNumber != nil {
				tagMappings[scoreInfo.UserID] = *scoreInfo.TagNumber
				s.metrics.RecordPlayerTag(ctx, roundID, scoreInfo.UserID, scoreInfo.TagNumber)
			}
		}

		s.metrics.RecordOperationAttempt(ctx, "ExtractTagInformation", roundID)
		s.metrics.RecordOperationDuration(ctx, "ExtractTagInformation", time.Duration(time.Since(extractStartTime).Seconds()))

		dbStart := time.Now()
		if err := s.ScoreDB.LogScores(ctx, guildID, roundID, processedScores, "auto"); err != nil {
			s.metrics.RecordDBQueryDuration(ctx, time.Duration(time.Since(dbStart).Seconds()))
			s.logger.ErrorContext(ctx, "Failed to log scores to database",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
				attr.Error(err),
			)
			s.logger.InfoContext(ctx, "Service returning: Error logging scores to database",
				attr.ExtractCorrelationID(ctx),
				roundIDAttr,
				attr.Error(err),
			)
			// Return a failure payload for business logic errors from LogScores
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  err.Error(),
				},
			}, nil // Return nil error to indicate handled business failure
		}
		s.metrics.RecordDBQueryDuration(ctx, time.Duration(time.Since(dbStart).Seconds()))
		s.metrics.RecordScoreProcessingSuccess(ctx, roundID)

		tagMappingPayload := make([]sharedtypes.TagMapping, 0, len(tagMappings))
		for discordID, tagNumber := range tagMappings {
			tagMappingPayload = append(tagMappingPayload, sharedtypes.TagMapping{
				DiscordID: discordID,
				TagNumber: tagNumber,
			})
		}

		s.logger.InfoContext(ctx, "Service returning: Success with tag mappings",
			attr.ExtractCorrelationID(ctx),
			roundIDAttr,
			attr.Int("num_tag_mappings", len(tagMappingPayload)),
		)
		// Wrap the tagMappingPayload in the expected success struct
		return ScoreOperationResult{
			Success: &sharedevents.ProcessRoundScoresSucceededPayloadV1{
				GuildID:     guildID,
				RoundID:     roundID,
				TagMappings: tagMappingPayload,
			},
		}, nil
	})
}
