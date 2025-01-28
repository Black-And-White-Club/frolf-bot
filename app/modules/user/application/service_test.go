package userservice

import (
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name string
		args struct {
			db       *userdb.MockUserDB          // Use mock type here
			eventBus *eventbusmocks.MockEventBus // Use mock type here
			logger   *slog.Logger
		}
		want    *UserServiceImpl
		wantErr bool // Add a field to indicate if an error is expected
	}{
		{
			name: "Success",
			args: struct {
				db       *userdb.MockUserDB
				eventBus *eventbusmocks.MockEventBus
				logger   *slog.Logger
			}{
				db:       mockUserDB,   // Use the mock here
				eventBus: mockEventBus, // Use the mock here
				logger:   logger,
			},
			want: &UserServiceImpl{
				UserDB:    mockUserDB,
				eventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(), // Initialize eventUtil
			},
			wantErr: false, // Expect no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUserService(tt.args.db, tt.args.eventBus, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUserService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserService() = %v, want %v", got, tt.want)
			}
		})
	}
}
