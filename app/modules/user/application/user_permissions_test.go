package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CheckUserPermissions(t *testing.T) {
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
		discordID   string
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
			name: "Successful Permission Check",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Use the actual implementation
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil),
				discordID:   testDiscordID,
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

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) {
							t.Errorf("Expected user ID %s, got %s", testDiscordID, payload.DiscordID)
						}

						if payload.Role != testRole {
							t.Errorf("Expected role %s, got %s", testRole, payload.Role)
						}

						if payload.RequesterID != testRequesterID {
							t.Errorf("Expected requester ID %s, got %s", testRequesterID, payload.RequesterID)
						}

						// Check correlation ID
						correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
						if correlationID != testCorrelationID {
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
				eventUtil: eventutil.NewEventUtil(), // Use the actual implementation
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil),
				discordID:   testDiscordID,
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

			if err := s.CheckUserPermissions(tt.args.ctx, tt.args.msg, usertypes.DiscordID(tt.args.discordID), usertypes.UserRoleEnum(tt.args.role), tt.args.requesterID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.CheckUserPermissions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestUserServiceImpl_CheckUserPermissionsInDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := usertypes.UserRoleEnum("admin")
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
		msg         *message.Message // Add msg argument
		discordID   usertypes.DiscordID
		role        usertypes.UserRoleEnum
		requesterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Permission Check in DB",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil), // Create a message with correlation ID
				discordID:   usertypes.DiscordID(testDiscordID),
				role:        testRole,
				requesterID: testRequesterID,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID) // Set correlation ID in metadata

				// Mock successful retrieval of user
				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, usertypes.DiscordID(a.requesterID)).
					Return(&userdbtypes.User{
						ID:        1,
						DiscordID: usertypes.DiscordID(a.requesterID),
						Role:      testRole,
					}, nil).
					Times(1)

				// Expect the UserPermissionsCheckResponse event to be published
				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserPermissionsCheckResponse {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckResponse, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserPermissionsCheckResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != string(a.discordID) || payload.Role != string(a.role) || payload.RequesterID != a.requesterID || !payload.HasPermission {
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
			name: "Requester Not Found",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil), // Create a message with correlation ID
				discordID:   usertypes.DiscordID(testDiscordID),
				role:        testRole,
				requesterID: testRequesterID,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID) // Set correlation ID in metadata

				// Mock user retrieval returning an error
				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, usertypes.DiscordID(a.requesterID)).
					Return(nil, fmt.Errorf("requester not found")).
					Times(1)

				// Expect the UserPermissionsCheckFailed event to be published
				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserPermissionsCheckFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckFailed, topic)
						}
						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserPermissionsCheckFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						expectedReason := "Failed to get requesting user"
						if payload.Reason != expectedReason {
							t.Errorf("Expected reason '%s', got '%s'", expectedReason, payload.Reason)
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
			name: "Requester Does Not Have Required Role",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:         testCtx,
				msg:         message.NewMessage(testCorrelationID, nil),
				discordID:   usertypes.DiscordID(testDiscordID),
				role:        testRole,
				requesterID: testRequesterID,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Mock successful retrieval of user with incorrect role
				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, usertypes.DiscordID(a.requesterID)).
					Return(&userdbtypes.User{
						ID:        1,
						DiscordID: usertypes.DiscordID(a.requesterID),
						Role:      "some_other_role", // Assuming this is different from testRole
					}, nil).
					Times(1)

				// Expect the UserPermissionsCheckFailed event to be published
				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserPermissionsCheckFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckFailed, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserPermissionsCheckFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						expectedReason := "Requester does not have required role"
						if payload.Reason != expectedReason {
							t.Errorf("Expected reason '%s', got '%s'", expectedReason, payload.Reason)
						}

						if correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
						}

						return nil
					}).
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

			if err := s.CheckUserPermissionsInDB(tt.args.ctx, tt.args.msg, tt.args.discordID, tt.args.role, tt.args.requesterID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.CheckUserPermissionsInDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_PublishUserPermissionsCheckResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := usertypes.DiscordID("123456789012345678")
	testRole := usertypes.UserRoleEnum("admin")
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
		ctx           context.Context
		msg           *message.Message
		discordID     usertypes.DiscordID
		role          usertypes.UserRoleEnum
		requesterID   string
		hasPermission bool
		reason        string
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
				ctx:           testCtx,
				msg:           message.NewMessage(testCorrelationID, nil), // Create a message with correlation ID
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				hasPermission: true,
				reason:        "",
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID) // Set correlation ID in metadata

				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.UserPermissionsCheckResponse {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckResponse, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.UserPermissionsCheckResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != string(a.discordID) || payload.Role != string(a.role) || payload.RequesterID != a.requesterID || !payload.HasPermission {
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
				ctx:           testCtx,
				msg:           message.NewMessage(testCorrelationID, nil), // Create a message with correlation ID
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				hasPermission: true,
				reason:        "",
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.eventBus.EXPECT().
					Publish(userevents.UserPermissionsCheckResponse, gomock.Any()).
					Return(fmt.Errorf("publish error")).
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

			if err := s.PublishUserPermissionsCheckResponse(tt.args.ctx, tt.args.msg, tt.args.discordID, tt.args.role, tt.args.requesterID, tt.args.hasPermission, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserPermissionsCheckResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
