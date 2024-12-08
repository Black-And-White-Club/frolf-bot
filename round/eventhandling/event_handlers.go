// eventhandling/event_handlers.go

package roundevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	roundapi "github.com/Black-And-White-Club/tcr-bot/round/api"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundEventHandler handles events related to rounds.
type RoundEventHandlerImpl struct {
	roundService roundapi.CommandService // Use roundapi.CommandService
	publisher    message.Publisher
}

// NewRoundEventHandler creates a new RoundEventHandler.
func NewRoundEventHandler(roundService roundapi.CommandService, publisher message.Publisher) *RoundEventHandlerImpl {
	return &RoundEventHandlerImpl{
		roundService: roundService,
		publisher:    publisher,
	}
}

// HandleRoundCreate implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundCreate(ctx context.Context, event *RoundCreateEvent) error {
	// No need to unmarshal here, event is already provided

	fmt.Printf("Received RoundCreateEvent: %+v\n", event)

	// --- Input validation ---
	if event.Course == "" || event.Date.IsZero() || event.Time == "" || event.UserID == "" {
		log.Printf("Invalid RoundCreateEvent: missing required fields")
		return errors.New("invalid RoundCreateEvent")
	}

	// --- Create a new round using RoundService ---
	input := apimodels.ScheduleRoundInput{
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

// HandlePlayerAddedToRound implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandlePlayerAddedToRound(ctx context.Context, msg *message.Message) error {
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

// HandleTagNumberRetrieved implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleTagNumberRetrieved(ctx context.Context, msg *message.Message) error {
	var evt TagNumberRetrievedEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal TagNumberRetrievedEvent: %w", err)
	}

	// --- Asynchronously call the service layer to update the participant ---
	go func() {
		// Create UpdateParticipantResponseInput from the event data
		input := apimodels.UpdateParticipantResponseInput{
			RoundID:   evt.RoundID,
			DiscordID: evt.UserID,
		}

		if _, err := h.roundService.UpdateParticipant(ctx, input); err != nil { // Pass the context here
			log.Printf("Error updating participant tag number: %v", err)
		}
	}()

	return nil
}

// TODO: Add corresponding event handlers to the leaderboard domain for
//       TagNumberRequestedEvent and TagNumberRetrievedEvent.

// HandleScoreSubmitted implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleScoreSubmitted(ctx context.Context, event *ScoreSubmittedEvent) error {

	// Call the service layer to handle the score submission
	if err := h.roundService.ProcessScoreSubmission(ctx, *event); err != nil { // Use *event here
		return fmt.Errorf("failed to process score submission: %w", err)
	}

	return nil
}

// HandleRoundStarted implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundStarted(ctx context.Context, event *RoundStartedEvent) error {
	// Update the round state to "IN_PROGRESS" using your service layer's RoundState
	err := h.roundService.UpdateRoundState(ctx, event.RoundID, apimodels.RoundStateInProgress)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}
	// ... (Potentially add other actions, like sending notifications) ...
	return nil
}

// HandleRoundStartingOneHour implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundStartingOneHour(ctx context.Context, event *RoundStartingOneHourEvent) error {
	// Implement logic to send a 1-hour notification for the round
	// ... (e.g., send a Discord message) ...

	return nil
}

// HandleRoundStartingThirtyMinutes implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundStartingThirtyMinutes(ctx context.Context, event *RoundStartingThirtyMinutesEvent) error {
	// Implement logic to send a 30-minute notification for the round
	// ... (e.g., send a Discord message) ...

	return nil
}

// HandleRoundUpdated implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundUpdated(ctx context.Context, event *RoundUpdatedEvent) error {
	// Implement logic to handle round updates (if needed)
	// ...

	return nil
}

// HandleRoundDeleted implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundDeleted(ctx context.Context, event *RoundDeletedEvent) error {
	// Implement logic to handle round deletions (if needed)
	// ...

	return nil
}

// HandleRoundFinalized implements RoundEventHandler interface.
func (h *RoundEventHandlerImpl) HandleRoundFinalized(ctx context.Context, event *RoundFinalizedEvent) error {
	// Implement logic to handle round finalization (if needed)
	// ...

	return nil
}
