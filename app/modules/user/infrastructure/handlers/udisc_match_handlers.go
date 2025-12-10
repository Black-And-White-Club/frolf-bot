package userhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleScorecardParsed consumes a parsed scorecard and emits match confirmation or confirmed events.
func (h *UserHandlers) HandleScorecardParsed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScorecardParsed",
		&roundevents.ParsedScorecardPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scorecard := payload.(*roundevents.ParsedScorecardPayload)

			h.logger.InfoContext(ctx, "Received parsed scorecard for matching",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", scorecard.ImportID),
				attr.String("guild_id", string(scorecard.GuildID)),
				attr.String("round_id", scorecard.RoundID.String()),
			)

			result, err := h.userService.MatchParsedScorecard(ctx, *scorecard)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to match parsed scorecard",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to match parsed scorecard: %w", err)
			}

			if result.Failure != nil {
				return nil, fmt.Errorf("matching returned failure: %v", result.Failure)
			}

			switch v := result.Success.(type) {
			case *userevents.UDiscMatchConfirmationRequiredPayload:
				msgOut, err := h.helpers.CreateResultMessage(msg, v, userevents.UserUDiscMatchConfirmationRequired)
				if err != nil {
					return nil, fmt.Errorf("failed to create confirmation required message: %w", err)
				}
				return []*message.Message{msgOut}, nil
			case *userevents.UDiscMatchConfirmedPayload:
				msgOut, err := h.helpers.CreateResultMessage(msg, v, userevents.UserUDiscMatchConfirmed)
				if err != nil {
					return nil, fmt.Errorf("failed to create match confirmed message: %w", err)
				}
				return []*message.Message{msgOut}, nil
			default:
				return nil, fmt.Errorf("unexpected success payload type: %T", v)
			}
		},
	)(msg)
}
