package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// type-assert helpers: explicit, tiny, and readable.
func (h *RoundHandlers) toJoined(v any) (*roundevents.ParticipantJoinedPayload, error) {
	p, ok := v.(*roundevents.ParticipantJoinedPayload)
	if !ok {
		return nil, fmt.Errorf("expected *roundevents.ParticipantJoinedPayload, got %T", v)
	}
	return p, nil
}

func (h *RoundHandlers) toJoinError(v any) (*roundevents.RoundParticipantJoinErrorPayload, error) {
	p, ok := v.(*roundevents.RoundParticipantJoinErrorPayload)
	if !ok {
		return nil, fmt.Errorf("expected *roundevents.RoundParticipantJoinErrorPayload, got %T", v)
	}
	return p, nil
}

// func (h *RoundHandlers) toRemovalRequest(v any) (*roundevents.ParticipantRemovalRequestPayload, error) {
// 	p, ok := v.(*roundevents.ParticipantRemovalRequestPayload)
// 	if !ok {
// 		return nil, fmt.Errorf("expected *roundevents.ParticipantRemovalRequestPayload, got %T", v)
// 	}
// 	return p, nil
// }

// func (h *RoundHandlers) toValidationRequest(v any) (*roundevents.ParticipantJoinValidationRequestPayload, error) {
// 	p, ok := v.(*roundevents.ParticipantJoinValidationRequestPayload)
// 	if !ok {
// 		return nil, fmt.Errorf("expected *roundevents.ParticipantJoinValidationRequestPayload, got %T", v)
// 	}
// 	return p, nil
// }

func (h *RoundHandlers) toTagLookupRequest(v any) (*roundevents.TagLookupRequestPayload, error) {
	p, ok := v.(*roundevents.TagLookupRequestPayload)
	if !ok {
		return nil, fmt.Errorf("expected *roundevents.TagLookupRequestPayload, got %T", v)
	}
	return p, nil
}

func (h *RoundHandlers) toJoinRequest(v any) (*roundevents.ParticipantJoinRequestPayload, error) {
	p, ok := v.(*roundevents.ParticipantJoinRequestPayload)
	if !ok {
		return nil, fmt.Errorf("expected *roundevents.ParticipantJoinRequestPayload, got %T", v)
	}
	return p, nil
}

// createResultMessage wraps helpers.CreateResultMessage with consistent logging & error wrapping.
func (h *RoundHandlers) createResultMessage(
	ctx context.Context,
	inMsg *message.Message,
	payload any,
	topic string,
	logMsg string,
) (*message.Message, error) {
	out, err := h.helpers.CreateResultMessage(inMsg, payload, topic)
	if err != nil {
		h.logger.ErrorContext(ctx, logMsg,
			attr.CorrelationIDFromMsg(inMsg),
			attr.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", logMsg, err)
	}
	return out, nil
}

// warn when guild id is missing on outgoing payloads
// func (h *RoundHandlers) warnMissingGuild(ctx context.Context, msg *message.Message, payload any) {
// 	// many of your payload structs have GuildID string; we only log existence and keep behavior unchanged.
// 	// Do a simple switch to access GuildID fields where available
// 	switch p := payload.(type) {
// 	case *roundevents.ParticipantJoinRequestPayload:
// 		if p.GuildID == "" {
// 			h.logger.WarnContext(ctx, "Missing guild_id in ParticipantJoinRequestPayload",
// 				attr.CorrelationIDFromMsg(msg),
// 				attr.String("guild_id", ""),
// 			)
// 		}
// 	case *roundevents.ParticipantJoinValidationRequestPayload:
// 		if p.GuildID == "" {
// 			h.logger.WarnContext(ctx, "Missing guild_id in ParticipantJoinValidationRequestPayload",
// 				attr.CorrelationIDFromMsg(msg),
// 				attr.String("guild_id", ""),
// 			)
// 		}
// 	case *roundevents.ParticipantRemovalRequestPayload:
// 		if p.GuildID == "" {
// 			h.logger.WarnContext(ctx, "Missing guild_id in ParticipantRemovalRequestPayload",
// 				attr.CorrelationIDFromMsg(msg),
// 				attr.String("guild_id", ""),
// 			)
// 		}
// 	case *roundevents.TagLookupRequestPayload:
// 		if p.GuildID == "" {
// 			h.logger.WarnContext(ctx, "Missing guild_id in TagLookupRequestPayload",
// 				attr.CorrelationIDFromMsg(msg),
// 				attr.String("guild_id", ""),
// 			)
// 		}
// 	// add more cases if you want more coverage
// 	default:
// 		// nothing
// 	}
// }

// ----------------------------- Handlers (refactored) -----------------------------

// HandleParticipantJoinRequest handles the event when a participant initiates a join request (likely from Discord).
func (h *RoundHandlers) HandleParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantJoinRequest",
		&roundevents.ParticipantJoinRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantJoinRequestPayload := payload.(*roundevents.ParticipantJoinRequestPayload)

			h.logger.InfoContext(ctx, "Received ParticipantJoinRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantJoinRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantJoinRequestPayload.UserID)),
				attr.String("response", string(participantJoinRequestPayload.Response)),
				attr.String("guild_id", fmt.Sprintf("%v", participantJoinRequestPayload.GuildID)),
			)

			result, err := h.roundService.CheckParticipantStatus(ctx, *participantJoinRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during CheckParticipantStatus service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("CheckParticipantStatus service failed: %w", err)
			}

			// Patch: propagate guild_id into outgoing success payloads if missing
			if result.Success != nil {
				switch s := result.Success.(type) {
				case *roundevents.ParticipantJoinValidationRequestPayload:
					if s.GuildID == "" {
						s.GuildID = participantJoinRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in ParticipantJoinValidationRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(s.GuildID)),
						)
					}
				case *roundevents.ParticipantRemovalRequestPayload:
					if s.GuildID == "" {
						s.GuildID = participantJoinRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in ParticipantRemovalRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(s.GuildID)),
						)
					}
				}
			}

			// Handle failure
			if result.Failure != nil {
				h.logger.InfoContext(ctx, "CheckParticipantStatus returned failure",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, err := h.createResultMessage(
					ctx,
					msg,
					result.Failure,
					roundevents.RoundParticipantStatusCheckError,
					"Failed to create failure message after status check",
				)
				if err != nil {
					return nil, err
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle success
			if result.Success != nil {
				h.logger.InfoContext(ctx, "CheckParticipantStatus returned success",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload_type", fmt.Sprintf("%T", result.Success)),
					attr.Any("success_payload_content", result.Success),
				)

				switch successPayload := result.Success.(type) {
				case *roundevents.ParticipantRemovalRequestPayload:
					removalRequestMsg, err := h.createResultMessage(
						ctx, msg, successPayload, roundevents.RoundParticipantRemovalRequest,
						"Failed to create removal request message",
					)
					if err != nil {
						return nil, err
					}
					h.logger.InfoContext(ctx, "Publishing RoundParticipantRemovalRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", removalRequestMsg.UUID),
						attr.String("topic", roundevents.RoundParticipantRemovalRequest),
					)
					return []*message.Message{removalRequestMsg}, nil

				case *roundevents.ParticipantJoinValidationRequestPayload:
					validationRequestMsg, err := h.createResultMessage(
						ctx, msg, successPayload, roundevents.RoundParticipantJoinValidationRequest,
						"Failed to create validation request message",
					)
					if err != nil {
						return nil, err
					}
					h.logger.InfoContext(ctx, "Publishing RoundParticipantJoinValidationRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", validationRequestMsg.UUID),
						attr.String("topic", roundevents.RoundParticipantJoinValidationRequest),
					)
					return []*message.Message{validationRequestMsg}, nil

				default:
					h.logger.ErrorContext(ctx, "Unexpected success payload type from CheckParticipantStatus service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, fmt.Errorf("unexpected success payload type from CheckParticipantStatus: %T", result.Success)
				}
			}

			h.logger.ErrorContext(ctx, "CheckParticipantStatus service returned unexpected nil result",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("service_result", result),
			)
			return nil, fmt.Errorf("CheckParticipantStatus service returned unexpected nil result")
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantJoinValidationRequest
func (h *RoundHandlers) HandleParticipantJoinValidationRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantJoinValidationRequest",
		&roundevents.ParticipantJoinValidationRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.ParticipantJoinValidationRequestPayload)

			h.logger.InfoContext(ctx, "Received ParticipantJoinValidationRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),
				attr.String("user_id", string(p.UserID)),
				attr.String("response", string(p.Response)),
			)

			if p.UserID == "" {
				h.logger.ErrorContext(ctx, "ParticipantJoinValidationRequest has empty UserID",
					attr.CorrelationIDFromMsg(msg),
				)
				errorPayload := &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  p.RoundID,
						UserID:   p.UserID,
						Response: p.Response,
					},
					Error: "User ID cannot be empty",
				}
				failureMsg, err := h.createResultMessage(ctx, msg, errorPayload, roundevents.RoundParticipantJoinError, "Failed to create failure message for empty user ID")
				if err != nil {
					return nil, err
				}
				return []*message.Message{failureMsg}, nil
			}

			result, err := h.roundService.ValidateParticipantJoinRequest(ctx, roundevents.ParticipantJoinRequestPayload{
				RoundID:  p.RoundID,
				UserID:   p.UserID,
				Response: p.Response,
				GuildID:  p.GuildID,
			})
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during ValidateParticipantJoinRequest service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("ValidateParticipantJoinRequest service failed: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Participant join validation failed (service returned failure)",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, err := h.createResultMessage(ctx, msg, result.Failure, roundevents.RoundParticipantJoinError, "Failed to create failure message after validation")
				if err != nil {
					return nil, err
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success == nil {
				h.logger.ErrorContext(ctx, "ValidateParticipantJoinRequest service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("service_result", result),
				)
				return nil, fmt.Errorf("ValidateParticipantJoinRequest service returned unexpected nil result")
			}

			// success handling
			h.logger.InfoContext(ctx, "Participant join validation successful - inspecting success payload type BEFORE assertion",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("success_payload_type_before_assertion", fmt.Sprintf("%T", result.Success)),
				attr.Any("success_payload_content_before_assertion", result.Success),
			)

			if p.Response == roundtypes.ResponseDecline {
				updateRequest, err := h.toJoinRequest(result.Success)
				if err != nil {
					h.logger.ErrorContext(ctx, "Type assertion failed for DECLINE validation success payload",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, err
				}
				// ensure guild id
				if updateRequest.GuildID == "" {
					updateRequest.GuildID = p.GuildID
					h.logger.WarnContext(ctx, "Patched missing guild_id in DECLINE ParticipantJoinRequestPayload",
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", string(updateRequest.GuildID)),
					)
				}

				h.logger.InfoContext(ctx, "Validation successful for DECLINE - Preparing StatusUpdateRequest",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("update_request_payload", updateRequest),
				)

				successMsg, err := h.createResultMessage(ctx, msg, updateRequest, roundevents.RoundParticipantStatusUpdateRequest, "Failed to create StatusUpdateRequest message")
				if err != nil {
					return nil, err
				}
				h.logger.InfoContext(ctx, "Publishing RoundParticipantStatusUpdateRequest message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", successMsg.UUID),
					attr.String("topic", roundevents.RoundParticipantStatusUpdateRequest),
				)
				return []*message.Message{successMsg}, nil
			}

			// ACCEPT or TENTATIVE -> Tag lookup flow
			tagLookupRequest, err := h.toTagLookupRequest(result.Success)
			if err != nil {
				h.logger.ErrorContext(ctx, "Type assertion failed for non-DECLINE validation success payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
				)
				return nil, err
			}

			if tagLookupRequest.GuildID == "" {
				tagLookupRequest.GuildID = p.GuildID
				h.logger.WarnContext(ctx, "Patched missing guild_id in TagLookupRequestPayload",
					attr.CorrelationIDFromMsg(msg),
					attr.String("guild_id", string(tagLookupRequest.GuildID)),
				)
			}

			h.logger.InfoContext(ctx, "Validation successful for ACCEPT/TENTATIVE - Preparing TagLookupRequest",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("tag_lookup_request_payload", tagLookupRequest),
			)

			successMsg, err := h.createResultMessage(ctx, msg, tagLookupRequest, roundevents.LeaderboardGetTagNumberRequest, "Failed to create TagLookupRequest message")
			if err != nil {
				return nil, err
			}
			h.logger.InfoContext(ctx, "Publishing LeaderboardGetTagNumberRequest message",
				attr.CorrelationIDFromMsg(msg),
				attr.String("message_id", successMsg.UUID),
				attr.String("topic", roundevents.LeaderboardGetTagNumberRequest),
			)
			return []*message.Message{successMsg}, nil
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantStatusUpdateRequest handles participant status updates.
func (h *RoundHandlers) HandleParticipantStatusUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantStatusUpdateRequest",
		&roundevents.ParticipantJoinRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updateRequestPayload := payload.(*roundevents.ParticipantJoinRequestPayload)

			h.logger.InfoContext(ctx, "Received ParticipantStatusUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", updateRequestPayload.RoundID.String()),
				attr.String("user_id", string(updateRequestPayload.UserID)),
				attr.String("response", string(updateRequestPayload.Response)),
				attr.Any("tag_number", updateRequestPayload.TagNumber),
			)

			result, err := h.roundService.UpdateParticipantStatus(ctx, *updateRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during UpdateParticipantStatus service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("UpdateParticipantStatus service failed: %w", err)
			}

			// Failure path
			if result.Failure != nil {
				h.logger.InfoContext(ctx, "UpdateParticipantStatus returned failure",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, err := h.createResultMessage(ctx, msg, result.Failure, roundevents.RoundParticipantJoinError, "Failed to create failure message after status update")
				if err != nil {
					return nil, err
				}
				return []*message.Message{failureMsg}, nil
			}

			// Success path
			if result.Success == nil {
				h.logger.ErrorContext(ctx, "UpdateParticipantStatus service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("service_result", result),
				)
				return nil, fmt.Errorf("UpdateParticipantStatus service returned unexpected nil result")
			}

			h.logger.InfoContext(ctx, "UpdateParticipantStatus returned success",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("success_payload_type", fmt.Sprintf("%T", result.Success)),
			)

			// Log payload before creating Discord message
			h.logger.InfoContext(ctx, "ParticipantJoinedPayload content BEFORE publishing to Discord",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("payload_to_publish", result.Success),
			)

			successMsg, err := h.createResultMessage(ctx, msg, result.Success, roundevents.RoundParticipantJoined, "Failed to create RoundParticipantJoined message")
			if err != nil {
				return nil, err
			}

			if pj, ok := result.Success.(*roundevents.ParticipantJoinedPayload); ok {
				if pj.EventMessageID != "" && successMsg.Metadata.Get("discord_message_id") == "" {
					successMsg.Metadata.Set("discord_message_id", pj.EventMessageID)
				}
			}

			h.logger.InfoContext(ctx, "Publishing RoundParticipantJoined message for Discord",
				attr.CorrelationIDFromMsg(msg),
				attr.String("message_id", successMsg.UUID),
				attr.String("topic", roundevents.RoundParticipantJoined),
			)

			return []*message.Message{successMsg}, nil
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantRemovalRequest
func (h *RoundHandlers) HandleParticipantRemovalRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantRemovalRequest",
		&roundevents.ParticipantRemovalRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantRemovalRequestPayload := payload.(*roundevents.ParticipantRemovalRequestPayload)

			h.logger.InfoContext(ctx, "Received ParticipantRemovalRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantRemovalRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantRemovalRequestPayload.UserID)),
			)

			result, err := h.roundService.ParticipantRemoval(ctx, *participantRemovalRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ParticipantRemovalRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantRemovalRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Participant removal request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, err := h.createResultMessage(ctx, msg, result.Failure, roundevents.RoundParticipantRemovalError, "Failed to create failure message")
				if err != nil {
					return nil, err
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success == nil {
				h.logger.ErrorContext(ctx, "ParticipantRemoval service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
				)
				return nil, fmt.Errorf("ParticipantRemoval service returned unexpected nil result")
			}

			successMsg, err := h.createResultMessage(ctx, msg, result.Success, roundevents.RoundParticipantRemoved, "Failed to create success message")
			if err != nil {
				return nil, err
			}
			return []*message.Message{successMsg}, nil
		},
	)

	return wrappedHandler(msg)
}

// HandleTagNumberFound
func (h *RoundHandlers) HandleTagNumberFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberFound",
		&sharedevents.RoundTagLookupResultPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagLookupResultPayload := payload.(*sharedevents.RoundTagLookupResultPayload)

			h.logger.InfoContext(ctx, "Received RoundTagLookupFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagLookupResultPayload.RoundID.String()),
				attr.String("user_id", string(tagLookupResultPayload.UserID)),
				attr.Int("tag_number", int(*tagLookupResultPayload.TagNumber)),
				attr.String("original_response", string(tagLookupResultPayload.OriginalResponse)),
				attr.Any("original_joined_late", tagLookupResultPayload.OriginalJoinedLate),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    tagLookupResultPayload.RoundID,
				UserID:     tagLookupResultPayload.UserID,
				TagNumber:  tagLookupResultPayload.TagNumber,
				JoinedLate: tagLookupResultPayload.OriginalJoinedLate,
				Response:   tagLookupResultPayload.OriginalResponse,
				GuildID:    tagLookupResultPayload.GuildID,
			}

			h.logger.InfoContext(ctx, "DEBUG: updatePayload before handleParticipantUpdate",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", fmt.Sprintf("%v", updatePayload.GuildID)),
				attr.String("round_id", updatePayload.RoundID.String()),
				attr.String("user_id", string(updatePayload.UserID)),
				attr.Any("tag_number", updatePayload.TagNumber),
				attr.Any("joined_late", updatePayload.JoinedLate),
				attr.String("response", string(updatePayload.Response)),
			)

			return h.handleParticipantUpdate(ctx, msg, updatePayload, tagLookupResultPayload.OriginalResponse)
		},
	)

	return wrappedHandler(msg)
}

// HandleTagNumberNotFound
func (h *RoundHandlers) HandleTagNumberNotFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberNotFound",
		&sharedevents.RoundTagLookupResultPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagLookupResultPayload := payload.(*sharedevents.RoundTagLookupResultPayload)

			h.logger.InfoContext(ctx, "Received RoundTagLookupNotFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagLookupResultPayload.RoundID.String()),
				attr.String("user_id", string(tagLookupResultPayload.UserID)),
				attr.Bool("found_in_payload", tagLookupResultPayload.Found),
				attr.String("error_in_payload", tagLookupResultPayload.Error),
				attr.String("original_response", string(tagLookupResultPayload.OriginalResponse)),
				attr.Any("original_joined_late", tagLookupResultPayload.OriginalJoinedLate),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    tagLookupResultPayload.RoundID,
				UserID:     tagLookupResultPayload.UserID,
				TagNumber:  nil,
				JoinedLate: tagLookupResultPayload.OriginalJoinedLate,
				Response:   tagLookupResultPayload.OriginalResponse,
				GuildID:    tagLookupResultPayload.GuildID,
			}

			return h.handleParticipantUpdate(ctx, msg, updatePayload, tagLookupResultPayload.OriginalResponse)
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantDeclined
func (h *RoundHandlers) HandleParticipantDeclined(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantDeclined",
		&roundevents.ParticipantDeclinedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			d := payload.(*roundevents.ParticipantDeclinedPayload)

			h.logger.InfoContext(ctx, "Received ParticipantDeclined event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", d.RoundID.String()),
				attr.String("user_id", string(d.UserID)),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    d.RoundID,
				UserID:     d.UserID,
				Response:   roundtypes.ResponseDecline,
				TagNumber:  nil,
				JoinedLate: nil,
			}

			return h.handleParticipantUpdate(ctx, msg, updatePayload, roundtypes.ResponseDecline)
		},
	)

	return wrappedHandler(msg)
}

// HandleTagNumberLookupFailed
func (h *RoundHandlers) HandleTagNumberLookupFailed(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberLookupFailed",
		&sharedevents.RoundTagLookupFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failurePayload := payload.(*sharedevents.RoundTagLookupFailedPayload)

			if (failurePayload.RoundID == sharedtypes.RoundID{}) || failurePayload.UserID == "" {
				h.logger.WarnContext(ctx, "Tag lookup failed payload missing round_id or user_id; skipping fallback participant update",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", failurePayload.RoundID),
					attr.String("user_id", string(failurePayload.UserID)),
					attr.String("reason", failurePayload.Reason),
				)
				return nil, nil
			}

			h.logger.InfoContext(ctx, "Handling tag lookup failure as join success without tag",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", failurePayload.RoundID),
				attr.String("user_id", string(failurePayload.UserID)),
				attr.String("reason", failurePayload.Reason),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				GuildID:   failurePayload.GuildID,
				RoundID:   failurePayload.RoundID,
				UserID:    failurePayload.UserID,
				Response:  roundtypes.ResponseAccept,
				TagNumber: nil,
			}

			return h.handleParticipantUpdate(ctx, msg, updatePayload, roundtypes.ResponseAccept)
		},
	)
	return wrappedHandler(msg)
}

// handleParticipantUpdate is a helper to process participant status updates triggered by various events.
func (h *RoundHandlers) handleParticipantUpdate(
	ctx context.Context,
	msg *message.Message,
	updatePayload *roundevents.ParticipantJoinRequestPayload,
	originalResponse roundtypes.Response,
) ([]*message.Message, error) {
	// Log missing GuildID; do not mutate or enforce it here (handler callers already patch where needed).
	if updatePayload.GuildID == "" {
		h.logger.WarnContext(ctx,
			"Missing guild_id in ParticipantJoinRequestPayload",
			attr.CorrelationIDFromMsg(msg),
			attr.String("guild_id", ""),
		)
	}

	if originalResponse != "" && originalResponse != updatePayload.Response {
		h.logger.InfoContext(ctx, "Participant changed response",
			attr.CorrelationIDFromMsg(msg),
			attr.String("old_status", string(originalResponse)),
			attr.String("new_status", string(updatePayload.Response)),
		)
	}

	updateResult, err := h.roundService.UpdateParticipantStatus(ctx, *updatePayload)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdateParticipantStatus failed",
			attr.CorrelationIDFromMsg(msg),
			attr.Error(err),
		)
		return nil, fmt.Errorf("update participant status: %w", err)
	}

	// Failure branch
	if updateResult.Failure != nil {
		h.logger.InfoContext(ctx, "UpdateParticipantStatus returned failure",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("failure_payload", updateResult.Failure),
		)

		payload, err := h.toJoinError(updateResult.Failure)
		if err != nil {
			h.logger.ErrorContext(ctx, "Unexpected failure payload type from UpdateParticipantStatus",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("payload_type", fmt.Sprintf("%T", updateResult.Failure)),
			)
			return nil, err
		}

		failureMsg, err := h.createResultMessage(ctx, msg, payload, roundevents.RoundParticipantJoinError, "Failed to create failure message after update")
		if err != nil {
			return nil, err
		}
		return []*message.Message{failureMsg}, nil
	}

	// Success branch
	if updateResult.Success != nil {
		h.logger.InfoContext(ctx, "UpdateParticipantStatus returned success",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("success_payload_type", fmt.Sprintf("%T", updateResult.Success)),
		)

		payload, err := h.toJoined(updateResult.Success)
		if err != nil {
			h.logger.ErrorContext(ctx, "Unexpected success payload type from UpdateParticipantStatus",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("payload_type", fmt.Sprintf("%T", updateResult.Success)),
			)
			return nil, err
		}

		discordUpdateMsg, err := h.createResultMessage(ctx, msg, payload, roundevents.RoundParticipantJoined, "Failed to create RoundParticipantJoined message")
		if err != nil {
			return nil, err
		}

		// Patch: copy EventMessageID into metadata for discord consumer fallback
		if payload.EventMessageID != "" && discordUpdateMsg.Metadata.Get("discord_message_id") == "" {
			discordUpdateMsg.Metadata.Set("discord_message_id", payload.EventMessageID)
		}

		h.logger.InfoContext(ctx, "Publishing RoundParticipantJoined message for Discord",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", discordUpdateMsg.UUID),
			attr.String("topic", roundevents.RoundParticipantJoined),
		)

		return []*message.Message{discordUpdateMsg}, nil
	}

	// Should not happen
	h.logger.ErrorContext(ctx, "UpdateParticipantStatus returned nil success and nil failure",
		attr.CorrelationIDFromMsg(msg),
		attr.Any("service_result", updateResult),
	)
	return nil, fmt.Errorf("invalid service result: both success and failure are nil")
}
