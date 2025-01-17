package userservice

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_publishUserRoleUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		payload      events.UserRoleUpdatedPayload
		mockEventBus func(mockEventBus *eventbusmock.MockEventBus)
		wantErr      bool
	}{
		{
			name: "Happy Path",
			payload: events.UserRoleUpdatedPayload{
				DiscordID: "1234567890",
				NewRole:   usertypes.UserRoleAdmin.String(),
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.UserStreamName, gomock.Any()). // Expect UserRoleUpdateResponseStreamName
					DoAndReturn(func(_ context.Context, streamName string, msg *message.Message) error {
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
			wantErr: false,
		},
		{
			name: "Error Publishing Event",
			payload: events.UserRoleUpdatedPayload{
				DiscordID: "1234567890",
				NewRole:   usertypes.UserRoleAdmin.String(),
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.UserStreamName, gomock.Any()). // Expect UserRoleUpdateResponseStreamName
					Return(fmt.Errorf("failed to publish"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				eventBus: mockEventBus,
				logger:   logger,
			}

			ctx := context.Background()
			err := s.publishUserRoleUpdated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.publishUserRoleUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
