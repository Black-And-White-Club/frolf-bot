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
			name: "Successful UpdateUserRole",
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

						if payload.DiscordID != usertypes.DiscordID(testDiscordID) || payload.Role != testRole || payload.RequesterID != testRequesterID {
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

			if err := s.UpdateUserRole(tt.args.ctx, tt.args.msg, usertypes.DiscordID(tt.args.discordID), tt.args.role, tt.args.requesterID); (err != nil) != tt.wantErr {
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
		ctx           context.Context
		discordID     string
		role          string
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
			name: "Successful Role Update",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          string(testRole), // Convert to string for the test
				correlationID: testCorrelationID,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				f.UserDB.EXPECT().
					UpdateUserRole(args.ctx, usertypes.DiscordID(args.discordID), testRole).
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
				ctx:           testCtx,
				discordID:     testDiscordID,
				role:          string(testRole), // Convert to string for the test
				correlationID: testCorrelationID,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				f.UserDB.EXPECT().
					UpdateUserRole(args.ctx, usertypes.DiscordID(args.discordID), testRole).
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

			if err := s.UpdateUserRoleInDatabase(tt.args.ctx, tt.args.discordID, string(usertypes.UserRoleEnum(tt.args.role)), tt.args.correlationID); (err != nil) != tt.wantErr {
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
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx       context.Context
		msg       *message.Message
		discordID string
		role      string
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
				ctx:       testCtx,
				msg:       message.NewMessage(watermill.NewUUID(), nil), // New message with new UUID
				discordID: testDiscordID,
				role:      testRole,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserRoleUpdated, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserRoleUpdated {
							t.Errorf("Expected topic %s, got %s", userevents.UserRoleUpdated, topic)
						}

						var payload userevents.UserRoleUpdatedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != testDiscordID || payload.Role != testRole {
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
				ctx:       testCtx,
				msg:       message.NewMessage(watermill.NewUUID(), nil), // New message with new UUID
				discordID: testDiscordID,
				role:      testRole,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserRoleUpdated, gomock.Any()).
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

			if err := s.PublishUserRoleUpdated(tt.args.ctx, tt.args.msg, tt.args.discordID, string(usertypes.UserRoleEnum(tt.args.role))); (err != nil) != tt.wantErr {
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
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx       context.Context
		msg       *message.Message
		discordID string
		role      string
		reason    string
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
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil), // Using correlationID as message ID
				discordID: testDiscordID,
				role:      testRole,
				reason:    testReason,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserRoleUpdateFailed, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.UserRoleUpdateFailed {
							t.Errorf("Expected topic %s, got %s", userevents.UserRoleUpdateFailed, topic)
						}

						var payload userevents.UserRoleUpdateFailedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != testDiscordID || payload.Role != testRole || payload.Reason != testReason {
							t.Errorf("Payload does not match expected values")
						}

						if msg.UUID != testCorrelationID {
							t.Errorf("Expected message UUID %s, got %s", testCorrelationID, msg.UUID)
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
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil), // Using correlationID as message ID
				discordID: testDiscordID,
				role:      testRole,
				reason:    testReason,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.UserRoleUpdateFailed, gomock.Any()).
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

			if err := s.PublishUserRoleUpdateFailed(tt.args.ctx, tt.args.msg, tt.args.discordID, string(usertypes.UserRoleEnum(tt.args.role)), tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.PublishUserRoleUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
