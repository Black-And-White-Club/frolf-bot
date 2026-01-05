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
	GuildID     sharedtypes.GuildID
	ChannelID   string
	Title       string
	Description string
	Location    string
	StartTime   string
	Timezone    string
}

// PublishRoundRequest publishes a round creation request and returns the message
func (h *RoundTestHelper) PublishRoundRequest(t *testing.T, ctx context.Context, req RoundRequest) *message.Message {
	t.Helper()

	payload := roundevents.CreateRoundRequestedPayloadV1{
		GuildID:     req.GuildID,
		Title:       roundtypes.Title(req.Title),
		Description: roundtypes.Description(req.Description),
		Location:    roundtypes.Location(req.Location),
		StartTime:   req.StartTime,
		UserID:      req.UserID,
		ChannelID:   req.ChannelID,
		Timezone:    roundtypes.Timezone(req.Timezone),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
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
	return h.capture.WaitForMessages(roundevents.RoundEntityCreatedV1, expectedCount, timeout)
}

// WaitForRoundValidationFailed waits for round validation failed messages
func (h *RoundTestHelper) WaitForRoundValidationFailed(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundValidationFailedV1, expectedCount, timeout)
}

// GetRoundEntityCreatedMessages returns captured round entity created messages
func (h *RoundTestHelper) GetRoundEntityCreatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundEntityCreatedV1)
}

// GetRoundValidationFailedMessages returns captured round validation failed messages
func (h *RoundTestHelper) GetRoundValidationFailedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundValidationFailedV1)
}

// GetAllCapturedMessages returns all captured messages for round topics
func (h *RoundTestHelper) GetAllCapturedMessages() map[string][]*message.Message {
	topics := []string{
		roundevents.RoundEntityCreatedV1,
		roundevents.RoundValidationFailedV1,
		roundevents.RoundCreatedV1,
		roundevents.RoundCreationFailedV1,
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

// ClearMessagesAndDrain clears captured messages, waits for consumers to drain
// any pending messages from NATS, then clears again. This ensures the capture
// buffer only contains messages from the current subtest.
func (h *RoundTestHelper) ClearMessagesAndDrain() {
	// Clear the buffer
	h.capture.Clear()
	// Wait longer for consumers to drain any pending messages from NATS
	// Consumers poll every 25ms, so we need enough time for multiple polling cycles
	// plus message processing time. 300ms = ~12 polling cycles.
	time.Sleep(300 * time.Millisecond)
	// Clear again to remove any drained messages
	h.capture.Clear()
	// Small delay to ensure clean state before test publishes new messages
	time.Sleep(50 * time.Millisecond)
}

// ValidateRoundEntityCreated parses and validates a round entity created message
func (h *RoundTestHelper) ValidateRoundEntityCreated(t *testing.T, msg *message.Message, expectedUserID sharedtypes.DiscordID) *roundevents.RoundEntityCreatedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundEntityCreatedPayloadV1](msg)
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
func (h *RoundTestHelper) ValidateRoundValidationFailed(t *testing.T, msg *message.Message, expectedUserID sharedtypes.DiscordID) *roundevents.RoundValidationFailedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundValidationFailedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round validation failed message: %v", err)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if len(result.ErrorMessages) == 0 {
		t.Errorf("Expected error messages to be populated")
	}

	return result
}

// PublishRoundEntityCreated publishes a RoundEntityCreated event and returns the message
func (h *RoundTestHelper) PublishRoundEntityCreated(t *testing.T, ctx context.Context, payload roundevents.RoundEntityCreatedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundCreated waits for round created messages
func (h *RoundTestHelper) WaitForRoundCreated(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundCreatedV1, expectedCount, timeout)
}

// WaitForRoundCreationFailed waits for round creation failed messages
func (h *RoundTestHelper) WaitForRoundCreationFailed(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundCreationFailedV1, expectedCount, timeout)
}

// GetRoundCreatedMessages returns captured round created messages
func (h *RoundTestHelper) GetRoundCreatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundCreatedV1)
}

// GetRoundCreationFailedMessages returns captured round creation failed messages
func (h *RoundTestHelper) GetRoundCreationFailedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundCreationFailedV1)
}

// ValidateRoundCreated parses and validates a round created message
func (h *RoundTestHelper) ValidateRoundCreated(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundCreatedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundCreatedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round created message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundCreationFailed parses and validates a round creation failed message
func (h *RoundTestHelper) ValidateRoundCreationFailed(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundCreationFailedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundCreationFailedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round creation failed message: %v", err)
	}

	if len(result.ErrorMessage) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// PublishRoundMessageIDUpdate publishes a RoundMessageIDUpdate event with Discord message ID in metadata
func (h *RoundTestHelper) PublishRoundMessageIDUpdate(t *testing.T, ctx context.Context, payload roundevents.RoundMessageIDUpdatePayloadV1, discordMessageID string) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	msg.Metadata.Set("discord_message_id", discordMessageID)

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEventMessageIDUpdateV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundMessageIDUpdateWithoutDiscordID publishes a RoundMessageIDUpdate event without Discord message ID
func (h *RoundTestHelper) PublishRoundMessageIDUpdateWithoutDiscordID(t *testing.T, ctx context.Context, payload roundevents.RoundMessageIDUpdatePayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	// Intentionally not setting discord_message_id

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundEventMessageIDUpdateV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundEventMessageIDUpdated waits for round event message ID updated messages
func (h *RoundTestHelper) WaitForRoundEventMessageIDUpdated(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundEventMessageIDUpdatedV1, expectedCount, timeout)
}

// GetRoundEventMessageIDUpdatedMessages returns captured round event message ID updated messages
func (h *RoundTestHelper) GetRoundEventMessageIDUpdatedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundEventMessageIDUpdatedV1)
}

// ValidateRoundEventMessageIDUpdated parses and validates a round event message ID updated message
func (h *RoundTestHelper) ValidateRoundEventMessageIDUpdated(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundScheduledPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundScheduledPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round event message ID updated message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.EventMessageID == "" {
		t.Errorf("Expected EventMessageID to be populated")
	}

	return result
}

// PublishRoundDeleteRequest publishes a RoundDeleteRequested event and returns the message.
func (h *RoundTestHelper) PublishRoundDeleteRequest(t *testing.T, ctx context.Context, payload roundevents.RoundDeleteRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundDeleteAuthorized publishes a RoundDeleteAuthorized event and returns the message
func (h *RoundTestHelper) PublishRoundDeleteAuthorized(t *testing.T, ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundDeleteAuthorizedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForRoundDeleteAuthorized waits for round delete authorized messages
func (h *RoundTestHelper) WaitForRoundDeleteAuthorized(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeleteAuthorizedV1, expectedCount, timeout)
}

// WaitForRoundDeleteError waits for round delete error messages
func (h *RoundTestHelper) WaitForRoundDeleteError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeleteErrorV1, expectedCount, timeout)
}

// WaitForRoundDeleted waits for round deleted messages
func (h *RoundTestHelper) WaitForRoundDeleted(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundDeletedV1, expectedCount, timeout)
}

// GetRoundDeleteAuthorizedMessages returns captured round delete authorized messages
func (h *RoundTestHelper) GetRoundDeleteAuthorizedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeleteAuthorizedV1)
}

// GetRoundDeleteErrorMessages returns captured round delete error messages
func (h *RoundTestHelper) GetRoundDeleteErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeleteErrorV1)
}

// GetRoundDeletedMessages returns captured round deleted messages
func (h *RoundTestHelper) GetRoundDeletedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundDeletedV1)
}

// ValidateRoundDeleteAuthorized parses and validates a round delete authorized message
func (h *RoundTestHelper) ValidateRoundDeleteAuthorized(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundDeleteAuthorizedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeleteAuthorizedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round delete authorized message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundDeleteError parses and validates a round delete error message
func (h *RoundTestHelper) ValidateRoundDeleteError(t *testing.T, msg *message.Message) *roundevents.RoundDeleteErrorPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeleteErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round delete error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// ValidateRoundDeleted parses and validates a round deleted message
func (h *RoundTestHelper) ValidateRoundDeleted(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundDeletedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundDeletedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round deleted message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// PublishAllScoresSubmitted publishes an AllScoresSubmitted event and returns the message
func (h *RoundTestHelper) PublishAllScoresSubmitted(t *testing.T, ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundAllScoresSubmittedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishRoundFinalized publishes a RoundFinalized event and returns the message
func (h *RoundTestHelper) PublishRoundFinalized(t *testing.T, ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundFinalizedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// GetRoundFinalizedMessages returns captured discord round finalized messages
func (h *RoundTestHelper) GetRoundFinalizedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundFinalizedV1)
}

// GetProcessRoundScoresRequestMessages returns captured process round scores request messages
func (h *RoundTestHelper) GetProcessRoundScoresRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.ProcessRoundScoresRequestedV1)
}

// GetRoundFinalizationErrorMessages returns captured round finalization error messages
func (h *RoundTestHelper) GetRoundFinalizationErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundFinalizationErrorV1)
}

// WaitForRoundFinalized waits for discord round finalized messages
func (h *RoundTestHelper) WaitForRoundFinalized(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundFinalizedV1, expectedCount, timeout)
}

// WaitForProcessRoundScoresRequest waits for process round scores request messages
func (h *RoundTestHelper) WaitForProcessRoundScoresRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.ProcessRoundScoresRequestedV1, expectedCount, timeout)
}

// WaitForRoundFinalizationError waits for round finalization error messages
func (h *RoundTestHelper) WaitForRoundFinalizationError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundFinalizationErrorV1, expectedCount, timeout)
}

// ValidateRoundFinalized parses and validates a discord round finalized message
func (h *RoundTestHelper) ValidateRoundFinalized(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.RoundFinalizedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundFinalizedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse discord round finalized message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateProcessRoundScoresRequest parses and validates a process round scores request message
func (h *RoundTestHelper) ValidateProcessRoundScoresRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ProcessRoundScoresRequestPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.ProcessRoundScoresRequestPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse process round scores request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

// ValidateRoundFinalizationError parses and validates a round finalization error message
func (h *RoundTestHelper) ValidateRoundFinalizationError(t *testing.T, msg *message.Message) *roundevents.RoundFinalizationErrorPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundFinalizationErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round finalization error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}
