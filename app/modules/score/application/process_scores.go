package scoreservice

import (
	"context"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundScores processes scores received from the round module using the service wrapper.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) (ScoreOperationResult, error) {
	s.metrics.RecordScoreProcessingAttempt(ctx, roundID)
	roundIDAttr := attr.RoundID("round_id", roundID)

	s.logger.InfoContext(ctx, "Starting to process round scores",
		attr.ExtractCorrelationID(ctx),
		roundIDAttr,
		attr.Int("num_scores", len(scores)),
	)

	return s.serviceWrapper(ctx, "ProcessRoundScores", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		processedScores, err := s.ProcessScoresForStorage(ctx, roundID, scores)
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
			return ScoreOperationResult{Error: err}, err
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
		if err := s.ScoreDB.LogScores(ctx, roundID, processedScores, "auto"); err != nil {
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
			return ScoreOperationResult{Error: err}, err
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
		return ScoreOperationResult{Success: tagMappingPayload}, nil
	})
}
