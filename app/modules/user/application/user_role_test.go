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

func TestUserServiceImpl_UpdateUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := "admin"
	testRequesterID := "requester456"
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB    *userdb.MockUserDB
		eventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx         context.Context
		msg         *message.Message
		discordID   usertypes.DiscordID
		role        string
		requesterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful UpdateUserRole",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil),
				discordID:   usertypes.DiscordID(testDiscordID),
				role:        testRole,
				requesterID: testRequesterID,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckRequest, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserPermissionsCheckRequest {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckRequest, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserPermissionsCheckRequestPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) || payload.Role != usertypes.UserRoleEnum(testRole) || payload.RequesterID != testRequesterID {
							t.Errorf("Payload does not match expected values")
						}

						if correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
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
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil),
				discordID:   usertypes.DiscordID(testDiscordID),
				role:        testRole,
				requesterID: testRequesterID,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckRequest, gomock.Any()).
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

			if err := s.UpdateUserRole(tt.args.ctx, tt.args.msg, tt.args.discordID, usertypes.UserRoleEnum(tt.args.role), tt.args.requesterID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.UpdateUserRole() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_UpdateUserRoleInDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl) // Assuming you don't need eventBus in this test
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := usertypes.UserRoleEnum("admin") // Ensure this type is used
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus // Now a mock
		logger   *slog.Logger
	}
	type args struct {
		ctx       context.Context
		msg       *message.Message // Add msg argument
		discordID string
		role      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Role Update",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil), // Create message
				discordID: testDiscordID,
				role:      string(testRole), // Convert to string for the test
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.UserDB.EXPECT().
					UpdateUserRole(a.ctx, usertypes.DiscordID(a.discordID), testRole).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Database Error on Role Update",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil), // Create message
				discordID: testDiscordID,
				role:      string(testRole), // Convert to string for the test
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.UserDB.EXPECT().
					UpdateUserRole(a.ctx, usertypes.DiscordID(a.discordID), testRole).
					Return(errors.New("database error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				UserDB:   tt.fields.UserDB,
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}

			// Call setup function to configure mocks before each test case
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			if err := s.UpdateUserRoleInDatabase(tt.args.ctx, tt.args.msg, usertypes.DiscordID(tt.args.discordID), usertypes.UserRoleEnum(tt.args.role)); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.UpdateUserRoleInDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_PublishUserRoleUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := "admin"
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB    *userdb.MockUserDB
		eventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx    context.Context
		msg    *message.Message
		userID string
		role   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Publish",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:    testCtx,
				msg:    message.NewMessage(testCorrelationID, nil), // New message with new UUID
				userID: testDiscordID,
				role:   testRole,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.eventBus.EXPECT().
					Publish(userevents.UserRoleUpdated, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserRoleUpdated {
							t.Errorf("Expected topic %s, got %s", userevents.UserRoleUpdated, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserRoleUpdatedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) || payload.Role != usertypes.UserRoleEnum(testRole) {
							t.Errorf("Payload does not match expected values")
						}

						if correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:    testCtx,
				msg:    message.NewMessage(testCorrelationID, nil), // New message with new UUID
				userID: testDiscordID,
				role:   testRole,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.eventBus.EXPECT().
					Publish(userevents.UserRoleUpdated, gomock.Any()).
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

			if err := s.PublishUserRoleUpdated(tt.args.ctx, tt.args.msg, usertypes.DiscordID(tt.args.userID), usertypes.UserRoleEnum(tt.args.role)); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserRoleUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_PublishUserRoleUpdateFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := "admin"
	testReason := "Test reason"
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB    *userdb.MockUserDB
		eventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil // Use actual interface
	}
	type args struct {
		ctx    context.Context
		msg    *message.Message
		userID string
		role   string
		reason string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Publish",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:    testCtx,
				msg:    message.NewMessage(testCorrelationID, nil), // Using correlationID as message ID
				userID: testDiscordID,
				role:   testRole,
				reason: testReason,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(userevents.UserRoleUpdateFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserRoleUpdateFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserRoleUpdateFailed, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserRoleUpdateFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) || payload.Role != usertypes.UserRoleEnum(testRole) || payload.Reason != testReason {
							t.Errorf("Payload does not match expected values")
						}

						if correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use actual implementation
			},
			args: args{
				ctx:    testCtx,
				msg:    message.NewMessage(testCorrelationID, nil), // Using correlationID as message ID
				userID: testDiscordID,
				role:   testRole,
				reason: testReason,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(userevents.UserRoleUpdateFailed, gomock.Any()).
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
				eventUtil: tt.fields.eventUtil, // Use actual implementation
			}

			// Call setup function to configure mocks before each test case
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			if err := s.PublishUserRoleUpdateFailed(tt.args.ctx, tt.args.msg, usertypes.DiscordID(tt.args.userID), usertypes.UserRoleEnum(tt.args.role), tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserRoleUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
