package scoreservice

import (
	"context"

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
		existingScores, err := s.repo.GetScoresForRound(ctx, guildID, roundID)
		if err != nil {
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  "failed to check existing scores",
				},
			}, nil
		}

		if len(existingScores) > 0 && !overwrite {
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  "SCORES_ALREADY_EXIST",
				},
			}, nil
		}

		// Process scores for storage (aggregating teams, tagging, etc.)
		processedScores, err := s.ProcessScoresForStorage(ctx, guildID, roundID, scores)
		if err != nil {
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  err.Error(),
				},
			}, nil
		}

		// Record tag mappings
		tagMappings := make([]sharedtypes.TagMapping, 0, len(processedScores))
		for _, scoreInfo := range processedScores {
			if scoreInfo.TagNumber != nil {
				tagMappings = append(tagMappings, sharedtypes.TagMapping{
					DiscordID: scoreInfo.UserID,
					TagNumber: *scoreInfo.TagNumber,
				})
				s.metrics.RecordPlayerTag(ctx, roundID, scoreInfo.UserID, scoreInfo.TagNumber)
			}
		}

		// Log processed scores to the repository
		if err := s.repo.LogScores(ctx, guildID, roundID, processedScores, "auto"); err != nil {
			return ScoreOperationResult{
				Failure: &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Reason:  err.Error(),
				},
			}, nil
		}

		s.metrics.RecordScoreProcessingSuccess(ctx, roundID)

		return ScoreOperationResult{
			Success: &sharedevents.ProcessRoundScoresSucceededPayloadV1{
				GuildID:     guildID,
				RoundID:     roundID,
				TagMappings: tagMappings,
			},
		}, nil
	})
}
