package userhandlers

import (
	"log/slog"
	"os"
	"testing"

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
			if got == nil {
				t.Errorf("NewUserHandlers() returned nil, expected non-nil")
			}
			if got != nil && tt.want != nil {
				if got.userService != tt.want.userService {
					t.Errorf("NewUserHandlers() userService = %v, want %v", got.userService, tt.want.userService)
				}
				if got.logger != tt.want.logger {
					t.Errorf("NewUserHandlers() logger = %v, want %v", got.logger, tt.want.logger)
				}
			}
		})
	}
}
