package scoreservice

import (
	"context"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundScores processes scores received from the round module.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) (ScoreOperationResult, error) {
	s.metrics.RecordScoreProcessingAttempt(roundID)
	correlationID := attr.ExtractCorrelationID(ctx)
	roundIDAttr := attr.RoundID("round_id", roundID)

	s.logger.Info("Starting to process round scores",
		attr.LogAttr(correlationID),
		roundIDAttr,
		attr.Int("num_scores", len(scores)),
	)

	// Call the serviceWrapper
	return s.serviceWrapper(ctx, "ProcessRoundScores", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		ctx, span := s.tracer.StartSpan(ctx, "ProcessRoundScores.DatabaseOperation", nil)
		defer span.End()

		// Process scores for storage
		processedScores, err := s.ProcessScoresForStorage(ctx, roundID, scores)
		if err != nil {
			s.logger.Error("Failed to process scores for storage",
				attr.LogAttr(correlationID),
				roundIDAttr,
				attr.Error(err),
			)
			return ScoreOperationResult{Error: err}, err
		}

		// Pre-allocate capacity for tag mappings based on scores length
		tagMappings := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(processedScores))

		// Extract tag mappings in a single pass
		extractStartTime := time.Now()
		for _, scoreInfo := range processedScores {
			if scoreInfo.TagNumber != nil {
				tagMappings[scoreInfo.UserID] = *scoreInfo.TagNumber
				s.metrics.RecordPlayerTag(roundID, scoreInfo.UserID, scoreInfo.TagNumber)
			}
		}

		// Record metrics for operation
		s.metrics.RecordOperationAttempt("ExtractTagInformation", roundID)
		s.metrics.RecordOperationDuration("ExtractTagInformation", time.Since(extractStartTime).Seconds())

		// Log to database with timing
		dbStart := time.Now()
		if err := s.ScoreDB.LogScores(ctx, roundID, processedScores, "auto"); err != nil {
			s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())
			s.logger.Error("Failed to log scores to database",
				attr.LogAttr(correlationID),
				roundIDAttr,
				attr.Error(err),
			)
			return ScoreOperationResult{Error: err}, err
		}
		s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())
		s.metrics.RecordScoreProcessingSuccess(roundID)

		// Pre-allocate the result slice with exact capacity
		tagMappingPayload := make([]sharedtypes.TagMapping, 0, len(tagMappings))

		// Build return payload
		for discordID, tagNumber := range tagMappings {
			tagMappingPayload = append(tagMappingPayload, sharedtypes.TagMapping{
				DiscordID: discordID,
				TagNumber: tagNumber,
			})
		}

		return ScoreOperationResult{
			Success: &scoreevents.ProcessRoundScoresSuccessPayload{
				RoundID:     roundID,
				TagMappings: tagMappingPayload,
			},
		}, nil
	})
}
