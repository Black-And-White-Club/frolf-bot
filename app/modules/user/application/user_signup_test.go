package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserSignupRequest(t *testing.T) {
	tests := []struct {
		name             string
		req              userevents.UserSignupRequestPayload
		mockUserDB       func(context.Context, *gomock.Controller, *userdb.MockUserDB)
		mockEventBus     func(context.Context, *gomock.Controller, *eventbusmock.MockEventBus)
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
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&userdbtypes.User{})).
					DoAndReturn(func(ctx context.Context, u usertypes.User) error {
						user, ok := u.(*userdbtypes.User)
						if !ok {
							return fmt.Errorf("unexpected type passed to CreateUser: %T", u)
						}
						if user.DiscordID != "user123" || user.Role != usertypes.UserRoleRattler {
							return fmt.Errorf("unexpected user data: %+v", user)
						}
						return nil
					}).
					Times(1)
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// Expect publish for CheckTagAvailabilityRequest on the leaderboard stream
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						if msg.Metadata.Get("subject") != userevents.CheckTagAvailabilityRequest {
							return fmt.Errorf("unexpected subject for CheckTagAvailabilityRequest: got %v, want %v", msg.Metadata.Get("subject"), userevents.CheckTagAvailabilityRequest)
						}
						return nil
					}).
					Times(1)

				// Expect subscribe for CheckTagAvailabilityResponse on the leaderboard stream
				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Eq(userevents.CheckTagAvailabilityResponse), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						responsePayload := userevents.CheckTagAvailabilityResponsePayload{
							IsAvailable: true, // Assuming the tag is available
						}
						payloadBytes, _ := json.Marshal(responsePayload)
						responseMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
						responseMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityResponse)

						go handler(ctx, responseMsg)
						return nil
					}).
					Times(1)

				// Expect publish for TagAssignedRequest on the leaderboard stream (after tag availability is confirmed)
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						if subject := msg.Metadata.Get("subject"); subject != userevents.TagAssignedRequest {
							return fmt.Errorf("unexpected subject: got %v, want %v", subject, userevents.TagAssignedRequest)
						}
						return nil
					}).
					Times(1)
			},
			want: &userevents.UserSignupResponsePayload{
				Success: true,
			},
			wantErr: false,
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
					Publish(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						// Ensure the subject is set correctly in the message metadata
						if msg.Metadata.Get("subject") != userevents.CheckTagAvailabilityRequest {
							return fmt.Errorf("unexpected subject: got %v, want %v", msg.Metadata.Get("subject"), userevents.CheckTagAvailabilityRequest)
						}
						return nil
					}).
					Times(1)

				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Eq(userevents.CheckTagAvailabilityResponse), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						if err == nil {
							responsePayload := userevents.CheckTagAvailabilityResponsePayload{
								IsAvailable: false, // Tag is NOT available
							}
							payloadBytes, _ := json.Marshal(responsePayload)
							responseMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
							// Set the subject in the metadata for the response message
							responseMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityResponse)

							// Directly call the handler function to simulate the response
							go func() {
								handler(ctx, responseMsg)
							}()
						}
						return nil
					}).AnyTimes()
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
					Publish(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Any()).
					Return(errors.New("tag check error")).
					Times(1)

				// Expect Subscribe call even though Publish fails
				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Eq(userevents.CheckTagAvailabilityResponse), gomock.Any()).
					Return(nil).
					AnyTimes()
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
				// No CreateUser expectation because tag assignment failed
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// Mock Publish for CheckTagAvailabilityRequest
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						sub := msg.Metadata.Get("subject") // Extract the subject for each call

						switch sub {
						case userevents.CheckTagAvailabilityRequest:
							return nil // Simulate success for CheckTagAvailabilityRequest
						case userevents.TagAssignedRequest:
							return errors.New("publish error") // Simulate failure for TagAssignedRequest
						default:
							return fmt.Errorf("unexpected subject: %v", sub)
						}
					}).
					Times(2) // One for CheckTagAvailabilityRequest, one for TagAssignedRequest

				// Mock Subscribe for CheckTagAvailabilityResponse
				mockEB.EXPECT().
					Subscribe(gomock.Any(), gomock.Eq(userstream.LeaderboardStreamName), gomock.Eq(userevents.CheckTagAvailabilityResponse), gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						responsePayload := userevents.CheckTagAvailabilityResponsePayload{
							IsAvailable: true, // Simulate tag is available
						}
						payloadBytes, _ := json.Marshal(responsePayload)
						responseMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
						responseMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityResponse)

						go handler(ctx, responseMsg) // Trigger the handler asynchronously
						return nil
					}).
					Times(1) // Only for CheckTagAvailabilityResponse
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
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil)) // Use os.Stderr for logging in tests

			ctx := context.Background()

			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, ctrl, mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, ctrl, mockEventBus)
			}
			if tt.mockCheckTag != nil {
				tt.mockCheckTag(ctrl, mockEventBus, true, nil) // Default to success
			}

			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
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
