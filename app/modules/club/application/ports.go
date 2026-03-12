package clubservice

import (
	"context"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
)

// ChallengeQueueService schedules and cancels challenge expiry jobs.
type ChallengeQueueService interface {
	ScheduleOpenExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	ScheduleAcceptedExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	CancelOpenExpiry(ctx context.Context, challengeID uuid.UUID) error
	CancelChallengeJobs(ctx context.Context, challengeID uuid.UUID) error
	HealthCheck(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ChallengeTagReader exposes the leaderboard tag list required for challenge validation and enrichment.
type ChallengeTagReader interface {
	GetTagList(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error)
}

// ChallengeRoundReader exposes the round read operations needed for round-link validation.
type ChallengeRoundReader interface {
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error)
}
