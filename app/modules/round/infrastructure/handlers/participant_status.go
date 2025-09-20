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

// HandleParticipantJoinRequest handles the event when a participant initiates a join request (likely from Discord).
func (h *RoundHandlers) HandleParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper( // Use the handlerWrapper
		"HandleParticipantJoinRequest",
		&roundevents.ParticipantJoinRequestPayload{}, // Target for unmarshalling
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// The payload variable is the unmarshalled ParticipantJoinRequestPayload
			participantJoinRequestPayload := payload.(*roundevents.ParticipantJoinRequestPayload)

			// Log the incoming payload to this handler, including guild_id
			h.logger.InfoContext(ctx, "Received ParticipantJoinRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantJoinRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantJoinRequestPayload.UserID)),
				attr.String("response", string(participantJoinRequestPayload.Response)),           // Log the response received here
				attr.String("guild_id", fmt.Sprintf("%v", participantJoinRequestPayload.GuildID)), // Explicitly log guild_id
			)

			// Call the service function to check participant status
			result, err := h.roundService.CheckParticipantStatus(ctx, *participantJoinRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during CheckParticipantStatus service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("CheckParticipantStatus service failed: %w", err) // Return the specific error
			}

			// Patch: Ensure guild_id is propagated in all outgoing payloads
			if result.Success != nil {
				switch successPayload := result.Success.(type) {
				case *roundevents.ParticipantJoinValidationRequestPayload:
					// If the payload is missing guild_id, set it from the incoming request
					if successPayload.GuildID == "" {
						successPayload.GuildID = participantJoinRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in ParticipantJoinValidationRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(successPayload.GuildID)),
						)
					}
				case *roundevents.ParticipantRemovalRequestPayload:
					if successPayload.GuildID == "" {
						successPayload.GuildID = participantJoinRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in ParticipantRemovalRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(successPayload.GuildID)),
						)
					}
				}
			}

			// --- Handle Service Result ---
			if result.Failure != nil {
				// The service returned a failure payload
				h.logger.InfoContext(ctx, "CheckParticipantStatus returned failure",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message - should be ParticipantStatusCheckErrorPayload
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantStatusCheckError, // Publish status check error event
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after status check",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message after check: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil // Return failure message to be published

			} else if result.Success != nil {
				// The service returned a success payload
				h.logger.InfoContext(ctx, "CheckParticipantStatus returned success",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload_type", fmt.Sprintf("%T", result.Success)), // Log the TYPE of the success payload
					attr.Any("success_payload_content", result.Success),                 // Log the CONTENT of the success payload
				) // --- Check the Type of the Success Payload to Determine Next Topic ---
				switch successPayload := result.Success.(type) {
				case *roundevents.ParticipantRemovalRequestPayload: // Changed to pointer type
					// If CheckParticipantStatus returned a Removal Payload, publish a Removal Request message
					removalRequestMsg, err := h.helpers.CreateResultMessage(
						msg,
						successPayload, // Use the removal payload as the message payload
						roundevents.RoundParticipantRemovalRequest, // Publish to the removal request topic
					)
					if err != nil {
						h.logger.ErrorContext(ctx, "Failed to create removal request message",
							attr.CorrelationIDFromMsg(msg),
							attr.Error(err),
						)
						return nil, fmt.Errorf("failed to create removal request message: %w", err)
					}
					h.logger.InfoContext(ctx, "Publishing RoundParticipantRemovalRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", removalRequestMsg.UUID),
						attr.String("topic", roundevents.RoundParticipantRemovalRequest),
					)
					return []*message.Message{removalRequestMsg}, nil // Return the removal message

				case *roundevents.ParticipantJoinValidationRequestPayload: // Changed to pointer type
					// If CheckParticipantStatus returned a Validation Payload, publish a Validation Request message
					validationRequestMsg, err := h.helpers.CreateResultMessage(
						msg,
						successPayload, // Use the validation payload as the message payload
						roundevents.RoundParticipantJoinValidationRequest, // Publish to the validation request topic
					)
					if err != nil {
						h.logger.ErrorContext(ctx, "Failed to create validation request message",
							attr.CorrelationIDFromMsg(msg),
							attr.Error(err),
						)
						return nil, fmt.Errorf("failed to create validation request message: %w", err)
					}
					h.logger.InfoContext(ctx, "Publishing RoundParticipantJoinValidationRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", validationRequestMsg.UUID),
						attr.String("topic", roundevents.RoundParticipantJoinValidationRequest),
					)
					return []*message.Message{validationRequestMsg}, nil // Return the validation message

				default:
					// Handle unexpected success payload type
					err := fmt.Errorf("unexpected success payload type from CheckParticipantStatus: %T", result.Success)
					h.logger.ErrorContext(ctx, "Unexpected success payload type from service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, err // Return the error
				}

			} else {
				// If neither Failure nor Success is set in the result, return an error
				h.logger.ErrorContext(ctx, "CheckParticipantStatus service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("service_result", result),
				)
				return nil, fmt.Errorf("CheckParticipantStatus service returned unexpected nil result")
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleParticipantJoinValidationRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantJoinValidationRequest",
		&roundevents.ParticipantJoinValidationRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantJoinValidationRequestPayload := payload.(*roundevents.ParticipantJoinValidationRequestPayload)

			h.logger.InfoContext(ctx, "Received ParticipantJoinValidationRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantJoinValidationRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantJoinValidationRequestPayload.UserID)),
				attr.String("response", string(participantJoinValidationRequestPayload.Response)),
			)

			// Validate input before calling service
			if participantJoinValidationRequestPayload.UserID == "" {
				h.logger.ErrorContext(ctx, "ParticipantJoinValidationRequest has empty UserID",
					attr.CorrelationIDFromMsg(msg),
				)

				// Create error payload for empty user ID
				errorPayload := &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  participantJoinValidationRequestPayload.RoundID,
						UserID:   participantJoinValidationRequestPayload.UserID,
						Response: participantJoinValidationRequestPayload.Response,
					},
					Error: "User ID cannot be empty",
				}

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					errorPayload,
					roundevents.RoundParticipantJoinError,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message for empty user ID",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message for empty user ID: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			// Call the service function to handle the event
			// Pass all relevant fields, including GuildID, to the service.
			result, err := h.roundService.ValidateParticipantJoinRequest(ctx, roundevents.ParticipantJoinRequestPayload{
				RoundID:  participantJoinValidationRequestPayload.RoundID,
				UserID:   participantJoinValidationRequestPayload.UserID,
				Response: participantJoinValidationRequestPayload.Response,
				GuildID:  participantJoinValidationRequestPayload.GuildID, // Ensure guild_id is propagated
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

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantJoinError,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after validation",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message after validation: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil

			} else if result.Success != nil {
				// Log the type of the success payload before attempting assertion
				h.logger.InfoContext(ctx, "Participant join validation successful - inspecting success payload type BEFORE assertion",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload_type_before_assertion", fmt.Sprintf("%T", result.Success)),
					attr.Any("success_payload_content_before_assertion", result.Success),
				)

				if participantJoinValidationRequestPayload.Response == roundtypes.ResponseDecline {
					updateRequest, ok := result.Success.(*roundevents.ParticipantJoinRequestPayload)
					if !ok {
						err := fmt.Errorf("unexpected success payload type for DECLINE validation: expected *ParticipantJoinRequestPayload, got %T", result.Success)
						h.logger.ErrorContext(ctx, "Type assertion failed for DECLINE validation success payload",
							attr.CorrelationIDFromMsg(msg),
							attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
						)
						return nil, err
					}
					// Patch: Ensure guild_id is present in outgoing DECLINE payload
					if updateRequest.GuildID == "" {
						updateRequest.GuildID = participantJoinValidationRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in DECLINE ParticipantJoinRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(updateRequest.GuildID)),
						)
					}

					h.logger.InfoContext(ctx, "Validation successful for DECLINE - Preparing StatusUpdateRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("update_request_payload", updateRequest),
					)

					successMsg, err := h.helpers.CreateResultMessage(
						msg,
						updateRequest,
						roundevents.RoundParticipantStatusUpdateRequest,
					)
					if err != nil {
						h.logger.ErrorContext(ctx, "Failed to create StatusUpdateRequest message",
							attr.CorrelationIDFromMsg(msg),
							attr.Error(err),
						)
						return nil, fmt.Errorf("failed to create StatusUpdateRequest message: %w", err)
					}
					h.logger.InfoContext(ctx, "Publishing RoundParticipantStatusUpdateRequest message",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", successMsg.UUID),
						attr.String("topic", roundevents.RoundParticipantStatusUpdateRequest),
					)
					return []*message.Message{successMsg}, nil

				} else { // Response is Accept or Tentative
					tagLookupRequest, ok := result.Success.(*roundevents.TagLookupRequestPayload)
					if !ok {
						err := fmt.Errorf("unexpected success payload type for non-DECLINE validation: expected *TagLookupRequestPayload, got %T", result.Success)
						h.logger.ErrorContext(ctx, "Type assertion failed for non-DECLINE validation success payload",
							attr.CorrelationIDFromMsg(msg),
							attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
						)
						return nil, err
					}
					// Patch: Ensure guild_id is present in outgoing ACCEPT/TENTATIVE payload
					if tagLookupRequest.GuildID == "" {
						tagLookupRequest.GuildID = participantJoinValidationRequestPayload.GuildID
						h.logger.WarnContext(ctx, "Patched missing guild_id in TagLookupRequestPayload",
							attr.CorrelationIDFromMsg(msg),
							attr.String("guild_id", string(tagLookupRequest.GuildID)),
						)
					}

					h.logger.InfoContext(ctx, "Validation successful for ACCEPT/TENTATIVE - Preparing TagLookupRequest",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("tag_lookup_request_payload", tagLookupRequest), // Log the asserted payload (includes JoinedLate)
					)

					successMsg, err := h.helpers.CreateResultMessage(
						msg,
						tagLookupRequest,
						roundevents.LeaderboardGetTagNumberRequest,
					)
					if err != nil {
						h.logger.ErrorContext(ctx, "Failed to create TagLookupRequest message",
							attr.CorrelationIDFromMsg(msg),
							attr.Error(err),
						)
						return nil, fmt.Errorf("failed to create TagLookupRequest message: %w", err)
					}
					h.logger.InfoContext(ctx, "Publishing LeaderboardGetTagNumberRequest message",
						attr.CorrelationIDFromMsg(msg),
						attr.String("message_id", successMsg.UUID),
						attr.String("topic", roundevents.LeaderboardGetTagNumberRequest),
					)
					return []*message.Message{successMsg}, nil
				}
			} else {
				h.logger.ErrorContext(ctx, "ValidateParticipantJoinRequest service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("service_result", result),
				)
				return nil, fmt.Errorf("ValidateParticipantJoinRequest service returned unexpected nil result")
			}
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantStatusUpdateRequest handles the event when a participant's status needs updating (e.g., after validation or tag lookup).
func (h *RoundHandlers) HandleParticipantStatusUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	// The outer handlerWrapper handles the high-level span, metrics, and start/end logs
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantStatusUpdateRequest",
		&roundevents.ParticipantJoinRequestPayload{}, // Target for unmarshalling (This handler receives a JoinRequestPayload)
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// The payload variable is the unmarshalled ParticipantJoinRequestPayload
			updateRequestPayload := payload.(*roundevents.ParticipantJoinRequestPayload)

			// Log the incoming payload to this handler
			h.logger.InfoContext(ctx, "Received ParticipantStatusUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", updateRequestPayload.RoundID.String()),
				attr.String("user_id", string(updateRequestPayload.UserID)),
				attr.String("response", string(updateRequestPayload.Response)),
				attr.Any("tag_number", updateRequestPayload.TagNumber), // Log tag number if present in the request
			)

			// Call the service function to update the participant status
			// UpdateParticipantStatus expects a ParticipantJoinRequestPayload, which is what this handler receives.
			result, err := h.roundService.UpdateParticipantStatus(ctx, *updateRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during UpdateParticipantStatus service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("UpdateParticipantStatus service failed: %w", err) // Return the specific error
			}

			// --- Handle Service Result ---
			if result.Failure != nil {
				// The service returned a failure payload
				h.logger.InfoContext(ctx, "UpdateParticipantStatus returned failure",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure), // Log the content of the failure payload
				)

				// Create failure message (assuming RoundParticipantJoinError exists)
				// The failure payload type should match what RoundParticipantJoinError expects
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,                        // Use the failure payload from the service
					roundevents.RoundParticipantJoinError, // Publish a join error event
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after status update",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message after update: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil // Return failure message to be published

			} else if result.Success != nil {
				// The service returned a success payload
				h.logger.InfoContext(ctx, "UpdateParticipantStatus returned success",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload_type", fmt.Sprintf("%T", result.Success)), // Log the TYPE of the success payload
				)

				// --- Add Logging Here ---
				// Log the payload content BEFORE passing it to CreateResultMessage for publishing to Discord.
				// This 'result.Success' should be the ParticipantJoinedPayload struct from your service layer.
				h.logger.InfoContext(ctx, "ParticipantJoinedPayload content BEFORE publishing to Discord",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_to_publish", result.Success), // Log the actual struct content
				)
				// --- End Added Logging ---

				// Create success message to publish for Discord
				// The payload source is result.Success from UpdateParticipantStatus
				// This payload type should match what RoundParticipantJoined event expects (ParticipantJoinedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,                     // <-- THIS struct is being passed here to be marshalled into the message body
					roundevents.RoundParticipantJoined, // <-- Publishing to the Discord joined topic
				)
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to create RoundParticipantJoined message",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
					)
					return nil, fmt.Errorf("failed to create RoundParticipantJoined message: %w", err)
				}

				// Patch: copy EventMessageID into metadata for discord consumer fallback
				if pj, ok := result.Success.(*roundevents.ParticipantJoinedPayload); ok {
					if pj.EventMessageID != "" && successMsg.Metadata.Get("discord_message_id") == "" {
						successMsg.Metadata.Set("discord_message_id", pj.EventMessageID)
					}
				}

				// Log the message being published
				h.logger.InfoContext(ctx, "Publishing RoundParticipantJoined message for Discord",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id", successMsg.UUID),
					attr.String("topic", roundevents.RoundParticipantJoined),
				)

				return []*message.Message{successMsg}, nil // Return the message to be published

			} else {
				// If neither Failure nor Success is set in the result, return an error
				h.logger.ErrorContext(ctx, "UpdateParticipantStatus service returned unexpected nil result",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("service_result", result), // Log the full result struct
				)
				return nil, fmt.Errorf("UpdateParticipantStatus service returned unexpected nil result")
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

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

			// Call the service function to handle the event
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

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantRemovalError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Participant removal request successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundParticipantRemoved,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ParticipantRemoval service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleTagNumberFound handles the event when a tag number lookup result is received.
func (h *RoundHandlers) HandleTagNumberFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberFound",
		&sharedevents.RoundTagLookupResultPayload{}, // Target payload type (shared result payload)
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Correct type assertion for the incoming payload
			tagLookupResultPayload, ok := payload.(*sharedevents.RoundTagLookupResultPayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Invalid payload type for HandleTagNumberFound",
					attr.Any("payload", payload),
				)
				return nil, fmt.Errorf("invalid payload type for HandleTagNumberFound")
			}

			h.logger.InfoContext(ctx, "Received RoundTagLookupFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagLookupResultPayload.RoundID.String()),
				attr.String("user_id", string(tagLookupResultPayload.UserID)),
				attr.Int("tag_number", int(*tagLookupResultPayload.TagNumber)), // Log the found tag
				attr.String("original_response", string(tagLookupResultPayload.OriginalResponse)),
				attr.Any("original_joined_late", tagLookupResultPayload.OriginalJoinedLate),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    tagLookupResultPayload.RoundID,
				UserID:     tagLookupResultPayload.UserID,
				TagNumber:  tagLookupResultPayload.TagNumber,
				JoinedLate: tagLookupResultPayload.OriginalJoinedLate,
				Response:   tagLookupResultPayload.OriginalResponse,
				GuildID:    tagLookupResultPayload.GuildID, // Patch: propagate guild_id from tag lookup result
			}
			// DEBUG: Log the full updatePayload before calling handleParticipantUpdate
			h.logger.InfoContext(ctx, "DEBUG: updatePayload before handleParticipantUpdate",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", fmt.Sprintf("%v", updatePayload.GuildID)),
				attr.String("round_id", updatePayload.RoundID.String()),
				attr.String("user_id", string(updatePayload.UserID)),
				attr.Any("tag_number", updatePayload.TagNumber),
				attr.Any("joined_late", updatePayload.JoinedLate),
				attr.String("response", string(updatePayload.Response)),
			)

			// Call the helper method to update participant status.
			// Pass the constructed payload and the original response from the result payload.
			return h.handleParticipantUpdate(ctx, msg, updatePayload, tagLookupResultPayload.OriginalResponse)
		},
	)

	return wrappedHandler(msg)
}

// HandleTagNumberNotFound processes events where a tag number lookup did not find a tag.
func (h *RoundHandlers) HandleTagNumberNotFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberNotFound",
		&sharedevents.RoundTagLookupResultPayload{}, // Target payload type (shared result payload)
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagLookupResultPayload, ok := payload.(*sharedevents.RoundTagLookupResultPayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Invalid payload type for HandleTagNumberNotFound",
					attr.Any("payload", payload),
				)
				return nil, fmt.Errorf("invalid payload type for HandleTagNumberNotFound")
			}

			h.logger.InfoContext(ctx, "Received RoundTagLookupNotFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagLookupResultPayload.RoundID.String()),
				attr.String("user_id", string(tagLookupResultPayload.UserID)),
				attr.Bool("found_in_payload", tagLookupResultPayload.Found),   // Log Found field
				attr.String("error_in_payload", tagLookupResultPayload.Error), // Log any error
				attr.String("original_response", string(tagLookupResultPayload.OriginalResponse)),
				attr.Any("original_joined_late", tagLookupResultPayload.OriginalJoinedLate),
			)

			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    tagLookupResultPayload.RoundID,
				UserID:     tagLookupResultPayload.UserID,
				TagNumber:  nil,
				JoinedLate: tagLookupResultPayload.OriginalJoinedLate,
				Response:   tagLookupResultPayload.OriginalResponse,
				GuildID:    tagLookupResultPayload.GuildID, // Patch: propagate guild_id from tag lookup result
			}

			return h.handleParticipantUpdate(ctx, msg, updatePayload, tagLookupResultPayload.OriginalResponse)
		},
	)

	return wrappedHandler(msg)
}

// HandleParticipantDeclined processes events where a participant declines a round invitation.
func (h *RoundHandlers) HandleParticipantDeclined(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantDeclined",
		&roundevents.ParticipantDeclinedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantDeclinedPayload, ok := payload.(*roundevents.ParticipantDeclinedPayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Invalid payload type for HandleParticipantDeclined",
					attr.Any("payload", payload),
				)
				return nil, fmt.Errorf("invalid payload type for HandleParticipantDeclined")
			}

			// Log the received event
			h.logger.InfoContext(ctx, "Received ParticipantDeclined event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantDeclinedPayload.RoundID.String()),
				attr.String("user_id", string(participantDeclinedPayload.UserID)),
			)

			// Construct payload for handleParticipantUpdate, as ParticipantDeclinedPayload is not directly compatible
			updatePayload := &roundevents.ParticipantJoinRequestPayload{
				RoundID:    participantDeclinedPayload.RoundID,
				UserID:     participantDeclinedPayload.UserID,
				Response:   roundtypes.ResponseDecline, // Explicitly set response for this handler
				TagNumber:  nil,                        // Tag number is not relevant for a decline
				JoinedLate: nil,                        // JoinedLate is not relevant for a decline
			}

			// Call the helper method to update participant status for decline
			return h.handleParticipantUpdate(ctx, msg, updatePayload, roundtypes.ResponseDecline)
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleTagNumberLookupFailed handles a failure from the leaderboard tag lookup (e.g., no active leaderboard).
// We treat this as a non-fatal condition for joining a round: proceed without a tag number.
func (h *RoundHandlers) HandleTagNumberLookupFailed(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberLookupFailed",
		&sharedevents.RoundTagLookupFailedPayload{}, // Correct payload includes round_id & user_id
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failurePayload := payload.(*sharedevents.RoundTagLookupFailedPayload)

			// Use the payload fields directly (previous version relied on metadata + wrong struct, losing IDs)
			if (failurePayload.RoundID == sharedtypes.RoundID{}) || failurePayload.UserID == "" { // Defensive: should not happen
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
				Response:  roundtypes.ResponseAccept, // TODO: preserve original response by extending failure payload
				TagNumber: nil,
			}

			return h.handleParticipantUpdate(ctx, msg, updatePayload, roundtypes.ResponseAccept)
		},
	)
	return wrappedHandler(msg)
}

// handleParticipantUpdate is a helper function to process participant status updates triggered by various events.
func (h *RoundHandlers) handleParticipantUpdate(
	ctx context.Context,
	msg *message.Message,
	updatePayload *roundevents.ParticipantJoinRequestPayload,
	originalResponse roundtypes.Response,
) ([]*message.Message, error) {
	// Patch: Ensure guild_id is present in outgoing updatePayload
	if updatePayload.GuildID == "" {
		// Try to extract guild_id from context or log a warning
		h.logger.WarnContext(ctx, "Patched missing guild_id in updatePayload for handleParticipantUpdate",
			attr.CorrelationIDFromMsg(msg),
			attr.String("guild_id", ""),
		)
	}
	// Call the service function to update participant status
	updateResult, updateErr := h.roundService.UpdateParticipantStatus(ctx, *updatePayload)
	if updateErr != nil {
		h.logger.ErrorContext(ctx, "UpdateParticipantStatus service failed in helper",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("error", updateErr),
		)
		return nil, fmt.Errorf("UpdateParticipantStatus service failed in helper: %w", updateErr)
	}

	var messagesToReturn []*message.Message

	if updateResult.Success != nil {
		// ALWAYS use RoundParticipantJoined for success, regardless of response type
		// This notifies Discord to update the participant list/status
		successTopic := roundevents.RoundParticipantJoined

		// Perform type assertion to get the concrete *roundevents.ParticipantJoinedPayload
		participantJoinedPayload, ok := updateResult.Success.(*roundevents.ParticipantJoinedPayload)
		if !ok {
			h.logger.ErrorContext(ctx, "Unexpected success payload type from UpdateParticipantStatus in helper",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("payload_type", fmt.Sprintf("%T", updateResult.Success)),
			)
			return nil, fmt.Errorf("unexpected success payload type from UpdateParticipantStatus in helper: expected *roundevents.ParticipantJoinedPayload, got %T", updateResult.Success)
		}

		// Create the message for Discord update
		discordUpdateMsg, err := h.helpers.CreateResultMessage(
			msg,
			participantJoinedPayload,
			successTopic,
		)
		if err != nil {
			h.logger.ErrorContext(ctx, "Failed to create success message",
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err),
			)
			return nil, fmt.Errorf("failed to create success message: %w", err)
		}
		messagesToReturn = append(messagesToReturn, discordUpdateMsg)

	} else if updateResult.Failure != nil {
		h.logger.InfoContext(ctx, "Participant status update failed with specific error",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("failure_payload", updateResult.Failure),
		)
		// ALWAYS use RoundParticipantJoinError for failures, regardless of response type
		// This notifies Discord that there was an error processing the request
		failurePayload, ok := updateResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
		if !ok {
			h.logger.ErrorContext(ctx, "Unexpected failure payload type from UpdateParticipantStatus in helper",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("payload_type", fmt.Sprintf("%T", updateResult.Failure)),
			)
			return nil, fmt.Errorf("unexpected failure payload type from UpdateParticipantStatus")
		}

		failureMsg, errMsg := h.helpers.CreateResultMessage(
			msg,
			failurePayload,
			roundevents.RoundParticipantJoinError, // ALWAYS use this for errors
		)
		if errMsg != nil {
			h.logger.ErrorContext(ctx, "Failed to create failure message after status update in helper",
				attr.CorrelationIDFromMsg(msg),
				attr.Error(errMsg),
			)
			return nil, fmt.Errorf("failed to create failure message after update in helper: %w", errMsg)
		}

		return []*message.Message{failureMsg}, nil

	} else {
		h.logger.ErrorContext(ctx, "UpdateParticipantStatus service returned unexpected nil result in helper",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("service_result", updateResult),
		)
		return nil, fmt.Errorf("UpdateParticipantStatus service returned unexpected nil result in helper")
	}

	return messagesToReturn, nil
}
