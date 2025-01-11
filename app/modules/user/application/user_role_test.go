package userservice

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	usertypemocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		req           userevents.UserRoleUpdateRequestPayload
		mockUserDB    func(context.Context, *userdb.MockUserDB)
		mockEventBus  func(context.Context, *eventbusmock.MockEventBus)
		want          *userevents.UserRoleUpdateResponsePayload
		wantErr       error
		publishCalled bool
	}{
		{
			name: "Success",
			req: userevents.UserRoleUpdateRequestPayload{
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
					Publish(gomock.Any(), gomock.Eq(userstream.UserRoleUpdateResponseStreamName), gomock.Any()). // Use UserRoleUpdateResponseStreamName
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						if streamName != userstream.UserRoleUpdateResponseStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.UserRoleUpdateResponseStreamName, streamName)
						}
						subject := msg.Metadata.Get("subject")
						if subject != userevents.UserRoleUpdated {
							t.Errorf("Expected subject: %s, got: %s", userevents.UserRoleUpdated, subject)
						}
						return nil
					}).
					Times(1)
			},
			want: &userevents.UserRoleUpdateResponsePayload{
				Success: true,
			},
			wantErr:       nil,
			publishCalled: true,
		},
		{
			name: "Invalid Role",
			req: userevents.UserRoleUpdateRequestPayload{
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
			req: userevents.UserRoleUpdateRequestPayload{
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
			req: userevents.UserRoleUpdateRequestPayload{
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
			req: userevents.UserRoleUpdateRequestPayload{
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
			req: userevents.UserRoleUpdateRequestPayload{
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
			req: userevents.UserRoleUpdateRequestPayload{
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
					Publish(gomock.Any(), gomock.Eq(userstream.UserRoleUpdateResponseStreamName), gomock.Any()). // Use UserRoleUpdateResponseStreamName
					Return(fmt.Errorf("publish error"))                                                          // Simulate publish error
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
