package userservice

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_createUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := mocks.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.Default()

	type fields struct {
		UserDB   *mocks.MockUserDB
		eventBus *eventbusmock.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx       context.Context
		discordID usertypes.DiscordID
		role      usertypes.UserRoleEnum
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:       context.Background(),
				discordID: usertypes.DiscordID("12345"),
				role:      usertypes.UserRoleRattler,
			},
			wantErr: false,
		},
		{
			name: "Error - CreateUser Fails",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:       context.Background(),
				discordID: usertypes.DiscordID("12345"),
				role:      usertypes.UserRoleRattler,
			},
			wantErr: true,
		},
		{
			name: "Error - Publish UserCreated Fails",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:       context.Background(),
				discordID: usertypes.DiscordID("12345"),
				role:      usertypes.UserRoleRattler,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				UserDB:   tt.fields.UserDB,
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}

			switch tt.name {
			case "Success":
				tt.fields.UserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
				tt.fields.eventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(nil)
			case "Error - CreateUser Fails":
				tt.fields.UserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
			case "Error - Publish UserCreated Fails":
				tt.fields.UserDB.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil)
				tt.fields.eventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(errors.New("event bus error"))
			}

			if err := s.createUser(tt.args.ctx, tt.args.discordID, tt.args.role); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.createUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserServiceImpl_publishUserCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.Default()

	type fields struct {
		UserDB   *mocks.MockUserDB
		eventBus *eventbusmock.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx     context.Context
		payload events.UserCreatedPayload
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx: context.Background(),
				payload: events.UserCreatedPayload{
					DiscordID: "12345",
					Role:      usertypes.UserRoleRattler,
				},
			},
			wantErr: false,
		},
		{
			name: "Error - Publish Fails",
			fields: fields{
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx: context.Background(),
				payload: events.UserCreatedPayload{
					DiscordID: "12345",
					Role:      usertypes.UserRoleRattler,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}

			switch tt.name {
			case "Success":
				tt.fields.eventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(nil)
			case "Error - Publish Fails":
				tt.fields.eventBus.EXPECT().Publish(gomock.Any(), gomock.Eq(events.UserStreamName), gomock.Any()).Return(errors.New("event bus error"))
			}

			if err := s.publishUserCreated(tt.args.ctx, tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishUserCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
