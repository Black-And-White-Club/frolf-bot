package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserSignupRequest(t *testing.T) {
	tests := []struct {
		name             string
		req              userevents.UserSignupRequestPayload
		mockUserDB       func(context.Context, *gomock.Controller, *userdb.MockUserDB)
		mockEventBus     func(context.Context, *gomock.Controller, *eventbusmock.MockEventBus)
		mockLogger       func(*testutils.MockLoggerAdapter)
		mockCheckTag     func(*gomock.Controller, *eventbusmock.MockEventBus, bool, error)
		want             *userevents.UserSignupResponsePayload
		wantErr          bool
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
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.TagAssignedRequest), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType shared.EventType, msg shared.Message) error {
						return nil
					}).
					Times(1)
			},
			mockCheckTag: func(ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus, available bool, err error) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, shared.Message) error) error {
						if err == nil {
							responsePayload := userevents.CheckTagAvailabilityResponsePayload{
								IsAvailable: available,
							}
							payloadBytes, _ := json.Marshal(responsePayload)
							responseMsg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadBytes)

							// Directly call the handler function to simulate the response
							go func() {
								handler(ctx, responseMsg)
							}()
						}
						return nil
					}).AnyTimes()
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1)
			},
			want: &userevents.UserSignupResponsePayload{
				Success: true,
			},
			wantErr: false,
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
			wantErr:          false,
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
			wantErr:          true,
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
				// No CreateUser expectation when the tag is not available
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// No additional expectations for event bus here, as they are set in mockCheckTag
			},
			mockCheckTag: func(ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus, available bool, err error) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, shared.Message) error) error {
						if err == nil {
							responsePayload := userevents.CheckTagAvailabilityResponsePayload{
								IsAvailable: false, // Tag is not available
							}
							payloadBytes, _ := json.Marshal(responsePayload)
							responseMsg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadBytes)

							// Directly call the handler function to simulate the response
							go func() {
								handler(ctx, responseMsg)
							}()
						}
						return nil
					}).AnyTimes()
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failed Signup - Tag Check Error",
			req: userevents.UserSignupRequestPayload{
				DiscordID: "user102",
				TagNumber: 99,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				// No CreateUser expectation because tag check fails
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					Return(errors.New("tag check error")).
					Times(1)

				// No Subscribe calls expected
				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(
					gomock.Eq("failed to publish CheckTagAvailabilityRequest"),
					gomock.AssignableToTypeOf(errors.New("")),
					gomock.Eq(shared.LogFields{"error": errors.New("tag check error")}),
				).Times(1)
			},
			want:    nil,
			wantErr: true,
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
				// Expect the Publish call for TagAssignedRequest and make it return an error.
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.TagAssignedRequest), gomock.Any()).
					Return(errors.New("publish error")).
					Times(1) // We expect this to be called once
			},
			mockCheckTag: func(ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus, available bool, err error) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.CheckTagAvailabilityRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, subject string, handler func(context.Context, shared.Message) error) error {
						if err == nil {
							responsePayload := userevents.CheckTagAvailabilityResponsePayload{
								IsAvailable: true, // Tag is available
							}
							payloadBytes, _ := json.Marshal(responsePayload)
							responseMsg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadBytes)

							// Directly call the handler function to simulate the response
							go func() {
								handler(ctx, responseMsg)
							}()
						}
						return nil
					}).AnyTimes()
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info("OnUserSignupRequest started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("checkTagAvailability started", gomock.Any()).Times(1)
				mockLogger.EXPECT().Info("publishTagAssigned called", gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(gomock.Eq("failed to publish TagAssignedRequest event"), gomock.Any(), gomock.Any()).Times(1)
			},
			want:    nil,
			wantErr: true,
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
			if tt.mockCheckTag != nil {
				if tt.name == "Failed Signup - Tag Check Error" {
					// Pass false for available to simulate failure
					tt.mockCheckTag(ctrl, mockEventBus, false, errors.New("tag check error"))
				} else {
					// For other cases, assume tag is available
					tt.mockCheckTag(ctrl, mockEventBus, true, nil)
				}
			}

			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   mockLogger,
			}

			got, err := service.OnUserSignupRequest(ctx, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("OnUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnUserSignupRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
