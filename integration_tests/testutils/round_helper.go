package testutils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// RoundTestHelper provides utilities for round testing
type RoundTestHelper struct {
	eventBus eventbus.EventBus
	capture  *MessageCapture
}

// NewRoundTestHelper creates a new round test helper
func NewRoundTestHelper(eventBus eventbus.EventBus, capture *MessageCapture) *RoundTestHelper {
	return &RoundTestHelper{
		eventBus: eventBus,
		capture:  capture,
	}
}

// RoundRequest represents a round creation request
type RoundRequest struct {
	UserID      sharedtypes.DiscordID
	Title       roundtypes.Title
	Description roundtypes.Description
	Location    roundtypes.Location
	StartTime   string
	Timezone    roundtypes.Timezone
}

// CreateValidRequest creates a valid round request with sensible defaults
func (h *RoundTestHelper) CreateValidRequest(userID sharedtypes.DiscordID) RoundRequest {
	return RoundRequest{
		UserID:      userID,
		Title:       "Weekly Frolf Championship",
		Description: "Join us for our weekly championship round!",
		Location:    "Central Park Course",
		StartTime:   "tomorrow at 3:00 PM",
		Timezone:    "UTC",
	}
}

// CreateMinimalRequest creates a minimal but valid round request
func (h *RoundTestHelper) CreateMinimalRequest(userID sharedtypes.DiscordID) RoundRequest {
	return RoundRequest{
		UserID:      userID,
		Title:       "Quick Round",
		Description: "Quick round for today",
		Location:    "Local Course",
		StartTime:   "tomorrow at 3:00 PM",
		Timezone:    "UTC",
	}
}

// CreateInvalidRequest creates various types of invalid requests
func (h *RoundTestHelper) CreateInvalidRequest(userID sharedtypes.DiscordID, invalidType string) RoundRequest {
	base := h.CreateValidRequest(userID)

	switch invalidType {
	case "empty_title":
		base.Title = ""
	case "empty_description":
		base.Description = ""
	case "empty_location":
		base.Location = ""
	case "invalid_time":
		base.StartTime = "not-a-valid-time"
	case "past_time":
		base.StartTime = "yesterday at 3:00 PM"
	case "missing_fields":
		return RoundRequest{
			UserID:      userID,
			Description: "Description only",
		}
	}

	return base
}

// PublishRoundRequest publishes a round creation request and returns the message
func (h *RoundTestHelper) PublishRoundRequest(t *testing.T, ctx context.Context, req RoundRequest) *message.Message {
	t.Helper()

	payload := roundevents.CreateRoundRequestedPayload{
		UserID:      req.UserID,
		Title:       roundtypes.Title(req.Title),
		Description: roundtypes.Description(req.Description),
		Location:    roundtypes.Location(req.Location),
		StartTime:   req.StartTime,
		Timezone:    req.Timezone,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundCreateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishInvalidJSON publishes an invalid JSON message
func (h *RoundTestHelper) PublishInvalidJSON(t *testing.T, ctx context.Context) *message.Message {
	t.Helper()

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundCreateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// ExpectSuccess waits for and validates a successful round creation
func (h *RoundTestHelper) ExpectSuccess(t *testing.T, originalRequest RoundRequest, timeout time.Duration) {
	t.Helper()

	if !h.capture.WaitForMessages(roundevents.RoundEntityCreated, 1, timeout) {
		t.Fatalf("Expected round.entity.created message within %v", timeout)
	}

	msgs := h.capture.GetMessages(roundevents.RoundEntityCreated)
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 success message, got %d", len(msgs))
	}

	result, err := ParsePayload[roundevents.RoundEntityCreatedPayload](msgs[0])
	if err != nil {
		t.Fatalf("Failed to parse success message: %v", err)
	}

	// Validate transformation
	if result.Round.CreatedBy != originalRequest.UserID {
		t.Errorf("CreatedBy mismatch: expected %s, got %s", originalRequest.UserID, result.Round.CreatedBy)
	}

	if result.Round.Title != roundtypes.Title(originalRequest.Title) {
		t.Errorf("Title mismatch: expected %s, got %s", originalRequest.Title, result.Round.Title)
	}

	if result.Round.Location == nil || *result.Round.Location != roundtypes.Location(originalRequest.Location) {
		t.Errorf("Location mismatch: expected %s, got %v", originalRequest.Location, result.Round.Location)
	}

	if result.Round.State != roundtypes.RoundStateUpcoming {
		t.Errorf("Expected state %s, got %s", roundtypes.RoundStateUpcoming, result.Round.State)
	}

	if len(result.Round.Participants) != 0 {
		t.Errorf("Expected empty participants, got %d", len(result.Round.Participants))
	}
}

// ExpectValidationFailure waits for and validates a validation failure
func (h *RoundTestHelper) ExpectValidationFailure(t *testing.T, originalRequest RoundRequest, timeout time.Duration) {
	t.Helper()

	if !h.capture.WaitForMessages(roundevents.RoundValidationFailed, 1, timeout) {
		t.Fatalf("Expected validation failure message within %v", timeout)
	}

	msgs := h.capture.GetMessages(roundevents.RoundValidationFailed)
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 failure message, got %d", len(msgs))
	}

	result, err := ParsePayload[roundevents.RoundValidationFailedPayload](msgs[0])
	if err != nil {
		t.Fatalf("Failed to parse failure message: %v", err)
	}

	if result.UserID != originalRequest.UserID {
		t.Errorf("UserID mismatch: expected %s, got %s", originalRequest.UserID, result.UserID)
	}

	if len(result.ErrorMessage) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	// Ensure no success message was published
	successMsgs := h.capture.GetMessages(roundevents.RoundEntityCreated)
	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages, got %d", len(successMsgs))
	}
}

// ExpectNoMessages validates that no messages were published (for JSON errors)
func (h *RoundTestHelper) ExpectNoMessages(t *testing.T, waitTime time.Duration) {
	t.Helper()

	time.Sleep(waitTime)

	topics := []string{
		roundevents.RoundEntityCreated,
		roundevents.RoundValidationFailed,
	}

	for _, topic := range topics {
		msgs := h.capture.GetMessages(topic)
		if len(msgs) > 0 {
			t.Errorf("Expected no messages on %s, got %d", topic, len(msgs))
		}
	}
}

// ClearMessages clears all captured messages
func (h *RoundTestHelper) ClearMessages() {
	h.capture.Clear()
}
