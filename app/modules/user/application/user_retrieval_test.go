package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	userdbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := usertypes.DiscordID("12345")
	testRole := usertypes.UserRoleAdmin
	testCorrelationID := watermill.NewUUID()
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
			name: "Successful GetUserRole",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.UserDB.EXPECT().
					GetUserRole(a.ctx, testDiscordID).
					Return(testRole, nil).
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserRoleResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.GetUserRoleResponse {
							t.Errorf("Expected topic %s, got %s", userevents.GetUserRoleResponse, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.GetUserRoleResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.DiscordID != testDiscordID {
							t.Errorf("Expected User ID %s, got %s", testDiscordID, payload.DiscordID)
						}

						if usertypes.UserRoleEnum(payload.Role) != testRole {
							t.Errorf("Expected role %s, got %s", testRole, payload.Role)
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
			name: "Database Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: true, // Expecting an error
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.UserDB.EXPECT().
					GetUserRole(a.ctx, testDiscordID).
					Return(usertypes.UserRoleEnum(""), errors.New("database error")).
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserRoleFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.GetUserRoleFailed {
							t.Errorf("Expected topic %s, got %s", userevents.GetUserRoleFailed, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish Event Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: true, // Expecting an error
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.UserDB.EXPECT().
					GetUserRole(a.ctx, testDiscordID).
					Return(testRole, nil).
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserRoleResponse, gomock.Any()).
					Return(errors.New("publish error")). // Simulate a publish error
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

			if err := s.GetUserRole(tt.args.ctx, tt.args.msg, tt.args.discordID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.GetUserRole() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := usertypes.DiscordID("12345")
	testCorrelationID := watermill.NewUUID()
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
			name: "Successful GetUser",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				testUser := &userdbtypes.User{
					ID:        1,
					DiscordID: testDiscordID,
					Role:      usertypes.UserRoleAdmin,
				}

				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, testDiscordID).
					Return(testUser, nil).
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.GetUserResponse {
							t.Errorf("Expected topic %s, got %s", userevents.GetUserResponse, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						var payload userevents.GetUserResponsePayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						expectedUser := &usertypes.UserData{
							ID:        testUser.ID,
							DiscordID: testUser.DiscordID,
							Role:      testUser.Role,
						}

						// Compare the entire UserData struct
						if !reflect.DeepEqual(payload.User, expectedUser) {
							t.Errorf("Expected user %+v, got %+v", expectedUser, payload.User)
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
			name: "Database Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: false, // Now we don't expect an error because we handle it by publishing an event
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, testDiscordID).
					Return(nil, userdbtypes.ErrUserNotFound). // Return the specific error
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != userevents.GetUserFailed {
							t.Errorf("Expected topic %s, got %s", userevents.GetUserFailed, topic)
						}
						// Add assertions for message content if necessary
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish Event Error",
			fields: fields{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx:       testCtx,
				msg:       message.NewMessage(testCorrelationID, nil),
				discordID: testDiscordID,
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				testUser := &userdbtypes.User{
					ID:        1,
					DiscordID: testDiscordID,
					Role:      usertypes.UserRoleAdmin,
				}

				f.UserDB.EXPECT().
					GetUserByDiscordID(a.ctx, testDiscordID).
					Return(testUser, nil).
					Times(1)

				f.eventBus.EXPECT().
					Publish(userevents.GetUserResponse, gomock.Any()).
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

			if err := s.GetUser(tt.args.ctx, tt.args.msg, tt.args.discordID); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.GetUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
