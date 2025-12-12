package userhandlers

import (
	"encoding/json"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUpdateUDiscIdentityRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testUsername := "testuser"
	testName := "Test User"

	createUpdateRequestMessage := func() *message.Message {
		payload := &userevents.UpdateUDiscIdentityRequestPayload{
			GuildID:  testGuildID,
			UserID:   testUserID,
			Username: &testUsername,
			Name:     &testName,
		}
		payloadBytes, _ := json.Marshal(payload)
		return message.NewMessage("test-id", payloadBytes)
	}

	mockUserService := usermocks.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		msg       *message.Message
		mockSetup func()
		want      []*message.Message
		wantErr   bool
	}{
		{
			name: "Successful update",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscIdentityUpdatedPayload{
							UserID: testUserID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscIdentityUpdated).
					Return(message.NewMessage("out-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("out-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Update failed (business logic failure)",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{
						Failure: &userevents.UDiscIdentityUpdateFailedPayload{
							GuildID: testGuildID,
							UserID:  testUserID,
							Reason:  "some error",
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscIdentityUpdateFailed).
					Return(message.NewMessage("out-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("out-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Service error",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{}, errors.New("service error"))
			},
			wantErr: true,
		},
		{
			name: "Unexpected result (nil success and failure)",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{}, nil)
			},
			wantErr: true,
		},
		{
			name: "Failure to create failure message",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{
						Failure: &userevents.UDiscIdentityUpdateFailedPayload{
							GuildID: testGuildID,
							UserID:  testUserID,
							Reason:  "some error",
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscIdentityUpdateFailed).
					Return(nil, errors.New("create message error"))
			},
			wantErr: true,
		},
		{
			name: "Failure to create success message",
			msg:  createUpdateRequestMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UpdateUDiscIdentityRequestPayload) = userevents.UpdateUDiscIdentityRequestPayload{
							GuildID:  testGuildID,
							UserID:   testUserID,
							Username: &testUsername,
							Name:     &testName,
						}
						return nil
					},
				)
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscIdentityUpdatedPayload{
							UserID: testUserID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscIdentityUpdated).
					Return(nil, errors.New("create message error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := NewUserHandlers(mockUserService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleUpdateUDiscIdentityRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUpdateUDiscIdentityRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("HandleUpdateUDiscIdentityRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
