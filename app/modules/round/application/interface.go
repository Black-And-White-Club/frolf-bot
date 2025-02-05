package roundservice

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundService defines the interface for the round service.
type Service interface {
	// Create Round
	ValidateRoundRequest(ctx context.Context, msg *message.Message) error
	ParseDateTime(ctx context.Context, msg *message.Message) error
	StoreRound(ctx context.Context, msg *message.Message) error
	ScheduleRoundEvents(ctx context.Context, msg *message.Message) error
	PublishRoundCreated(ctx context.Context, msg *message.Message) error

	// Update Round
	ValidateRoundUpdateRequest(ctx context.Context, msg *message.Message) error
	GetRound(ctx context.Context, msg *message.Message) error
	UpdateRoundEntity(ctx context.Context, msg *message.Message) error
	StoreRoundUpdate(ctx context.Context, msg *message.Message) error
	PublishRoundUpdated(ctx context.Context, msg *message.Message) error

	// Delete Round
	ValidateRoundDeleteRequest(ctx context.Context, msg *message.Message) error
	CheckRoundExists(ctx context.Context, msg *message.Message) error
	CheckUserAuthorization(ctx context.Context, msg *message.Message) error
	UserRoleCheckResult(ctx context.Context, msg *message.Message) error
	DeleteRound(ctx context.Context, msg *message.Message) error

	// Start Round
	ProcessRoundStart(msg *message.Message) error

	// Join Round
	ValidateParticipantJoinRequest(ctx context.Context, msg *message.Message) error
	CheckParticipantTag(ctx context.Context, msg *message.Message) error
	ParticipantTagFound(ctx context.Context, msg *message.Message) error
	ParticipantTagNotFound(ctx context.Context, msg *message.Message) error

	// Score Round
	ValidateScoreUpdateRequest(ctx context.Context, msg *message.Message) error
	UpdateParticipantScore(ctx context.Context, msg *message.Message) error
	CheckAllScoresSubmitted(ctx context.Context, msg *message.Message) error

	// Finalize Round
	FinalizeRound(ctx context.Context, msg *message.Message) error
	NotifyScoreModule(ctx context.Context, msg *message.Message) error

	// Tag Retrieval
	RequestTagNumber(ctx context.Context, msg *message.Message) error
	TagNumberRequest(ctx context.Context, msg *message.Message) error
	TagNumberResponse(ctx context.Context, msg *message.Message) error

	// Round Reminder
	ProcessRoundReminder(msg *message.Message) error
}
