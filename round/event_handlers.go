// round/event_handlers.go

package round

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundEventHandler handles events related to rounds.
type RoundEventHandler struct {
	roundService *RoundService
	publisher    message.Publisher
}

// NewRoundEventHandler creates a new RoundEventHandler.
func NewRoundEventHandler(roundService *RoundService, publisher message.Publisher) *RoundEventHandler { // Add publisher as argument
	return &RoundEventHandler{
		roundService: roundService,
		publisher:    publisher,
	}
}

func (h *RoundEventHandler) HandleRoundCreate(ctx context.Context, event *RoundCreateEvent) error {
	// No need to unmarshal here, event is already provided

	fmt.Printf("Received RoundCreateEvent: %+v\n", event)

	// --- Input validation ---
	if event.Course == "" || event.Date.IsZero() || event.Time == "" || event.UserID == "" {
		log.Printf("Invalid RoundCreateEvent: missing required fields")
		return errors.New("invalid RoundCreateEvent")
	}

	// --- Create a new round using RoundService ---
	input := ScheduleRoundInput{
		Title:     fmt.Sprintf("Round created by %s", event.UserID),
		Location:  event.Course,
		Date:      event.Date,
		Time:      event.Time,
		DiscordID: event.UserID,
	}

	if _, err := h.roundService.ScheduleRound(ctx, input); err != nil {
		log.Printf("Failed to schedule round: %v, input: %+v", err, input)
		return err
	}

	return nil
}

func (h *RoundEventHandler) HandlePlayerAddedToRound(ctx context.Context, msg *message.Message) error {
	var evt PlayerAddedToRoundEvent // No need for round. prefix
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal PlayerAddedToRoundEvent: %w", err)
	}

	// --- Publish TagNumberRequestedEvent ---
	if err := h.publisher.Publish(evt.Topic(), message.NewMessage(watermill.NewUUID(), msg.Payload)); err != nil {
		return fmt.Errorf("failed to publish TagNumberRequestedEvent: %w", err)
	}

	return nil
}

func (h *RoundEventHandler) HandleTagNumberRetrieved(ctx context.Context, msg *message.Message) error {
	var evt TagNumberRetrievedEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal TagNumberRetrievedEvent: %w", err)
	}

	// --- Asynchronously call the service layer to update the participant ---
	go func() {
		// Create UpdateParticipantResponseInput from the event data
		input := UpdateParticipantResponseInput{
			RoundID:   evt.RoundID,
			DiscordID: evt.UserID,
		}

		if _, err := h.roundService.UpdateParticipant(context.Background(), input); err != nil {
			log.Printf("Error updating participant tag number: %v", err)
		}
	}()

	return nil
}

// TODO: Add corresponding event handlers to the leaderboard domain for
//       TagNumberRequestedEvent and TagNumberRetrievedEvent.

// ScoreSubmittedEventHandler handles ScoreSubmittedEvent.
func (h *RoundEventHandler) HandleScoreSubmitted(ctx context.Context, event *ScoreSubmittedEvent) error {

	// Call the service layer to handle the score submission
	if err := h.roundService.ProcessScoreSubmission(ctx, *event); err != nil { // Use *event here
		return fmt.Errorf("failed to process score submission: %w", err)
	}

	return nil
}

func (h *RoundEventHandler) HandleRoundStarted(ctx context.Context, event *RoundStartedEvent) error {
	// Update the round state to "IN_PROGRESS" using your service layer's RoundState
	err := h.roundService.UpdateRoundState(ctx, event.RoundID, RoundStateInProgress)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	// ... (Potentially add other actions, like sending notifications) ...
	return nil
}

func (h *RoundEventHandler) HandleRoundStartingOneHour(ctx context.Context, event *RoundStartingOneHourEvent) error {
	// Implement logic to send a 1-hour notification for the round
	// ... (e.g., send a Discord message) ...

	return nil
}

func (h *RoundEventHandler) HandleRoundStartingThirtyMinutes(ctx context.Context, event *RoundStartingThirtyMinutesEvent) error {
	// Implement logic to send a 30-minute notification for the round
	// ... (e.g., send a Discord message) ...

	return nil
}

func (h *RoundEventHandler) HandleRoundUpdated(ctx context.Context, event *RoundUpdatedEvent) error {
	// Implement logic to handle round updates (if needed)
	// ...

	return nil
}

func (h *RoundEventHandler) HandleRoundDeleted(ctx context.Context, event *RoundDeletedEvent) error {
	// Implement logic to handle round deletions (if needed)
	// ...

	return nil
}

func (h *RoundEventHandler) HandleRoundFinalized(ctx context.Context, event *RoundFinalizedEvent) error {
	// Implement logic to handle round finalization (if needed)
	// ...

	return nil
}
