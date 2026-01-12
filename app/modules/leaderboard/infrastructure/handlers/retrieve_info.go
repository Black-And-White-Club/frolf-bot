package leaderboardhandlers

import (
	"context"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(
	ctx context.Context,
	payload *leaderboardevents.GetLeaderboardRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received GetLeaderboardRequest event",
		attr.ExtractCorrelationID(ctx),
	)

	result, err := h.leaderboardService.GetLeaderboard(ctx, payload.GuildID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get leaderboard",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*leaderboardevents.GetLeaderboardFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}
		h.logger.InfoContext(ctx, "Get leaderboard failed",
			attr.ExtractCorrelationID(ctx),
			attr.Any("failure_payload", failurePayload),
		)
		return []handlerwrapper.Result{
			{Topic: leaderboardevents.GetLeaderboardFailedV1, Payload: failurePayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*leaderboardevents.GetLeaderboardResponsePayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}
		h.logger.InfoContext(ctx, "Get leaderboard successful",
			attr.ExtractCorrelationID(ctx),
		)
		return []handlerwrapper.Result{
			{Topic: leaderboardevents.GetLeaderboardResponseV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("get leaderboard service returned unexpected result")
}

// HandleGetTagByUserIDRequest handles the GetTagByUserIDRequest event.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(
	ctx context.Context,
	payload *sharedevents.DiscordTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received DiscordTagLookUpByUserIDRequest event",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
	)

	result, err := h.leaderboardService.GetTagByUserID(ctx, sharedtypes.GuildID(payload.GuildID), payload.UserID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed during GetTagByUserID service call",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*sharedevents.DiscordTagLookupResultPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}

		eventType := sharedevents.LeaderboardTagLookupNotFoundV1
		if successPayload.Found && successPayload.TagNumber != nil {
			eventType = sharedevents.LeaderboardTagLookupSucceededV1
			h.logger.InfoContext(ctx, "Tag lookup successful: Tag found",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(successPayload.UserID)),
				attr.Int("tag_number", int(*successPayload.TagNumber)),
			)
		} else {
			h.logger.InfoContext(ctx, "Tag lookup completed: Tag not found (Business Outcome)",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(successPayload.UserID)),
			)
		}

		return []handlerwrapper.Result{
			{Topic: eventType, Payload: successPayload},
		}, nil

	} else if result.Failure != nil {
		failurePayload, ok := result.Failure.(*sharedevents.DiscordTagLookupFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}

		h.logger.InfoContext(ctx, "GetTagByUserID service returned business failure",
			attr.ExtractCorrelationID(ctx),
			attr.String("reason", failurePayload.Reason),
		)

		return []handlerwrapper.Result{
			{Topic: sharedevents.LeaderboardTagLookupFailedV1, Payload: failurePayload},
		}, nil

	} else if result.Error != nil {
		h.logger.ErrorContext(ctx, "GetTagByUserID service returned system error within result",
			attr.ExtractCorrelationID(ctx),
			attr.Error(result.Error),
		)

		handlerFailurePayload := sharedevents.DiscordTagLookupFailedPayloadV1{
			UserID: payload.UserID,
			Reason: result.Error.Error(),
		}

		return []handlerwrapper.Result{
			{Topic: sharedevents.LeaderboardTagLookupFailedV1, Payload: &handlerFailurePayload},
		}, nil
	}

	return nil, errors.New("get tag by user ID service returned unexpected result")
}

// HandleRoundGetTagRequest handles the RoundTagLookupRequest event.
func (h *LeaderboardHandlers) HandleRoundGetTagRequest(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received RoundTagLookupRequest event",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.RoundID("round_id", payload.RoundID),
		attr.String("response", string(payload.Response)),
		attr.Any("joined_late", payload.JoinedLate),
	)

	result, err := h.leaderboardService.RoundGetTagByUserID(ctx, sharedtypes.GuildID(payload.GuildID), *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed during RoundGetTagByUserID service call",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	if result.Success != nil {
		responsePayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}

		eventType := sharedevents.RoundTagLookupNotFoundV1
		if responsePayload.Found {
			eventType = sharedevents.RoundTagLookupFoundV1
			h.logger.InfoContext(ctx, "Tag lookup successful: Tag found",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(responsePayload.UserID)),
				attr.Int("tag_number", int(*responsePayload.TagNumber)),
			)
		} else {
			h.logger.InfoContext(ctx, "Tag lookup completed: Tag not found (Business Outcome)",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(responsePayload.UserID)),
			)
		}

		return []handlerwrapper.Result{
			{Topic: eventType, Payload: responsePayload},
		}, nil

	} else if result.Failure != nil {
		failurePayload, ok := result.Failure.(*sharedevents.RoundTagLookupFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}

		h.logger.InfoContext(ctx, "RoundGetTagByUserID service returned business failure",
			attr.ExtractCorrelationID(ctx),
			attr.String("reason", failurePayload.Reason),
		)

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.GetTagNumberFailedV1, Payload: failurePayload},
		}, nil

	} else if result.Error != nil {
		h.logger.ErrorContext(ctx, "RoundGetTagByUserID service returned system error within result",
			attr.ExtractCorrelationID(ctx),
			attr.Error(result.Error),
		)

		failurePayload := sharedevents.RoundTagLookupFailedPayloadV1{
			UserID:  payload.UserID,
			RoundID: payload.RoundID,
			Reason:  result.Error.Error(),
		}

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.GetTagNumberFailedV1, Payload: &failurePayload},
		}, nil
	}

	return nil, errors.New("round get tag service returned unexpected result")
}
