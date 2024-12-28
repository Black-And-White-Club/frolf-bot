package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/events"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"

	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserSignupRequest(t *testing.T) {
	tests := []struct {
		name             string
		req              userevents.UserSignupRequestPayload
		mockUserDB       func(context.Context, *gomock.Controller, *userdb.MockUserDB)
		mockEventBus     func(context.Context, *gomock.Controller, *eventbusmock.MockEventBus)
		mockLogger       func(*testutils.MockLoggerAdapter)
		want             *userevents.UserSignupResponsePayload
		wantErr          error
		checkTagCalled   bool
		publishTagCalled bool
	}{
		{
			name: "Successful Signup with Tag",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user123",
				TagNumber: 42,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}

						if user.DiscordID != "user123" {
							return fmt.Errorf("unexpected DiscordID: got %v, want %v", user.DiscordID, "user123")
						}
						if user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected Role: got %v, want %v", user.Role, usertypes.UserRoleRattler)
						}

						return nil
					}).Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// Expect Publish for CheckTagAvailabilityRequest
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						// You can validate the message content here if needed
						return nil
					}).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, types.Message) error) error {
						responsePayload := map[string]interface{}{
							"is_available": true,
						}

						// Pass the ctrl to the handler function
						if handler != nil {
							mockMsg := testutils.NewMockMessage(ctrl)
							mockMsg.EXPECT().Metadata().Return(message.Metadata{}).AnyTimes()
							mockMsg.EXPECT().UUID().Return("").AnyTimes()

							mockMsg.EXPECT().Payload().Return(func() []byte {
								payloadBytes, err := json.Marshal(responsePayload)
								if err != nil {
									t.Fatalf("Failed to marshal response payload: %v", err)
								}
								return payloadBytes
							}()).Times(1)
							err := handler(ctx, mockMsg)
							if err != nil {
								return fmt.Errorf("handler returned an error: %v", err)
							}
						}
						return nil
					}).
					Times(1)

				// Expect Publish for TagAssignedRequest
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.TagAssignedRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						// You can validate the message content here if needed
						return nil
					}).
					Times(1)
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Before checkTagAvailability", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Generated replySubject", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1) // Expect this log
				mockLogger.EXPECT().Error("Response channel full", nil, gomock.Any()).Times(0)
			},
			want: &userevents.UserSignupResponsePayload{
				Success: true,
			},
			wantErr: nil,
		},
		{
			name: "Successful Signup without Tag",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user456",
				TagNumber: 0, // No tag requested
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}

						if user.DiscordID != "user456" {
							return fmt.Errorf("unexpected DiscordID: got %v, want %v", user.DiscordID, "user456")
						}
						if user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected Role: got %v, want %v", user.Role, usertypes.UserRoleRattler)
						}

						return nil
					}).Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// Expect no calls to event bus methods when no tag is requested
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
			},
			want: &userevents.UserSignupResponsePayload{
				Success: true,
			},
			wantErr:          nil,
			checkTagCalled:   false,
			publishTagCalled: false,
		},
		{
			name: "Failed Signup - CreateUser Error",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user789",
				TagNumber: 0,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}

						if user.DiscordID != "user789" {
							return fmt.Errorf("unexpected DiscordID: got %v, want %v", user.DiscordID, "user789")
						}
						if user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected Role: got %v, want %v", user.Role, usertypes.UserRoleRattler)
						}

						return errors.New("database error")
					}).Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// Expect no calls to event bus methods when user creation fails
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
			},
			want:             nil,
			wantErr:          fmt.Errorf("failed to create user: %w", errors.New("database error")),
			checkTagCalled:   false,
			publishTagCalled: false,
		},
		{
			name: "Failed Signup - Tag Not Available",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user101",
				TagNumber: 99,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}

						if user.DiscordID != "user101" {
							return fmt.Errorf("unexpected DiscordID: got %v, want %v", user.DiscordID, "user101")
						}
						if user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected Role: got %v, want %v", user.Role, usertypes.UserRoleRattler)
						}

						return nil
					}).Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						return nil
					}).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, types.Message) error) error {
						responsePayload := map[string]interface{}{
							"is_available": false,
						}

						// Pass the ctrl to the handler function
						if handler != nil {
							mockMsg := testutils.NewMockMessage(ctrl)
							mockMsg.EXPECT().Metadata().Return(message.Metadata{}).AnyTimes()
							mockMsg.EXPECT().UUID().Return("").AnyTimes()

							mockMsg.EXPECT().Payload().Return(func() []byte {
								payloadBytes, err := json.Marshal(responsePayload)
								if err != nil {
									t.Fatalf("Failed to marshal response payload: %v", err)
								}
								return payloadBytes
							}()).Times(1)
							err := handler(ctx, mockMsg)
							if err != nil {
								return fmt.Errorf("handler returned an error: %v", err)
							}
						}
						return nil
					}).
					Times(1)
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Before checkTagAvailability", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1) // Expect this log
				mockLogger.EXPECT().Debug("Generated replySubject", gomock.Any()).Times(1)
				mockLogger.EXPECT().Error("Response channel full", nil, gomock.Any()).Times(0)
			},
			want: &userevents.UserSignupResponsePayload{
				Success: true,
			},
			wantErr: nil,
		},
		{
			name: "Failed Signup - Tag Check Error",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user102",
				TagNumber: 99,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				// No CreateUser expectation needed as it should not be called
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						// Simulate an error during tag availability check
						return errors.New("tag check error")
					}).
					Times(1)

				// No Subscribe expectation needed as Publish returns an error
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Before checkTagAvailability", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Generated replySubject", gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(gomock.Eq("failed to check tag availability"), gomock.Any(), gomock.Any())
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to check tag availability: %w", errors.New("failed to publish CheckTagAvailabilityRequest: tag check error")),
		},
		{
			name: "Failed Signup - Publish TagAssignedRequest Error",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user103",
				TagNumber: 100,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}

						if user.DiscordID != "user103" {
							return fmt.Errorf("unexpected DiscordID: got %v, want %v", user.DiscordID, "user103")
						}
						if user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected Role: got %v, want %v", user.Role, usertypes.UserRoleRattler)
						}

						return nil
					}).Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						return nil
					}).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, types.Message) error) error {
						responsePayload := map[string]interface{}{
							"is_available": true,
						}

						if handler != nil {
							mockMsg := testutils.NewMockMessage(ctrl)
							mockMsg.EXPECT().Metadata().Return(message.Metadata{}).AnyTimes()
							mockMsg.EXPECT().UUID().Return("").AnyTimes()

							mockMsg.EXPECT().Payload().Return(func() []byte {
								payloadBytes, err := json.Marshal(responsePayload)
								if err != nil {
									t.Fatalf("Failed to marshal response payload: %v", err)
								}
								return payloadBytes
							}()).Times(1)

							err := handler(ctx, mockMsg)
							if err != nil {
								return fmt.Errorf("handler returned an error: %v", err)
							}
						}
						return nil
					}).
					Times(1)

				// Mock the publishTagAssigned call to return an error
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.TagAssignedRequest), gomock.Any()).
					Return(errors.New("publish error")) // Simulate publish error
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Before checkTagAvailability", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Debug("Generated replySubject", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1)
				// Expect only one error log for the publishTagAssigned error
				mockLogger.EXPECT().Error(gomock.Eq("failed to publish TagAssignedRequest event"), gomock.Any(), gomock.Any())
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to publish TagAssignedRequest event: %w", errors.New("failed to publish TagAssignedRequest event: publish error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserDB := userdb.NewMockUserDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			ctx := context.Background()

			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, ctrl, mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, ctrl, mockEventBus)
			}
			if tt.mockLogger != nil {
				tt.mockLogger(mockLogger)
			}

			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   mockLogger,
			}

			got, err := service.OnUserSignupRequest(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.wantErr == nil {
					t.Errorf("Expected no error, but got: %v", err)
				} else if err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected result: %v, got: %v", tt.want, got)
			}
		})
	}
}
