package userhandlers

import (
	"log/slog"
	"os"
	"testing"

	"reflect"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl) // Mock the userservice.Service interface
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	type args struct {
		userService *mocks.MockService
		logger      *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want *UserHandlers
	}{
		{
			name: "Successfully creates new UserHandlers",
			args: args{
				userService: mockUserService,
				logger:      logger,
			},
			want: &UserHandlers{
				userService: mockUserService,
				logger:      logger,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUserHandlers(tt.args.userService, tt.args.logger)

			// Check if got is nil
			if got == nil {
				t.Errorf("NewUserHandlers() returned nil, expected non-nil")
			}

			// Type assert got to *UserHandlers to access userService
			if userHandlers, ok := got.(*UserHandlers); ok {
				// Now that we have the concrete type, we can access userService
				if !reflect.DeepEqual(userHandlers.userService, tt.want.userService) {
					t.Errorf("NewUserHandlers() userService = %v, want %v", userHandlers.userService, tt.want.userService)
				}
				// Logger comparison can be omitted or handled similarly as needed
			} else {
				t.Errorf("Expected *UserHandlers, got %T", got)
			}
		})
	}
}
