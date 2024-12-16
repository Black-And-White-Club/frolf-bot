package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboardservices "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/services"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

// TagSwapHandler handles TagSwapRequest events.
type TagSwapHandler struct {
	js                 nats.JetStreamContext
	eventBus           watermillutil.PubSuber
	leaderboardService leaderboardservices.LeaderboardService
	swapRequestQueue   chan *leaderboardcommands.TagSwapRequest
}

// NewTagSwapHandler creates a new TagSwapHandler.
func NewTagSwapHandler(js nats.JetStreamContext, eventBus watermillutil.PubSuber, leaderboardService leaderboardservices.LeaderboardService) *TagSwapHandler {
	h := &TagSwapHandler{
		js:                 js,
		eventBus:           eventBus,
		leaderboardService: leaderboardService,
		swapRequestQueue:   make(chan *leaderboardcommands.TagSwapRequest),
	}

	// Start the queue processing goroutine
	go h.processSwapRequests()

	return h
}

// Handle processes TagSwapRequest events.
func (h *TagSwapHandler) Handle(msg *message.Message) error {
	var swapRequest leaderboardcommands.TagSwapRequest
	if err := watermillutil.Marshaler.Unmarshal(msg, &swapRequest); err != nil {
		return errors.Wrap(err, "failed to unmarshal TagSwapRequest")
	}

	// Add the request to the queue
	h.swapRequestQueue <- &swapRequest

	return nil
}

func (h *TagSwapHandler) processSwapRequests() {
	for swapRequest := range h.swapRequestQueue {
		// Initiate the tag swap
		result, err := h.leaderboardService.InitiateTagSwap(context.Background(), swapRequest)
		if err != nil {
			// Handle the error (e.g., log it)
			fmt.Println("Error initiating tag swap:", err)
			continue
		}

		if result.MatchFound {
			// Publish TagSwapCommand to trigger the tag swap in another handler
			tagSwapCommand := &leaderboardcommands.TagSwapRequest{
				RequestorID:  result.RequestorID,
				RequestorTag: result.RequestorTag,
				TargetTag:    result.TargetTag,
			}

			payload, err := json.Marshal(tagSwapCommand)
			if err != nil {
				fmt.Println("Error marshaling TagSwapCommand:", err)
				continue
			}

			if err := h.eventBus.Publish(TopicTagSwapCommand, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
				fmt.Println("Error publishing TagSwapCommand:", err)
				continue
			}

			// Remove the swap group from Jetstream in the handler
			if err := h.leaderboardService.RemoveSwapGroup(context.Background(), h.js, result.SwapGroup); err != nil {
				fmt.Println("Error removing swap group:", err)
				continue
			}
		} else {
			// Store the swap group in Jetstream in the handler
			if err := h.leaderboardService.StoreSwapGroup(context.Background(), h.js, result.SwapGroup); err != nil {
				fmt.Println("Error storing swap group:", err)
				continue
			}
		}
	}
}
