package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	user_mocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_checkTagAvailability(t *testing.T) {
	tests := []struct {
		name        string
		tagNumber   int
		setupMocks  func(mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber)
		want        bool
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "Success - Available Tag",
			tagNumber: 123,
			setupMocks: func(mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				mockPublisher.EXPECT().Publish(
					userevents.CheckTagAvailabilityRequestSubject,
					gomock.AssignableToTypeOf(&message.Message{}),
				).Do(func(_ string, msg *message.Message) {
					replySubject := msg.Metadata.Get("Reply-To")
					replyMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available":true}`))
					msgChan := make(chan *message.Message, 1)
					msgChan <- replyMsg
					close(msgChan)
					mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil)
				}).Return(nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "Timeout waiting for response",
			tagNumber: 123,
			setupMocks: func(mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				expectedPayload := map[string]int{"tag_number": 123}

				mockPublisher.EXPECT().Publish(
					userevents.CheckTagAvailabilityRequestSubject,
					gomock.AssignableToTypeOf(&message.Message{}),
				).Do(func(_ string, msg *message.Message) {
					// Validate payload
					payload := map[string]int{}
					if err := json.Unmarshal(msg.Payload, &payload); err != nil {
						t.Fatalf("Failed to unmarshal payload: %v", err)
					}
					if !reflect.DeepEqual(payload, expectedPayload) {
						t.Errorf("Expected payload: %v, got: %v", expectedPayload, payload)
					}
					msgChan := make(chan *message.Message)
					mockSubscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(msgChan, nil)

				}).Return(nil)
			},
			want:        false,
			wantErr:     true,
			expectedErr: errTimeout,
		},
		{
			name:      "Subscribe error",
			tagNumber: 123,
			setupMocks: func(mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				mockPublisher.EXPECT().Publish(
					userevents.CheckTagAvailabilityRequestSubject,
					gomock.AssignableToTypeOf(&message.Message{}),
				).Return(nil)

				mockSubscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(nil, errSubscribe)
			},
			want:        false,
			wantErr:     true,
			expectedErr: errSubscribe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPublisher := testutils.NewMockPublisher(ctrl)
			mockSubscriber := testutils.NewMockSubscriber(ctrl)

			tt.setupMocks(mockPublisher, mockSubscriber)

			s := &UserServiceImpl{
				Publisher:  mockPublisher,
				Subscriber: mockSubscriber,
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			got, err := s.checkTagAvailability(ctx, tt.tagNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkTagAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error: %v, got: %v", tt.expectedErr, err)
			}
			if got != tt.want {
				t.Errorf("checkTagAvailability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserServiceImpl_OnUserSignupRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         userevents.UserSignupRequest
		setupMocks  func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber)
		want        *userevents.UserSignupResponse
		wantErr     bool
		expectedErr error
	}{
		{
			name: "Success - Tag Available",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				TagNumber: 123,
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// Expect a call to checkTagAvailability and return true (tag is available)
				mockPublisher.EXPECT().Publish(userevents.CheckTagAvailabilityRequestSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(_ string, msg *message.Message) {
						replySubject := msg.Metadata.Get("Reply-To")
						replyMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available":true}`))
						msgChan := make(chan *message.Message, 1)
						msgChan <- replyMsg
						close(msgChan)
						mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil)
					}).Return(nil)

				// Expect a call to CreateUser with the correct user data
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(nil)

				// Expect a call to publishTagAssigned with the correct data
				mockPublisher.EXPECT().Publish(userevents.TagAssignedSubject, gomock.AssignableToTypeOf(&message.Message{})).Return(nil)
			},
			want: &userevents.UserSignupResponse{
				Success: true,
			},
			wantErr: false,
		},
		{
			name: "Tag Not Available",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				TagNumber: 123,
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// Expect a call to checkTagAvailability and return false (tag is not available)
				mockPublisher.EXPECT().Publish(userevents.CheckTagAvailabilityRequestSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(_ string, msg *message.Message) {
						replySubject := msg.Metadata.Get("Reply-To")
						replyMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available":false}`))
						msgChan := make(chan *message.Message, 1)
						msgChan <- replyMsg
						close(msgChan)
						mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil)
					}).Return(nil)

				// Expect a call to CreateUser since we're proceeding with signup despite the tag being unavailable
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(nil)
			},
			want: &userevents.UserSignupResponse{
				Success: true, // Expect signup to succeed even if the tag is unavailable
			},
			wantErr: false,
		},
		{
			name: "Error Checking Tag Availability",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				TagNumber: 123,
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// Expect a call to checkTagAvailability and return an error
				mockPublisher.EXPECT().Publish(userevents.CheckTagAvailabilityRequestSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(_ string, msg *message.Message) {
						replySubject := msg.Metadata.Get("Reply-To")
						msgChan := make(chan *message.Message)
						mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil) // Simulate an error by not sending a reply message
					}).Return(nil)

				// Expect a call to CreateUser with the correct user data (since we're proceeding with signup despite the error)
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(nil)

				// No expectation for publishTagAssigned (since the tag availability check failed)

			},
			want: &userevents.UserSignupResponse{ // Expect a successful response
				Success: true,
			},
			wantErr: false,
		},
		{
			name: "Error Creating User - General Error with Tag",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				TagNumber: 123, // Include a tag number
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// Expect checkTagAvailability to be called and return true (tag is available)
				mockPublisher.EXPECT().Publish(userevents.CheckTagAvailabilityRequestSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(_ string, msg *message.Message) {
						replySubject := msg.Metadata.Get("Reply-To")
						replyMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available":true}`))
						msgChan := make(chan *message.Message, 1)
						msgChan <- replyMsg
						close(msgChan)
						mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil)
					}).Return(nil)

				// Expect CreateUser to return an error
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(errors.New("database error"))

				// No expectations for publishTagAssigned (as CreateUser should fail)
			},
			want:    nil, // Expect an error response
			wantErr: true,
		},
		{
			name: "Error Creating User - General Error",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				// No tag number
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// No expectations for tag availability check

				// Expect CreateUser to return an error
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(errors.New("database error"))

				// No expectations for publishTagAssigned
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Error Publishing TagAssigned Event",
			req: userevents.UserSignupRequest{
				DiscordID: "12345",
				TagNumber: 123,
			},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) {
				// Expect a call to checkTagAvailability and return true (tag is available)
				mockPublisher.EXPECT().Publish(userevents.CheckTagAvailabilityRequestSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(_ string, msg *message.Message) {
						replySubject := msg.Metadata.Get("Reply-To")
						replyMsg := message.NewMessage(watermill.NewUUID(), []byte(`{"is_available":true}`))
						msgChan := make(chan *message.Message, 1)
						msgChan <- replyMsg
						close(msgChan)
						mockSubscriber.EXPECT().Subscribe(gomock.Any(), replySubject).Return(msgChan, nil)
					}).Return(nil)

				// Expect a call to CreateUser with the correct user data
				mockDB.EXPECT().CreateUser(gomock.Any(), &userdb.User{DiscordID: "12345", Role: userdb.UserRoleRattler}).Return(nil)

				// Expect a call to publishTagAssigned and return an error
				mockPublisher.EXPECT().Publish(userevents.TagAssignedSubject, gomock.AssignableToTypeOf(&message.Message{})).Return(errors.New("publishing error"))
			},
			want:    nil,
			wantErr: true,
			// You might want to check for a specific publishing error here
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserDB := user_mocks.NewMockUserDB(ctrl)
			mockPublisher := testutils.NewMockPublisher(ctrl)
			mockSubscriber := testutils.NewMockSubscriber(ctrl)

			tt.setupMocks(mockUserDB, mockPublisher, mockSubscriber) // Pass the mocks to setupMocks

			s := &UserServiceImpl{
				UserDB:     mockUserDB,
				Publisher:  mockPublisher,
				Subscriber: mockSubscriber,
				logger:     watermill.NopLogger{},
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			got, err := s.OnUserSignupRequest(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error: %v, got: %v", tt.expectedErr, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnUserSignupRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserServiceImpl_publishTagAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := testutils.NewMockPublisher(ctrl)

	s := &UserServiceImpl{
		Publisher: mockPublisher,
		logger:    watermill.NopLogger{},
	}

	tests := []struct {
		name       string
		discordID  string
		tagNumber  int
		setupMocks func()
		wantErr    bool
	}{
		{
			name:      "Success",
			discordID: "12345",
			tagNumber: 123,
			setupMocks: func() {
				// Expect a call to Publish with the correct topic and message payload
				expectedPayload := userevents.TagAssigned{DiscordID: "12345", TagNumber: 123}
				mockPublisher.EXPECT().Publish(userevents.TagAssignedSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Do(func(topic string, msg *message.Message) {
						// Validate the message payload
						var evt userevents.TagAssigned
						if err := json.Unmarshal(msg.Payload, &evt); err != nil {
							t.Fatalf("Failed to unmarshal payload: %v", err)
						}
						if !reflect.DeepEqual(evt, expectedPayload) {
							t.Errorf("Expected payload: %v, got: %v", expectedPayload, evt)
						}
					}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "Publish Error",
			discordID: "12345",
			tagNumber: 123,
			setupMocks: func() {
				// Expect a call to Publish and return an error
				mockPublisher.EXPECT().Publish(userevents.TagAssignedSubject, gomock.AssignableToTypeOf(&message.Message{})).
					Return(errors.New("publishing error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			ctx := context.Background()
			err := s.publishTagAssigned(ctx, tt.discordID, tt.tagNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("publishTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
