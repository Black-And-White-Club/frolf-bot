package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleParticipantJoinRequest processes the initial rsvp/join intent from a user.
func (h *RoundHandlers) HandleParticipantJoinRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.JoinRoundRequest{
		GuildID:    payload.GuildID,
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		Response:   payload.Response,
		TagNumber:  payload.TagNumber,
		JoinedLate: payload.JoinedLate,
	}

	result, err := h.service.CheckParticipantStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		results := []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantStatusCheckErrorV1,
				Payload: &roundevents.ParticipantStatusCheckErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					UserID:  payload.UserID,
					Error:   (*result.Failure).Error(),
				},
			},
		}
		return results, nil
	}

	checkResult := result.Success
	var results []handlerwrapper.Result
	switch (*checkResult).Action {
	case "VALIDATE":
		results = []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantJoinValidationRequestedV1,
				Payload: &roundevents.ParticipantJoinValidationRequestPayloadV1{
					GuildID:  (*checkResult).GuildID,
					RoundID:  (*checkResult).RoundID,
					UserID:   (*checkResult).UserID,
					Response: (*checkResult).Response,
				},
			},
		}
	case "REMOVE":
		results = []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantRemovalRequestedV1,
				Payload: &roundevents.ParticipantRemovalRequestPayloadV1{
					GuildID: (*checkResult).GuildID,
					RoundID: (*checkResult).RoundID,
					UserID:  (*checkResult).UserID,
				},
			},
		}
	default:
		return nil, sharedtypes.ValidationError{Message: "unexpected action from CheckParticipantStatus: " + (*checkResult).Action}
	}
	return results, nil
}

// HandleParticipantJoinValidationRequest validates if a user is allowed to join (e.g. check if round is full).
func (h *RoundHandlers) HandleParticipantJoinValidationRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinValidationRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.JoinRoundRequest{
		GuildID:  payload.GuildID,
		RoundID:  payload.RoundID,
		UserID:   payload.UserID,
		Response: payload.Response,
	}

	result, err := h.service.ValidateParticipantJoinRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		results := []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantJoinErrorV1,
				Payload: &roundevents.RoundParticipantJoinErrorPayloadV1{
					GuildID: payload.GuildID,
					Error:   (*result.Failure).Error(),
				},
			},
		}
		return results, nil
	}

	validatedReq := result.Success

	// If user declined, bypass tag lookup and go straight to update.
	if (*validatedReq).Response == roundtypes.ResponseDecline {
		updateRequest := &roundevents.ParticipantJoinRequestPayloadV1{
			GuildID:    (*validatedReq).GuildID,
			RoundID:    (*validatedReq).RoundID,
			UserID:     (*validatedReq).UserID,
			Response:   (*validatedReq).Response,
			TagNumber:  (*validatedReq).TagNumber,
			JoinedLate: (*validatedReq).JoinedLate,
		}
		results := []handlerwrapper.Result{
			{Topic: roundevents.RoundParticipantStatusUpdateRequestedV1, Payload: updateRequest},
		}
		return results, nil
	}

	// If Accept/Tentative, request a tag lookup from the leaderboard service.
	tagLookupRequest := &sharedevents.RoundTagLookupRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{
			GuildID: (*validatedReq).GuildID,
		},
		RoundID:          (*validatedReq).RoundID,
		UserID:           (*validatedReq).UserID,
		Response:         (*validatedReq).Response,
		OriginalResponse: (*validatedReq).Response,
		JoinedLate:       (*validatedReq).JoinedLate,
	}

	results := []handlerwrapper.Result{
		{Topic: sharedevents.RoundTagLookupRequestedV1, Payload: tagLookupRequest},
	}

	return results, nil
}

// HandleParticipantStatusUpdateRequest applies the final participant state to the database.
func (h *RoundHandlers) HandleParticipantStatusUpdateRequest(
	ctx context.Context,
	payload *roundevents.ParticipantJoinRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.JoinRoundRequest{
		GuildID:    payload.GuildID,
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		Response:   payload.Response,
		TagNumber:  payload.TagNumber,
		JoinedLate: payload.JoinedLate,
	}

	result, err := h.service.UpdateParticipantStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		results := []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantJoinErrorV1,
				Payload: &roundevents.RoundParticipantJoinErrorPayloadV1{
					GuildID:                payload.GuildID,
					Error:                  (*result.Failure).Error(),
					ParticipantJoinRequest: payload,
				},
			},
		}
		return results, nil
	}

	round := result.Success
	joinedPayload := h.createJoinedPayload(*round, payload.JoinedLate)

	results := []handlerwrapper.Result{
		{Topic: roundevents.RoundParticipantJoinedV1, Payload: joinedPayload},
	}

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundParticipantJoinedV1, payload.GuildID)

	return results, nil
}

// HandleParticipantRemovalRequest handles the logic for removing a participant from a round.
func (h *RoundHandlers) HandleParticipantRemovalRequest(
	ctx context.Context,
	payload *roundevents.ParticipantRemovalRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.JoinRoundRequest{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		UserID:  payload.UserID,
	}

	result, err := h.service.ParticipantRemoval(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		results := []handlerwrapper.Result{
			{
				Topic: roundevents.RoundParticipantRemovalErrorV1,
				Payload: &roundevents.ParticipantRemovalErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					UserID:  payload.UserID,
					Error:   (*result.Failure).Error(),
				},
			},
		}
		return results, nil
	}

	round := result.Success
	removedPayload := &roundevents.ParticipantRemovedPayloadV1{
		GuildID:        (*round).GuildID,
		RoundID:        (*round).ID,
		UserID:         payload.UserID,
		EventMessageID: (*round).EventMessageID,
	}
	removedPayload.AcceptedParticipants, removedPayload.DeclinedParticipants, removedPayload.TentativeParticipants = h.splitParticipants((*round).Participants)

	results := []handlerwrapper.Result{
		{Topic: roundevents.RoundParticipantRemovedV1, Payload: removedPayload},
	}
	return results, nil
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
	return h.HandleParticipantStatusUpdateRequest(ctx, updatePayload)
}

func (h *RoundHandlers) createJoinedPayload(round *roundtypes.Round, joinedLate *bool) *roundevents.ParticipantJoinedPayloadV1 {
	accepted, declined, tentative := h.splitParticipants(round.Participants)
	return &roundevents.ParticipantJoinedPayloadV1{
		GuildID:               round.GuildID,
		RoundID:               round.ID,
		AcceptedParticipants:  accepted,
		DeclinedParticipants:  declined,
		TentativeParticipants: tentative,
		EventMessageID:        round.EventMessageID,
		JoinedLate:            joinedLate,
	}
}

func (h *RoundHandlers) splitParticipants(participants []roundtypes.Participant) (accepted, declined, tentative []roundtypes.Participant) {
	for _, p := range participants {
		switch p.Response {
		case roundtypes.ResponseAccept:
			accepted = append(accepted, p)
		case roundtypes.ResponseDecline:
			declined = append(declined, p)
		case roundtypes.ResponseTentative:
			tentative = append(tentative, p)
		}
	}
	return
}
