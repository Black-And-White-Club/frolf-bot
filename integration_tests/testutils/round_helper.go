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

// RoundRequest represents a round creation request - generic structure
type RoundRequest struct {
	UserID      sharedtypes.DiscordID
	Title       string
	Description string
	Location    string
	StartTime   string
	Timezone    string
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
		Timezone:    roundtypes.Timezone(req.Timezone),
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
func (h *RoundTestHelper) PublishInvalidJSON(t *testing.T, ctx context.Context, topic string) *message.Message {
	t.Helper()

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, topic, msg); err != nil {
		t.Fatalf("Publish failed for topic %s: %v", topic, err)
	}

	return msg
}

// WaitForRoundEntityCreated waits for round entity created messages
func (h *RoundTestHelper) WaitForRoundEntityCreated(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundEntityCreated, expectedCount, timeout)
}

// WaitForRoundValidationFailed waits for round validation failed messages
func (h *RoundTestHelper) WaitForRoundValidationFailed(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundValidationFailed, expectedCount, timeout)
}

// GetRoundEntityCreatedMessages returns captured round entity created messages
func (h *RoundTestHelper) GetRoundEntityCreatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundEntityCreated)
}

// GetRoundValidationFailedMessages returns captured round validation failed messages
func (h *RoundTestHelper) GetRoundValidationFailedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundValidationFailed)
}

// GetAllCapturedMessages returns all captured messages for round topics
func (h *RoundTestHelper) GetAllCapturedMessages() map[string][]*message.Message {
	topics := []string{
		roundevents.RoundEntityCreated,
		roundevents.RoundValidationFailed,
		roundevents.RoundCreated,
		roundevents.RoundCreationFailed,
	}

	result := make(map[string][]*message.Message)
	for _, topic := range topics {
		result[topic] = h.capture.GetMessages(topic)
	}
	return result
}

// ClearMessages clears all captured messages
func (h *RoundTestHelper) ClearMessages() {
	h.capture.Clear()
}

// ValidateRoundEntityCreated parses and validates a round entity created message
func (h *RoundTestHelper) ValidateRoundEntityCreated(t *testing.T, msg *message.Message, expectedUserID sharedtypes.DiscordID) *roundevents.RoundEntityCreatedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundEntityCreatedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round entity created message: %v", err)
	}

	if result.Round.CreatedBy != expectedUserID {
		t.Errorf("CreatedBy mismatch: expected %s, got %s", expectedUserID, result.Round.CreatedBy)
	}

	if result.Round.State != roundtypes.RoundStateUpcoming {
		t.Errorf("Expected state %s, got %s", roundtypes.RoundStateUpcoming, result.Round.State)
	}

	return result
}

// ValidateRoundValidationFailed parses and validates a round validation failed message
func (h *RoundTestHelper) ValidateRoundValidationFailed(t *testing.T, msg *message.Message, expectedUserID sharedtypes.DiscordID) *roundevents.RoundValidationFailedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundValidationFailedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round validation failed message: %v", err)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if len(result.ErrorMessage) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// PublishRoundEntityCreated publishes a RoundEntityCreated event and returns the message
func (h *RoundTestHelper) PublishRoundEntityCreated(t *testing.T, ctx context.Context, payload roundevents.RoundEntityCreatedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEntityCreated, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundCreated waits for round created messages
func (h *RoundTestHelper) WaitForRoundCreated(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundCreated, expectedCount, timeout)
}

// WaitForRoundCreationFailed waits for round creation failed messages
func (h *RoundTestHelper) WaitForRoundCreationFailed(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundCreationFailed, expectedCount, timeout)
}

// GetRoundCreatedMessages returns captured round created messages
func (h *RoundTestHelper) GetRoundCreatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundCreated)
}

// GetRoundCreationFailedMessages returns captured round creation failed messages
func (h *RoundTestHelper) GetRoundCreationFailedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundCreationFailed)
}

// ValidateRoundCreated parses and validates a round created message
func (h *RoundTestHelper) ValidateRoundCreated(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundCreatedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundCreatedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round created message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundCreationFailed parses and validates a round creation failed message
func (h *RoundTestHelper) ValidateRoundCreationFailed(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundCreationFailedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundCreationFailedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round creation failed message: %v", err)
	}

	if len(result.ErrorMessage) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// PublishRoundMessageIDUpdate publishes a RoundMessageIDUpdate event with Discord message ID in metadata
func (h *RoundTestHelper) PublishRoundMessageIDUpdate(t *testing.T, ctx context.Context, payload roundevents.RoundMessageIDUpdatePayload, discordMessageID string) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	msg.Metadata.Set("discord_message_id", discordMessageID)

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEventMessageIDUpdate, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundMessageIDUpdateWithoutDiscordID publishes a RoundMessageIDUpdate event without Discord message ID
func (h *RoundTestHelper) PublishRoundMessageIDUpdateWithoutDiscordID(t *testing.T, ctx context.Context, payload roundevents.RoundMessageIDUpdatePayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	// Intentionally not setting discord_message_id

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEventMessageIDUpdate, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundEventMessageIDUpdated waits for round event message ID updated messages
func (h *RoundTestHelper) WaitForRoundEventMessageIDUpdated(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundEventMessageIDUpdated, expectedCount, timeout)
}

// GetRoundEventMessageIDUpdatedMessages returns captured round event message ID updated messages
func (h *RoundTestHelper) GetRoundEventMessageIDUpdatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundEventMessageIDUpdated)
}

// ValidateRoundEventMessageIDUpdated parses and validates a round event message ID updated message
func (h *RoundTestHelper) ValidateRoundEventMessageIDUpdated(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundScheduledPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundScheduledPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round event message ID updated message: %v", err)
	}

	if result.BaseRoundPayload.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.BaseRoundPayload.RoundID)
	}

	if result.EventMessageID == "" {
		t.Errorf("Expected EventMessageID to be populated")
	}

	return result
}

// PublishRoundDeleteRequest publishes a RoundDeleteRequest event and returns the message
func (h *RoundTestHelper) PublishRoundDeleteRequest(t *testing.T, ctx context.Context, payload roundevents.RoundDeleteRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundDeleteRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundDeleteAuthorized publishes a RoundDeleteAuthorized event and returns the message
func (h *RoundTestHelper) PublishRoundDeleteAuthorized(t *testing.T, ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundDeleteAuthorized, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundDeleteAuthorized waits for round delete authorized messages
func (h *RoundTestHelper) WaitForRoundDeleteAuthorized(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeleteAuthorized, expectedCount, timeout)
}

// WaitForRoundDeleteError waits for round delete error messages
func (h *RoundTestHelper) WaitForRoundDeleteError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeleteError, expectedCount, timeout)
}

// WaitForRoundDeleted waits for round deleted messages
func (h *RoundTestHelper) WaitForRoundDeleted(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeleted, expectedCount, timeout)
}

// GetRoundDeleteAuthorizedMessages returns captured round delete authorized messages
func (h *RoundTestHelper) GetRoundDeleteAuthorizedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeleteAuthorized)
}

// GetRoundDeleteErrorMessages returns captured round delete error messages
func (h *RoundTestHelper) GetRoundDeleteErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeleteError)
}

// GetRoundDeletedMessages returns captured round deleted messages
func (h *RoundTestHelper) GetRoundDeletedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeleted)
}

// ValidateRoundDeleteAuthorized parses and validates a round delete authorized message
func (h *RoundTestHelper) ValidateRoundDeleteAuthorized(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundDeleteAuthorizedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeleteAuthorizedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round delete authorized message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundDeleteError parses and validates a round delete error message
func (h *RoundTestHelper) ValidateRoundDeleteError(t *testing.T, msg *message.Message) *roundevents.RoundDeleteErrorPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeleteErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round delete error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// ValidateRoundDeleted parses and validates a round deleted message
func (h *RoundTestHelper) ValidateRoundDeleted(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundDeletedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeletedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round deleted message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// PublishAllScoresSubmitted publishes an AllScoresSubmitted event and returns the message
func (h *RoundTestHelper) PublishAllScoresSubmitted(t *testing.T, ctx context.Context, payload roundevents.AllScoresSubmittedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundAllScoresSubmitted, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundFinalized publishes a RoundFinalized event and returns the message
func (h *RoundTestHelper) PublishRoundFinalized(t *testing.T, ctx context.Context, payload roundevents.RoundFinalizedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundFinalized, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// GetDiscordRoundFinalizedMessages returns captured discord round finalized messages
func (h *RoundTestHelper) GetDiscordRoundFinalizedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.DiscordRoundFinalized)
}

// GetProcessRoundScoresRequestMessages returns captured process round scores request messages
func (h *RoundTestHelper) GetProcessRoundScoresRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.ProcessRoundScoresRequest)
}

// GetRoundFinalizationErrorMessages returns captured round finalization error messages
func (h *RoundTestHelper) GetRoundFinalizationErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundFinalizationError)
}

// WaitForDiscordRoundFinalized waits for discord round finalized messages
func (h *RoundTestHelper) WaitForDiscordRoundFinalized(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.DiscordRoundFinalized, expectedCount, timeout)
}

// WaitForProcessRoundScoresRequest waits for process round scores request messages
func (h *RoundTestHelper) WaitForProcessRoundScoresRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.ProcessRoundScoresRequest, expectedCount, timeout)
}

// WaitForRoundFinalizationError waits for round finalization error messages
func (h *RoundTestHelper) WaitForRoundFinalizationError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundFinalizationError, expectedCount, timeout)
}

// ValidateDiscordRoundFinalized parses and validates a discord round finalized message
func (h *RoundTestHelper) ValidateDiscordRoundFinalized(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundFinalizedPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundFinalizedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse discord round finalized message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateProcessRoundScoresRequest parses and validates a process round scores request message
func (h *RoundTestHelper) ValidateProcessRoundScoresRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ProcessRoundScoresRequestPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.ProcessRoundScoresRequestPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse process round scores request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundFinalizationError parses and validates a round finalization error message
func (h *RoundTestHelper) ValidateRoundFinalizationError(t *testing.T, msg *message.Message) *roundevents.RoundFinalizationErrorPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundFinalizationErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round finalization error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}
