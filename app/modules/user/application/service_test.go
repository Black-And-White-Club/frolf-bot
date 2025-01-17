package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	usertypemocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		userDB   *userdb.MockUserDB
		eventBus *eventbusmock.MockEventBus
		logger   *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want *UserServiceImpl
	}{
		{
			name: "Success",
			args: args{
				userDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			want: &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUserService(tt.args.userDB, tt.args.eventBus, tt.args.logger)
			if got.UserDB != tt.want.UserDB || got.eventBus != tt.want.eventBus || got.logger != tt.want.logger {
				t.Errorf("NewUserService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserServiceImpl_signupOrchestrator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.Default()

	s := &UserServiceImpl{
		UserDB:   mockUserDB,
		eventBus: mockEventBus,
		logger:   logger,
	}

	testCases := []struct {
		name      string
		req       events.UserSignupRequestPayload
		mockSetup func(respPublished chan struct{})
		wantErr   bool
	}{
		{
			name: "Success",
			req: events.UserSignupRequestPayload{
				DiscordID: "12345",
				TagNumber: 1,
			},
			mockSetup: func(respPublished chan struct{}) {
				// Channel to signal when subscription is ready
				subReady := make(chan struct{})

				// Mock Subscribe for CheckTagAvailabilityResponse
				mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Eq(events.CheckTagAvailabilityResponse), gomock.Any()).DoAndReturn(
					func(ctx context.Context, streamName, subject string, handler interface{}) error {
						// Call the handler function in a separate goroutine after signaling subReady
						go func() {
							close(subReady)
							<-respPublished // Wait for the signal to send the response
							msg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available": true}`))
							msg.Metadata.Set("subject", events.CheckTagAvailabilityResponse)
							h, ok := handler.(func(ctx context.Context, msg *message.Message) error)
							if !ok {
								panic("invalid handler function type")
							}
							if err := h(ctx, msg); err != nil {
								panic("failed to publish response: " + err.Error())
							}
						}()
						return nil
					},
				)

				// Mock Publish for CheckTagAvailabilityRequest
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						// Ensure subscription is ready before proceeding
						<-subReady
						return nil
					},
				).Times(1)

				// Mock Publish for TagAssignedRequest
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						// Signal that TagAssignedRequest is published, allowing the response to be sent
						close(respPublished)
						return nil
					},
				).Times(1)

				mockUserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)

				// Mock Publish for UserCreated
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "Error - Tag Unavailable",
			req: events.UserSignupRequestPayload{
				DiscordID: "12345",
				TagNumber: 1,
			},
			mockSetup: func(respPublished chan struct{}) {
				subReady := make(chan struct{})
				mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Eq(events.CheckTagAvailabilityResponse), gomock.Any()).DoAndReturn(
					func(ctx context.Context, streamName, subject string, handler interface{}) error {
						go func() {
							close(subReady)
							<-respPublished
							msg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available": false}`))
							msg.Metadata.Set("subject", events.CheckTagAvailabilityResponse)
							h, ok := handler.(func(ctx context.Context, msg *message.Message) error)
							if !ok {
								panic("invalid handler function type")
							}
							if err := h(ctx, msg); err != nil {
								panic("failed to publish response: " + err.Error())
							}
						}()
						return nil
					},
				)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						<-subReady
						close(respPublished)
						return nil
					},
				)
			},
			wantErr: true,
		},
		{
			name: "Error - CreateUser Fails",
			req: events.UserSignupRequestPayload{
				DiscordID: "12345",
				TagNumber: 1,
			},
			mockSetup: func(respPublished chan struct{}) {
				subReady := make(chan struct{})
				mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Eq(events.CheckTagAvailabilityResponse), gomock.Any()).DoAndReturn(
					func(ctx context.Context, streamName, subject string, handler interface{}) error {
						go func() {
							close(subReady)
							<-respPublished
							msg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available": true}`))
							msg.Metadata.Set("subject", events.CheckTagAvailabilityResponse)
							h, ok := handler.(func(ctx context.Context, msg *message.Message) error)
							if !ok {
								panic("invalid handler function type")
							}
							if err := h(ctx, msg); err != nil {
								panic("failed to publish response: " + err.Error())
							}
						}()
						return nil
					},
				)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						<-subReady
						return nil
					},
				)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						close(respPublished)
						return nil
					},
				)
				mockUserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to create user"))
			},
			wantErr: true,
		},
		{
			name: "Error - Publish TagAssignedRequest Fails",
			req: events.UserSignupRequestPayload{
				DiscordID: "12345",
				TagNumber: 1,
			},
			mockSetup: func(respPublished chan struct{}) {
				subReady := make(chan struct{})
				mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Eq(events.CheckTagAvailabilityResponse), gomock.Any()).DoAndReturn(
					func(ctx context.Context, streamName, subject string, handler interface{}) error {
						go func() {
							close(subReady)
							<-respPublished
							msg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available": true}`))
							msg.Metadata.Set("subject", events.CheckTagAvailabilityResponse)
							h, ok := handler.(func(ctx context.Context, msg *message.Message) error)
							if !ok {
								panic("invalid handler function type")
							}
							if err := h(ctx, msg); err != nil {
								panic("failed to publish response: " + err.Error())
							}
						}()
						return nil
					},
				)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						<-subReady
						return nil
					},
				)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.LeaderboardStreamName), gomock.Any()).DoAndReturn(
					func(ctx context.Context, topic string, msg *message.Message) error {
						close(respPublished)
						return fmt.Errorf("failed to publish event")
					},
				)
			},
			wantErr: true,
		},
		{
			name: "Error - Publish UserCreated Fails",
			req: events.UserSignupRequestPayload{
				DiscordID: "12345",
				TagNumber: 0,
			},
			mockSetup: func(respPublished chan struct{}) {
				mockUserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(fmt.Errorf("failed to publish event"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Channel to signal when response is published in checkTagAvailability
			respPublished := make(chan struct{})

			tc.mockSetup(respPublished)

			// Wrap the original checkTagAvailability method for the test
			originalCheckTagAvailability := s.checkTagAvailability
			s.checkTagAvailability = func(ctx context.Context, tagNumber int) (bool, error) {
				s.logger.Info("checkTagAvailability", slog.Int("tag_number", tagNumber))

				// Prepare the request payload
				payload, err := json.Marshal(events.CheckTagAvailabilityRequestPayload{
					TagNumber: tagNumber,
				})
				if err != nil {
					return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequest payload: %w", err)
				}

				// Create the request message
				requestMsg := message.NewMessage(watermill.NewUUID(), payload)
				requestMsg.Metadata.Set("correlation_id", watermill.NewUUID())
				requestMsg.Metadata.Set("subject", events.CheckTagAvailabilityRequest)

				// Publish the request
				if err := s.eventBus.Publish(ctx, events.LeaderboardStreamName, requestMsg); err != nil {
					return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
				}

				// Use a context with timeout for the subscription
				subscribeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				// Create a channel to receive the response message
				responseChan := make(chan *message.Message)

				// Subscribe to the response
				err = mockEventBus.Subscribe(subscribeCtx, events.LeaderboardStreamName, events.CheckTagAvailabilityResponse, func(ctx context.Context, msg *message.Message) error {
					responseChan <- msg
					return nil
				})
				if err != nil {
					return false, fmt.Errorf("failed to subscribe to CheckTagAvailabilityResponse: %w", err)
				}

				// Wait for the response or timeout
				select {
				case responseMsg := <-responseChan:
					var response events.CheckTagAvailabilityResponsePayload
					if err := json.Unmarshal(responseMsg.Payload, &response); err != nil {
						return false, fmt.Errorf("failed to unmarshal response payload: %w", err)
					}
					return response.IsAvailable, nil
				case <-subscribeCtx.Done():
					return false, fmt.Errorf("timeout waiting for tag availability response")
				}
			}

			err := s.signupOrchestrator(context.Background(), tc.req)
			if (err != nil) != tc.wantErr {
				t.Errorf("signupOrchestrator() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Restore the original checkTagAvailability method after the test
			s.checkTagAvailability = originalCheckTagAvailability
		})
	}
}

func TestUserServiceImpl_OnUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		req           events.UserRoleUpdateRequestPayload
		mockUserDB    func(context.Context, *userdb.MockUserDB)
		mockEventBus  func(context.Context, *eventbusmock.MockEventBus)
		want          *events.UserRoleUpdateResponsePayload
		wantErr       error
		publishCalled bool
	}{
		{
			name: "Success",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)
				mockUser := usertypemocks.NewMockUser(ctrl)
				mockDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(mockUser, nil).AnyTimes()
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()). // Use UserRoleUpdateResponseStreamName
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						if streamName != events.UserStreamName {
							t.Errorf("Expected stream name: %s, got: %s", events.UserStreamName, streamName)
						}
						subject := msg.Metadata.Get("subject")
						if subject != events.UserRoleUpdated {
							t.Errorf("Expected subject: %s, got: %s", events.UserRoleUpdated, subject)
						}
						return nil
					}).
					Times(1)
			},
			want: &events.UserRoleUpdateResponsePayload{
				Success: true,
			},
			wantErr:       nil,
			publishCalled: true,
		},
		{
			name: "Invalid Role",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   "InvalidRole", // Invalid role
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				// No expectations on UserDB as the role is invalid
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on EventBus as the role is invalid
			},
			want:          nil,
			wantErr:       fmt.Errorf("invalid user role: InvalidRole"),
			publishCalled: false,
		},
		{
			name: "Empty Discord ID",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "", // Empty Discord ID
				NewRole:   usertypes.UserRoleEditor,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				// No expectations on UserDB as Discord ID is empty
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on EventBus as Discord ID is empty
			},
			want:          nil,
			wantErr:       fmt.Errorf("missing DiscordID in request"),
			publishCalled: false,
		},
		{
			name: "UpdateUserRole Error",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleEditor,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleEditor).
					Return(fmt.Errorf("update error"))
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on EventBus as UpdateUserRole fails
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to update user role: %w", fmt.Errorf("update error")),
			publishCalled: false,
		},
		{
			name: "GetUserByDiscordID Error",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)
				mockDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).
					Return(nil, fmt.Errorf("get user error"))
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on EventBus as GetUserByDiscordID fails
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to get user: %w", fmt.Errorf("get user error")),
			publishCalled: false,
		},
		{
			name: "User Not Found",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)
				mockDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).
					Return(nil, nil) // User not found
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on EventBus as user is not found
			},
			want:          nil,
			wantErr:       fmt.Errorf("user not found: 12345"),
			publishCalled: false,
		},
		{
			name: "Publish Event Error",
			req: events.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)
				mockUser := usertypemocks.NewMockUser(ctrl)
				mockDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(mockUser, nil).AnyTimes()
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()). // Use events.UserStreamName
					Return(fmt.Errorf("publish error"))                                    // Simulate publish error
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to publish UserRoleUpdated event: eventBus.Publish UserRoleUpdated: publish error"),
			publishCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserDB := userdb.NewMockUserDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)

			ctx := context.Background()

			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, mockEventBus)
			}

			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			got, err := service.OnUserRoleUpdateRequest(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("OnUserRoleUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("OnUserRoleUpdateRequest() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnUserRoleUpdateRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockUserDB := userdb.NewMockUserDB(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	s := &UserServiceImpl{
		eventBus: mockEventBus,
		UserDB:   mockUserDB,
		logger:   logger,
	}

	tests := []struct {
		name        string
		discordID   usertypes.DiscordID
		mockUserDB  func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		want        usertypes.UserRoleEnum
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "Success",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleAdmin, nil)
			},
			want:    usertypes.UserRoleAdmin,
			wantErr: false,
		},
		{
			name:      "GetUser Role Error",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleUnknown, errors.New("database error"))
			},
			want:        usertypes.UserRoleUnknown,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user role: %w", errors.New("database error")),
		},
		{
			name:      "User Not Found",
			discordID: "nonexistent",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("nonexistent")).Return(usertypes.UserRoleUnknown, errors.New("user not found"))
			},
			want:        usertypes.UserRoleUnknown,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user role: %w", errors.New("user not found")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}

			got, err := s.GetUserRole(ctx, tt.discordID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser Role() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUser Role() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserServiceImpl_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	s := &UserServiceImpl{
		eventBus: mockEventBus,
		UserDB:   mockUserDB,
		logger:   logger,
	}

	tests := []struct {
		name        string
		discordID   usertypes.DiscordID
		mockUserDB  func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		want        usertypes.User
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "Success",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				expectedUser := usertypemocks.NewMockUser(ctrl)
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(expectedUser, nil)
			},
			want:    usertypemocks.NewMockUser(ctrl),
			wantErr: false,
		},
		{
			name:      "User  Not Found",
			discordID: "nonexistent",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("nonexistent")).Return(nil, errors.New("user not found"))
			},
			want:        nil,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("user not found")),
		},
		{
			name:      "Database Error",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(nil, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("database error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}

			got, err := s.GetUser(ctx, tt.discordID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUser () = %v, want %v", got, tt.want)
			}
		})
	}
}
