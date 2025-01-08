package userservice

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	adaptermock "github.com/Black-And-White-Club/tcr-bot/app/adapters/mocks"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	usertypemocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)
	mockEventAdapter := new(adaptermock.MockEventAdapterInterface)

	s := &UserServiceImpl{
		UserDB:       mockUserDB,
		eventBus:     mockEventBus,
		logger:       mockLogger,
		eventAdapter: mockEventAdapter,
	}

	tests := []struct {
		name          string
		req           userevents.UserRoleUpdateRequestPayload
		mockUserDB    func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		mockEventBus  func(ctx context.Context, mockEventBus *eventbusmock.MockEventBus)
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
				mockEB.EXPECT().Publish(ctx, gomock.Eq(userevents.UserRoleUpdated), gomock.Any()).Return(nil).Times(1)
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
				NewRole:   usertypes.UserRoleEnum("InvalidRole"),
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				// No expectations on mockUserDB as it should not be called
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on mockEventBus as it should not be called
			},
			want:          nil,
			wantErr:       fmt.Errorf("invalid user role: %s", usertypes.UserRoleEnum("InvalidRole")),
			publishCalled: false,
		},
		{
			name: "Missing DiscordID",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				// No expectations on mockUserDB as it should not be called
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on mockEventBus as it should not be called
			},
			want:          nil,
			wantErr:       fmt.Errorf("missing DiscordID in request"),
			publishCalled: false,
		},
		{
			name: "Error Updating Role",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).
					Return(errors.New("database error"))

				// Add expectation for GetUserByDiscordID
				mockUser := usertypemocks.NewMockUser(ctrl)
				mockDB.EXPECT().GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("12345")).Return(mockUser, nil).AnyTimes()
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				// No expectations on mockEventBus as it should not be called
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to update user role: %w", errors.New("database error")),
			publishCalled: false,
		},
		{
			name: "Error Publishing Event",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(ctx, usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)

				// Add expectation for GetUserByDiscordID
				mockUser := usertypemocks.NewMockUser(ctrl)
				mockDB.EXPECT().GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("12345")).Return(mockUser, nil).AnyTimes()
			},
			mockEventBus: func(ctx context.Context, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.UserRoleUpdated), gomock.Any()).
					Return(fmt.Errorf("simulated publish error")).
					Times(1)
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to publish UserRoleUpdated event: %w", fmt.Errorf("eventBus.Publish UserRoleUpdated: %w", fmt.Errorf("simulated publish error"))), // Include context
			publishCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, mockEventBus)
			}

			got, err := s.OnUserRoleUpdateRequest(ctx, tt.req)

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
