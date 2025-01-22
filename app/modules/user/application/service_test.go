package userservice

import (
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name string
		args struct {
			db       userdb.MockUserDB
			eventBus eventbusmock.MockEventBus
			logger   *slog.Logger
		}
		want *UserServiceImpl
	}{
		{
			name: "Success",
			args: struct {
				db       userdb.MockUserDB
				eventBus eventbusmock.MockEventBus
				logger   *slog.Logger
			}{
				db:       *mockUserDB,
				eventBus: *mockEventBus,
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
			got, err := NewUserService(&tt.args.db, &tt.args.eventBus, tt.args.logger)
			if err != nil {
				t.Errorf("NewUserService() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserService() = %v, want %v", got, tt.want)
			}
		})
	}
}
