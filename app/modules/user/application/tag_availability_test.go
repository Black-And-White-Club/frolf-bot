package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CheckTagAvailability(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testTagNumber := 123
	testCorrelationID := watermill.NewUUID()
	testDiscordID := "123456789012345678" // Valid Discord ID
	testCtx := context.Background()

	type fields struct {
		UserDB    *userdb.MockUserDB
		eventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx       context.Context
		msg       *message.Message
		tagNumber int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful CheckTagAvailability",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       createTestMessageWithMetadata(testCorrelationID, testDiscordID),
				tagNumber: testTagNumber,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.eventBus.EXPECT().
					Publish(userevents.LeaderboardTagAvailabilityCheckRequest, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.LeaderboardTagAvailabilityCheckRequest {
							t.Errorf("Expected topic %s, got %s", userevents.LeaderboardTagAvailabilityCheckRequest, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.TagAvailabilityCheckRequestedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.TagNumber != testTagNumber {
							t.Errorf("Expected tag number %d, got %d", testTagNumber, payload.TagNumber)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) {
							t.Errorf("Expected Discord ID %s, got %s", testDiscordID, payload.DiscordID)
						}

						// Check correlation ID
						if mid := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); mid != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, mid)
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Failed to Publish Event",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       createTestMessageWithMetadata(testCorrelationID, testDiscordID),
				tagNumber: testTagNumber,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.eventBus.EXPECT().
					Publish(userevents.LeaderboardTagAvailabilityCheckRequest, gomock.Any()).
					Return(errors.New("publish error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				UserDB:    tt.fields.UserDB,
				eventBus:  tt.fields.eventBus,
				logger:    tt.fields.logger,
				eventUtil: tt.fields.eventUtil,
			}

			// Call setup function to configure mocks before each test case
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			if err := s.CheckTagAvailability(tt.args.ctx, tt.args.msg, tt.args.tagNumber, usertypes.DiscordID(testDiscordID)); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.CheckTagAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_HandleTagUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	// Use a valid Discord ID format
	testDiscordID := "123456789012345678"
	testTagNumber := 123
	testCorrelationID := watermill.NewUUID()
	testReason := "tag not available"

	type fields struct {
		UserDB    *userdb.MockUserDB
		eventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx       context.Context
		msg       *message.Message
		tagNumber int
		discordID usertypes.DiscordID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Tag Unavailable",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventUtil,
			},
			args: args{
				ctx: context.Background(),
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagUnavailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID), // Set a valid Discord ID here
					TagNumber: testTagNumber,
					Reason:    testReason,
				}),
				tagNumber: testTagNumber,
				discordID: usertypes.DiscordID(testDiscordID), // Pass it here
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Expect the service to publish a UserCreationFailed event
				f.eventBus.EXPECT().
					Publish(userevents.TagUnavailable, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.TagUnavailable {
							t.Errorf("Expected topic %s, got %s", userevents.TagUnavailable, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						// Verify the correlation ID
						correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
						if correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
						}

						// Verify the payload
						var payload userevents.UserCreationFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("Failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) {
							t.Errorf("Expected Discord ID %s, got %s", testDiscordID, payload.DiscordID)
						}

						if payload.TagNumber == nil || *payload.TagNumber != testTagNumber {
							t.Errorf("Expected TagNumber %d, got %v", testTagNumber, payload.TagNumber)
						}

						if payload.Reason != testReason {
							t.Errorf("Expected Reason '%s', got '%s'", testReason, payload.Reason)
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish UserCreationFailed Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventUtil,
			},
			args: args{
				ctx: context.Background(),
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagUnavailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID), // Set a valid Discord ID here
					TagNumber: testTagNumber,
					Reason:    testReason,
				}),
				tagNumber: testTagNumber,
				discordID: usertypes.DiscordID(testDiscordID), // Pass it here
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Expect the service to publish a UserCreationFailed event, but simulate an error
				f.eventBus.EXPECT().
					Publish(userevents.TagUnavailable, gomock.Any()).
					Return(errors.New("publish error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				UserDB:    tt.fields.UserDB,
				eventBus:  tt.fields.eventBus,
				logger:    tt.fields.logger,
				eventUtil: tt.fields.eventUtil,
			}
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}
			if err := s.TagUnavailable(tt.args.ctx, tt.args.msg, tt.args.tagNumber, tt.args.discordID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.HandleTagUnavailable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create a test message with a payload and correlation ID in metadata
func createTestMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	return msg
}

// Helper function to create a test message with metadata
func createTestMessageWithMetadata(correlationID string, discordID string) *message.Message {
	msg := message.NewMessage(watermill.NewUUID(), nil)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	msg.Metadata.Set("user_id", discordID)
	return msg
}
