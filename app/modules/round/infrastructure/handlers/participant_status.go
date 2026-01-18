package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Helper methods for clean type assertions within the handlers.
func (h *RoundHandlers) toJoined(v any) (*roundevents.ParticipantJoinedPayloadV1, error) {
	p, ok := v.(*roundevents.ParticipantJoinedPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "unexpected payload type: expected ParticipantJoinedPayloadV1"}
	}
	return p, nil
}

func (h *RoundHandlers) toJoinError(v any) (*roundevents.RoundParticipantJoinErrorPayloadV1, error) {
	p, ok := v.(*roundevents.RoundParticipantJoinErrorPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "unexpected payload type: expected RoundParticipantJoinErrorPayloadV1"}
	}
	return p, nil
}

func (h *RoundHandlers) toTagLookupRequest(v any) (*sharedevents.RoundTagLookupRequestedPayloadV1, error) {
	p, ok := v.(*sharedevents.RoundTagLookupRequestedPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "unexpected payload type: expected RoundTagLookupRequestedPayloadV1 (shared payload)"}
	}
	return p, nil
}

func (h *RoundHandlers) toJoinRequest(v any) (*roundevents.ParticipantJoinRequestPayloadV1, error) {
	p, ok := v.(*roundevents.ParticipantJoinRequestPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "unexpected payload type: expected ParticipantJoinRequestPayloadV1"}
	}
	return p, nil
}

// HandleParticipantJoinRequest processes the initial rsvp/join intent from a user.
func (h *RoundHandlers) HandleParticipantJoinRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.CheckParticipantStatus(ctx, *payload)
	if err != nil {
		return nil, err
	}

	// Safety check: ensure GuildID is propagated to outgoing payloads.
	if result.Success != nil {
		switch s := result.Success.(type) {
		case *roundevents.ParticipantJoinValidationRequestPayloadV1:
			if s.GuildID == "" {
				s.GuildID = payload.GuildID
			}
		case *roundevents.ParticipantRemovalRequestPayloadV1:
			if s.GuildID == "" {
				s.GuildID = payload.GuildID
			}
		}
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantStatusCheckErrorV1, Payload: result.Failure},
		}, nil
	}

	switch successPayload := result.Success.(type) {
	case *roundevents.ParticipantRemovalRequestPayloadV1:
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantRemovalRequestedV1, Payload: successPayload},
		}, nil

	case *roundevents.ParticipantJoinValidationRequestPayloadV1:
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantJoinValidationRequestedV1, Payload: successPayload},
		}, nil

	default:
		return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from CheckParticipantStatus"}
	}
}

// HandleParticipantJoinValidationRequest validates if a user is allowed to join (e.g. check if round is full).
func (h *RoundHandlers) HandleParticipantJoinValidationRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinValidationRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ValidateParticipantJoinRequest(ctx, roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:  payload.RoundID,
		UserID:   payload.UserID,
		Response: payload.Response,
		GuildID:  payload.GuildID,
	})
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantJoinErrorV1, Payload: result.Failure},
		}, nil
	}

	// If user declined, bypass tag lookup and go straight to update.
	if payload.Response == roundtypes.ResponseDecline {
		updateRequest, err := h.toJoinRequest(result.Success)
		if err != nil {
			return nil, err
		}
		if updateRequest.GuildID == "" {
			updateRequest.GuildID = payload.GuildID
		}
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantStatusUpdateRequestedV1, Payload: updateRequest},
		}, nil
	}

	// If Accept/Tentative, request a tag lookup from the leaderboard service.
	tagLookupRequest, err := h.toTagLookupRequest(result.Success)
	if err != nil {
		return nil, err
	}

	tagLookupRequest.Response = payload.Response
	tagLookupRequest.OriginalResponse = payload.Response

	if tagLookupRequest.GuildID == "" {
		tagLookupRequest.GuildID = payload.GuildID
	}

	// Publish the shared payload directly (we already accept the shared shape in `toTagLookupRequest`).
	return []handlerwrapper.Result{
		{Topic: sharedevents.RoundTagLookupRequestedV1, Payload: tagLookupRequest},
	}, nil
}

// HandleParticipantStatusUpdateRequest applies the final participant state to the database.
func (h *RoundHandlers) HandleParticipantStatusUpdateRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.UpdateParticipantStatus(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantJoinErrorV1, Payload: result.Failure},
		}, nil
	}

	return []handlerwrapper.Result{
		{Topic: roundevents.RoundParticipantJoinedV1, Payload: result.Success},
	}, nil
}

// HandleParticipantRemovalRequest handles the logic for removing a participant from a round.
func (h *RoundHandlers) HandleParticipantRemovalRequest(
	ctx context.Context,
	payload *roundevents.ParticipantRemovalRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ParticipantRemoval(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantRemovalErrorV1, Payload: result.Failure},
		}, nil
	}

	return []handlerwrapper.Result{
		{Topic: roundevents.RoundParticipantRemovedV1, Payload: result.Success},
	}, nil
}

// HandleTagNumberFound processes a successful tag lookup and proceeds to update status.
func (h *RoundHandlers) HandleTagNumberFound(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupResultPayloadV1,
) ([]handlerwrapper.Result, error) {
	updatePayload := &roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		TagNumber:  payload.TagNumber,
		JoinedLate: payload.OriginalJoinedLate,
		Response:   payload.OriginalResponse,
		GuildID:    payload.GuildID,
	}
	return h.handleParticipantUpdate(ctx, updatePayload)
}

// HandleTagNumberNotFound processes a lookup where no tag was found and proceeds to update status.
func (h *RoundHandlers) HandleTagNumberNotFound(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupResultPayloadV1,
) ([]handlerwrapper.Result, error) {
	updatePayload := &roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		TagNumber:  nil,
		JoinedLate: payload.OriginalJoinedLate,
		Response:   payload.OriginalResponse,
		GuildID:    payload.GuildID,
	}
	return h.handleParticipantUpdate(ctx, updatePayload)
}

// HandleParticipantDeclined handles the declined state directly.
func (h *RoundHandlers) HandleParticipantDeclined(
	ctx context.Context,
	payload *roundevents.ParticipantDeclinedPayloadV1,
) ([]handlerwrapper.Result, error) {
	updatePayload := &roundevents.ParticipantJoinRequestPayloadV1{
		GuildID:    payload.GuildID,
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		Response:   roundtypes.ResponseDecline,
		TagNumber:  nil,
		JoinedLate: nil,
	}
	return h.handleParticipantUpdate(ctx, updatePayload)
}

// HandleTagNumberLookupFailed handles technical failures in tag lookup.
func (h *RoundHandlers) HandleTagNumberLookupFailed(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupFailedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if (payload.RoundID == sharedtypes.RoundID{}) || payload.UserID == "" {
		return []handlerwrapper.Result{}, nil
	}

	updatePayload := &roundevents.ParticipantJoinRequestPayloadV1{
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		Response:  roundtypes.ResponseAccept,
		TagNumber: nil,
	}
	return h.handleParticipantUpdate(ctx, updatePayload)
}

// handleParticipantUpdate is an internal helper to finalize the status update via the service.
func (h *RoundHandlers) handleParticipantUpdate(
	ctx context.Context,
	updatePayload *roundevents.ParticipantJoinRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	updateResult, err := h.service.UpdateParticipantStatus(ctx, *updatePayload)
	if err != nil {
		return nil, err
	}

	if updateResult.Failure != nil {
		payload, err := h.toJoinError(updateResult.Failure)
		if err != nil {
			return nil, err
		}
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantJoinErrorV1, Payload: payload},
		}, nil
	}

	payload, err := h.toJoined(updateResult.Success)
	if err != nil {
		return nil, err
	}
	return []handlerwrapper.Result{
		{Topic: roundevents.RoundParticipantJoinedV1, Payload: payload},
	}, nil
}
