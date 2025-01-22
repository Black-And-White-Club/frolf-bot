package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
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
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
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
					Publish(args.ctx, userevents.UserPermissionsCheckRequest, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckRequest {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckRequest, topic)
						}

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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
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
					Publish(args.ctx, userevents.UserPermissionsCheckRequest, gomock.Any()).
					Return(errors.New("publish error")).
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
	testRole := "admin"
	testRequesterID := "requester456"
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx           context.Context
		discordID     string
		role          string
		requesterID   string
		correlationID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful Permission Check in DB",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				// Mock successful retrieval of user
				f.UserDB.EXPECT().
					GetUserByDiscordID(args.ctx, usertypes.DiscordID(testRequesterID)).
					Return(&usertypes.UserData{
						ID:        1,
						Name:      "test",
						DiscordID: usertypes.DiscordID(testRequesterID),
						Role:      usertypes.UserRoleEnum(testRole),
					}, nil).
					Times(1)

				// Expect the UserPermissionsCheckResponse event to be published
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckResponse, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckResponse {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckResponse, topic)
						}

						var payload userevents.UserPermissionsCheckResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != testDiscordID || payload.Role != testRole || payload.RequesterID != testRequesterID || !payload.HasPermission {
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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				// Mock user retrieval returning an error
				f.UserDB.EXPECT().
					GetUserByDiscordID(args.ctx, usertypes.DiscordID(testRequesterID)).
					Return(nil, errors.New("requester not found")).
					Times(1)

				// Expect the UserPermissionsCheckFailed event to be published
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckFailed, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckFailed, topic)
						}

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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				// Mock successful retrieval of user with incorrect role
				f.UserDB.EXPECT().
					GetUserByDiscordID(args.ctx, usertypes.DiscordID(testRequesterID)).
					Return(&usertypes.UserData{
						ID:        1,
						Name:      "test",
						DiscordID: usertypes.DiscordID(testRequesterID),
						Role:      "some_other_role", // Assuming this is different from testRole
					}, nil).
					Times(1)

				// Expect the UserPermissionsCheckFailed event to be published
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckFailed, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckFailed, topic)
						}

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
				UserDB:   tt.fields.UserDB,
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}

			// Call setup function to configure mocks before each test case
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			if err := s.CheckUserPermissionsInDB(tt.args.ctx, usertypes.DiscordID(tt.args.discordID), usertypes.UserRoleEnum(tt.args.role), tt.args.requesterID, tt.args.correlationID); (err != nil) != tt.wantErr {
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

	testDiscordID := "123456789012345678"
	testRole := "admin"
	testRequesterID := "requester456"
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx           context.Context
		discordID     string
		role          string
		requesterID   string
		correlationID string
		hasPermission bool
		reason        string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful Publish",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
				hasPermission: true,
				reason:        "",
			},
			wantErr: false,
			setup: func(f fields, args args) {
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckResponse, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckResponse {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckResponse, topic)
						}

						var payload userevents.UserPermissionsCheckResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != testDiscordID || payload.Role != testRole || payload.RequesterID != testRequesterID || !payload.HasPermission {
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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
				hasPermission: true,
				reason:        "",
			},
			wantErr: true,
			setup: func(f fields, args args) {
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckResponse, gomock.Any()).
					Return(errors.New("publish error")).
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

			if err := s.PublishUserPermissionsCheckResponse(tt.args.ctx, usertypes.DiscordID(tt.args.discordID), usertypes.UserRoleEnum(tt.args.role), tt.args.requesterID, tt.args.correlationID, tt.args.hasPermission, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserPermissionsCheckResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_PublishUserPermissionsCheckFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678"
	testRole := "admin"
	testRequesterID := "requester456"
	testCorrelationID := watermill.NewUUID()
	testReason := "Permission denied"
	testCtx := context.Background()

	type fields struct {
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx           context.Context
		discordID     string
		role          string
		requesterID   string
		correlationID string
		reason        string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful Publish",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
				reason:        testReason,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckFailed, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserPermissionsCheckFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserPermissionsCheckFailed, topic)
						}

						var payload userevents.UserPermissionsCheckFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) || payload.Role != testRole || payload.RequesterID != testRequesterID || payload.Reason != testReason {
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
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          testRole,
				requesterID:   testRequesterID,
				correlationID: testCorrelationID,
				reason:        testReason,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserPermissionsCheckFailed, gomock.Any()).
					Return(errors.New("publish error")).
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

			if err := s.PublishUserPermissionsCheckFailed(tt.args.ctx, usertypes.DiscordID(tt.args.discordID), usertypes.UserRoleEnum(tt.args.role), tt.args.requesterID, tt.args.correlationID, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserPermissionsCheckFailed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
