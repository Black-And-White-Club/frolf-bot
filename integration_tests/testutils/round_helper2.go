package testutils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Add these methods to your existing RoundTestHelper:

// PublishParticipantJoinRequest publishes a ParticipantJoinRequest event and returns the message
func (h *RoundTestHelper) PublishParticipantJoinRequest(t *testing.T, ctx context.Context, payload roundevents.ParticipantJoinRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundParticipantJoinRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// GetParticipantRemovalRequestMessages returns captured participant removal request messages
func (h *RoundTestHelper) GetParticipantRemovalRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantRemovalRequestedV1)
}

// GetParticipantJoinValidationRequestMessages returns captured participant join validation request messages
func (h *RoundTestHelper) GetParticipantJoinValidationRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantJoinValidationRequestedV1)
}

// GetParticipantStatusCheckErrorMessages returns captured participant status check error messages
func (h *RoundTestHelper) GetParticipantStatusCheckErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantStatusCheckErrorV1)
}

// ValidateParticipantRemovalRequest parses and validates a participant removal request message
func (h *RoundTestHelper) ValidateParticipantRemovalRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) *roundevents.ParticipantRemovalRequestPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantRemovalRequestPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant removal request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	return result
}

// ValidateParticipantJoinValidationRequest parses and validates a participant join validation request message
func (h *RoundTestHelper) ValidateParticipantJoinValidationRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) *roundevents.ParticipantJoinValidationRequestPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantJoinValidationRequestPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant join validation request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	return result
}

// ValidateParticipantStatusCheckError parses and validates a participant status check error message
func (h *RoundTestHelper) ValidateParticipantStatusCheckError(t *testing.T, msg *message.Message) *roundevents.ParticipantStatusCheckErrorPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantStatusCheckErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant status check error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// WaitForParticipantJoinRequest waits for participant join request messages
func (h *RoundTestHelper) WaitForParticipantJoinRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantJoinRequest, expectedCount, timeout)
}

// GetParticipantJoinRequestMessages returns captured participant join request messages
func (h *RoundTestHelper) GetParticipantJoinRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantJoinRequest)
}

// ValidateParticipantJoinRequest parses and validates a participant join request message
func (h *RoundTestHelper) ValidateParticipantJoinRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) *roundevents.ParticipantJoinRequestPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantJoinRequestPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant join request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	return result
}

// WaitForParticipantRemovalRequest waits for participant removal request messages
func (h *RoundTestHelper) WaitForParticipantRemovalRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantRemovalRequest, expectedCount, timeout)
}

func (h *RoundTestHelper) WaitForParticipantStatusCheckError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantStatusCheckError, expectedCount, timeout)
}

// CreateRoundWithParticipantInDB creates a round with an existing participant directly in the database
func (h *RoundTestHelper) CreateRoundWithParticipantInDB(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, existingResponse roundtypes.Response) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundData := generator.GenerateRound(DiscordID(userID), 0, []User{}) // Start with 0 participants

	// Generate a tag number for the participant (you might want to make this configurable)
	tagNumber := sharedtypes.TagNumber(generator.GenerateTagNumber())

	// Add the participant with the specified response
	participant := roundtypes.Participant{
		UserID:    userID,
		Response:  existingResponse,
		TagNumber: &tagNumber, // Set a tag number
		Score:     nil,        // Score can be nil for this test
	}
	roundData.Participants = []roundtypes.Participant{participant}

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round with participant: %v", err)
	}

	return roundData.ID
}

// CreateRoundWithParticipantAndTagInDB creates a round with a participant that has a specific tag number
func (h *RoundTestHelper) CreateRoundWithParticipantAndTagInDB(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, existingResponse roundtypes.Response, tagNumber *sharedtypes.TagNumber) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundData := generator.GenerateRound(DiscordID(userID), 0, []User{}) // Start with 0 participants

	// Add the participant with the specified response and tag number
	participant := roundtypes.Participant{
		UserID:    userID,
		Response:  existingResponse,
		TagNumber: tagNumber, // Use the provided tag number (can be nil)
		Score:     nil,
	}
	roundData.Participants = []roundtypes.Participant{participant}

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round with participant: %v", err)
	}

	return roundData.ID
}

// CreateRoundInDB creates a round directly in the database for testing
func (h *RoundTestHelper) CreateRoundInDB(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundData := generator.GenerateRound(DiscordID(userID), 0, []User{}) // 0 participants initially

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round: %v", err)
	}

	return roundData.ID
}

// CreateRoundInDBWithState creates a round directly in the database with a specific state
func (h *RoundTestHelper) CreateRoundInDBWithState(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, state roundtypes.RoundState) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundOptions := RoundOptions{
		CreatedBy:        DiscordID(userID),
		ParticipantCount: 0,
		Users:            []User{},
		State:            state, // Set the desired state
	}
	roundData := generator.GenerateRoundWithConstraints(roundOptions)

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round: %v", err)
	}

	return roundData.ID
}

// PublishParticipantStatusUpdateRequest publishes a ParticipantStatusUpdateRequest event and returns the message
func (h *RoundTestHelper) PublishParticipantStatusUpdateRequest(t *testing.T, ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundParticipantStatusUpdateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishParticipantRemovalRequest publishes a ParticipantRemovalRequest event and returns the message
func (h *RoundTestHelper) PublishParticipantRemovalRequest(t *testing.T, ctx context.Context, payload roundevents.ParticipantRemovalRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundParticipantRemovalRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishTagNumberFound publishes a TagNumberFound event and returns the message
func (h *RoundTestHelper) PublishTagNumberFound(t *testing.T, ctx context.Context, payload roundevents.RoundTagNumberFoundPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundTagNumberFound, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// PublishTagNumberNotFound publishes a TagNumberNotFound event and returns the message
func (h *RoundTestHelper) PublishTagNumberNotFound(t *testing.T, ctx context.Context, payload roundevents.RoundTagNumberNotFoundPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundTagNumberNotFound, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// WaitForParticipantJoinValidationRequest waits for participant join validation request messages
func (h *RoundTestHelper) WaitForParticipantJoinValidationRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantJoinValidationRequest, expectedCount, timeout)
}

// WaitForParticipantStatusUpdateRequest waits for participant status update request messages
func (h *RoundTestHelper) WaitForParticipantStatusUpdateRequest(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantStatusUpdateRequestedV1, expectedCount, timeout)
}

// WaitForParticipantJoinValidationError waits for participant join validation error messages
func (h *RoundTestHelper) WaitForParticipantJoinValidationError(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, expectedCount, timeout)
}

// GetParticipantStatusUpdateRequestMessages returns captured participant status update request messages
func (h *RoundTestHelper) GetParticipantStatusUpdateRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantStatusUpdateRequestedV1)
}

// GetParticipantJoinValidationErrorMessages returns captured participant join validation error messages
func (h *RoundTestHelper) GetParticipantJoinValidationErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

// ValidateParticipantJoinValidationError parses and validates a participant join validation error message
func (h *RoundTestHelper) ValidateParticipantJoinValidationError(t *testing.T, msg *message.Message) *roundevents.RoundParticipantJoinErrorPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant join validation error message: %v", err)
	}

	if len(result.Error) == 0 {
		t.Errorf("Expected error message to be populated")
	}

	return result
}

// WaitForParticipantJoined waits for participant joined messages
func (h *RoundTestHelper) WaitForParticipantJoined(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantJoinedV1, expectedCount, timeout)
}

// GetParticipantJoinedMessages returns captured participant joined messages
func (h *RoundTestHelper) GetParticipantJoinedMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantJoinedV1)
}

// ValidateParticipantJoined parses and validates a participant joined message
func (h *RoundTestHelper) ValidateParticipantJoined(t *testing.T, msg *message.Message) *roundevents.ParticipantJoinedPayloadV1 {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantJoinedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant joined message: %v", err)
	}

	return result
}

// WaitForLeaderboardTagLookup waits for leaderboard tag lookup request messages
func (h *RoundTestHelper) WaitForLeaderboardTagLookup(expectedCount int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.LeaderboardGetTagNumberRequest, expectedCount, timeout)
}

// GetLeaderboardTagLookupMessages returns captured leaderboard tag lookup request messages
func (h *RoundTestHelper) GetLeaderboardTagLookupMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.LeaderboardGetTagNumberRequest)
}

// ValidateLeaderboardTagLookup parses and validates a leaderboard tag lookup request message
func (h *RoundTestHelper) ValidateLeaderboardTagLookup(t *testing.T, msg *message.Message) *roundevents.TagLookupRequestPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.TagLookupRequestPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse leaderboard tag lookup request message: %v", err)
	}

	return result
}

// PublishLeaderboardTagLookupRequest publishes a LeaderboardGetTagNumberRequest event and returns the message
func (h *RoundTestHelper) PublishLeaderboardTagLookupRequest(t *testing.T, ctx context.Context, payload roundevents.TagLookupRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.LeaderboardGetTagNumberRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func (h *RoundTestHelper) PublishParticipantJoinValidationRequest(t *testing.T, ctx context.Context, payload *roundevents.ParticipantJoinValidationRequestPayload) *message.Message {
	t.Helper()

	t.Logf("PublishParticipantJoinValidationRequest: payload=%+v", payload)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	t.Logf("Marshaled payload: %d bytes", len(payloadBytes))

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	t.Logf("Created message: UUID=%s, CorrelationID=%s", msg.UUID, msg.Metadata.Get(middleware.CorrelationIDMetadataKey))

	if err := PublishMessage(t, h.eventBus, ctx, roundevents.RoundParticipantJoinValidationRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	t.Logf("Successfully published to topic: %s", roundevents.RoundParticipantJoinValidationRequest)
	return msg
}

func (h *RoundTestHelper) WaitForTagLookupRequest(count int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.LeaderboardGetTagNumberRequest, count, timeout)
}

func (h *RoundTestHelper) GetTagLookupRequestMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.LeaderboardGetTagNumberRequest)
}

func (h *RoundTestHelper) ValidateTagLookupRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.TagLookupRequestPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.TagLookupRequestPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse tag lookup request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return result
}

func (h *RoundTestHelper) ValidateParticipantStatusUpdateRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.ParticipantJoinRequestPayload {
	t.Helper()

	result, err := ParsePayload[roundevents.ParticipantJoinRequestPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant status update request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return result
}

func (h *RoundTestHelper) WaitForParticipantJoinError(count int, timeout time.Duration) bool {
	return h.capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, count, timeout)
}

func (h *RoundTestHelper) GetParticipantJoinErrorMessages() []*message.Message {
	return h.capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

func (h *RoundTestHelper) ValidateParticipantJoinError(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	t.Helper()

	result, err := ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant join error message: %v", err)
	}

	if result.ParticipantJoinRequest == nil {
		t.Error("Expected ParticipantJoinRequest to be populated in error payload")
		return
	}

	if result.ParticipantJoinRequest.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.ParticipantJoinRequest.RoundID)
	}

	if result.ParticipantJoinRequest.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.ParticipantJoinRequest.UserID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// CreateUpcomingRoundWithParticipantAndTagInDB creates an upcoming round with a participant that has a specific tag number
func (h *RoundTestHelper) CreateUpcomingRoundWithParticipantAndTagInDB(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, existingResponse roundtypes.Response, tagNumber *sharedtypes.TagNumber) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())

	// Use RoundOptions to explicitly set the state to Upcoming
	roundOptions := RoundOptions{
		CreatedBy:        DiscordID(userID),
		ParticipantCount: 0,
		Users:            []User{},
		State:            roundtypes.RoundStateUpcoming, // Explicitly set to upcoming
	}
	roundData := generator.GenerateRoundWithConstraints(roundOptions)

	// Add the participant with the specified response and tag number
	participant := roundtypes.Participant{
		UserID:    userID,
		Response:  existingResponse,
		TagNumber: tagNumber, // Use the provided tag number (can be nil)
		Score:     nil,
	}
	roundData.Participants = []roundtypes.Participant{participant}

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundtypes.RoundStateUpcoming, // Ensure it's upcoming
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round with participant: %v", err)
	}

	t.Logf("Created upcoming round %s with participant %s (tag: %v)", roundData.ID, userID, tagNumber)

	return roundData.ID
}

// Alternative simpler version if RoundOptions doesn't support StartTime:

// CreateRoundInDBWithTime creates a round directly in the database with a specific start time and state
func (h *RoundTestHelper) CreateRoundInDBWithTime(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, state roundtypes.RoundState, startTime *time.Time) sharedtypes.RoundID {
	t.Helper()

	generator := NewTestDataGenerator(time.Now().UnixNano())
	roundOptions := RoundOptions{
		CreatedBy:        DiscordID(userID),
		ParticipantCount: 0,
		Users:            []User{},
		State:            state,
	}
	roundData := generator.GenerateRoundWithConstraints(roundOptions)

	// Override the start time if provided
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		roundData.StartTime = &converted
	}

	// Convert to DB model
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild", // ✅ FIX: Must match handler test payloads
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round with time: %v", err)
	}

	return roundData.ID
}
