package round

import (
	"encoding/json"
	"fmt"
	"strconv"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	roundutil "github.com/Black-And-White-Club/tcr-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandlers handles incoming messages and publishes corresponding events.
type MessageHandlers struct {
	Publisher message.Publisher
}

// HandleMessage processes incoming messages and publishes corresponding events.
func (h *MessageHandlers) HandleMessage(msg *message.Message) error {
	// 1. Determine message type (e.g., from metadata or payload)
	messageType := msg.Metadata.Get("type")

	switch messageType {
	case "create_round":
		return h.handleCreateRound(msg)
	case "update_round":
		return h.handleUpdateRound(msg)
	case "delete_round":
		return h.handleDeleteRound(msg)
	case "participant_response":
		return h.handleParticipantResponse(msg)
	case "score_updated":
		return h.handleScoreUpdated(msg)
	default:
		return fmt.Errorf("unknown message type: %s", messageType)
	}
}

func (h *MessageHandlers) handleCreateRound(msg *message.Message) error {
	var createRoundDto rounddto.CreateRoundParams
	if err := json.Unmarshal(msg.Payload, &createRoundDto); err != nil {
		return fmt.Errorf("failed to unmarshal CreateRoundParams: %w", err)
	}

	// Publish RoundCreatedEvent (using parsed date from DTO)
	startTime, err := roundutil.ParseDateTime(createRoundDto.DateTime.Date + " " + createRoundDto.DateTime.Time) // Parse the combined date and time
	if err != nil {
		return fmt.Errorf("failed to parse date/time: %w", err)
	}

	// Publish RoundCreatedEvent
	event := roundevents.RoundCreatedEvent{
		Name:      createRoundDto.Title,
		StartTime: startTime, // Use the parsed startTime
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundCreatedEvent: %w", err)
	}
	if err := h.Publisher.Publish(roundevents.RoundCreatedSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		return fmt.Errorf("failed to publish RoundCreatedEvent: %w", err)
	}

	return nil
}

func (h *MessageHandlers) handleUpdateRound(msg *message.Message) error {
	var updateRoundDto rounddto.EditRoundInput
	if err := json.Unmarshal(msg.Payload, &updateRoundDto); err != nil {
		return fmt.Errorf("failed to unmarshal EditRoundInput: %w", err)
	}

	// Publish RoundUpdatedEvent (extract relevant fields from DTO)
	event := roundevents.RoundUpdatedEvent{
		RoundID:   strconv.FormatInt(updateRoundDto.RoundID, 10),
		Title:     &updateRoundDto.Title,
		Location:  &updateRoundDto.Location,
		EventType: updateRoundDto.EventType,
		Date:      &updateRoundDto.Date,
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundUpdatedEvent: %w", err)
	}
	if err := h.Publisher.Publish(roundevents.RoundUpdatedSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		return fmt.Errorf("failed to publish RoundUpdatedEvent: %w", err)
	}

	return nil
}

func (h *MessageHandlers) handleDeleteRound(msg *message.Message) error {
	var deleteRoundDto struct {
		RoundID string `json:"round_id"`
	}
	if err := json.Unmarshal(msg.Payload, &deleteRoundDto); err != nil {
		return fmt.Errorf("failed to unmarshal delete round request: %w", err)
	}

	// Publish RoundDeletedEvent
	event := roundevents.RoundDeletedEvent{
		RoundID: deleteRoundDto.RoundID,
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundDeletedEvent: %w", err)
	}
	if err := h.Publisher.Publish(roundevents.RoundDeletedSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		return fmt.Errorf("failed to publish RoundDeletedEvent: %w", err)
	}

	return nil
}

func (h *MessageHandlers) handleParticipantResponse(msg *message.Message) error {
	var participantResponseDto rounddto.UpdateParticipantResponseInput
	if err := json.Unmarshal(msg.Payload, &participantResponseDto); err != nil {
		return fmt.Errorf("failed to unmarshal UpdateParticipantResponseInput: %w", err)
	}

	event := roundevents.ParticipantResponseEvent{
		RoundID:     strconv.FormatInt(participantResponseDto.RoundID, 10), // Convert int64 to string
		Participant: participantResponseDto.DiscordID,
		Response:    string(participantResponseDto.Response),
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ParticipantResponseEvent: %w", err)
	}
	if err := h.Publisher.Publish(roundevents.ParticipantResponseSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		return fmt.Errorf("failed to publish ParticipantResponseEvent: %w", err)
	}

	return nil
}

func (h *MessageHandlers) handleScoreUpdated(msg *message.Message) error {
	var scoreUpdatedDto rounddto.SubmitScoreInput
	if err := json.Unmarshal(msg.Payload, &scoreUpdatedDto); err != nil {
		return fmt.Errorf("failed to unmarshal SubmitScoreInput: %w", err)
	}

	// Publish ScoreUpdatedEvent
	event := roundevents.ScoreUpdatedEvent{
		RoundID:     strconv.FormatInt(scoreUpdatedDto.RoundID, 10), // Convert int64 to string
		Participant: scoreUpdatedDto.DiscordID,
		Score:       scoreUpdatedDto.Score,
		UpdateType:  rounddb.ScoreUpdateTypeRegular, // Set the UpdateType
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ScoreUpdatedEvent: %w", err)
	}
	if err := h.Publisher.Publish(roundevents.ScoreUpdatedSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		return fmt.Errorf("failed to publish ScoreUpdatedEvent: %w", err)
	}

	return nil
}
