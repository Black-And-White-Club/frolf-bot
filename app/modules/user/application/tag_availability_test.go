package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CheckTagAvailability(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testTagNumber := 123
	testCorrelationID := watermill.NewUUID()
	testCtx := context.Background()

	type fields struct {
		UserDB   *userdb.MockUserDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx           context.Context
		msg           *message.Message
		tagNumber     int
		correlationID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful CheckTagAvailability",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				msg:           message.NewMessage(testCorrelationID, nil),
				tagNumber:     testTagNumber,
				correlationID: testCorrelationID,
			},
			wantErr: false,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, args.correlationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.LeaderboardTagAvailabilityCheckRequest, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic string, msg *message.Message) error {
						if topic != userevents.LeaderboardTagAvailabilityCheckRequest {
							t.Errorf("Expected topic %s, got %s", userevents.LeaderboardTagAvailabilityCheckRequest, topic)
						}

						var payload userevents.CheckTagAvailabilityRequestPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("failed to unmarshal message payload: %v", err)
						}

						if payload.TagNumber != testTagNumber {
							t.Errorf("Expected tag number %d, got %d", testTagNumber, payload.TagNumber)
						}

						if msg.UUID == args.correlationID {
							t.Errorf("Expected new message UUID, but it matched correlation ID: %s", msg.UUID)
						}

						if mid := msg.Metadata.Get(middleware.CorrelationIDMetadataKey); mid != args.correlationID {
							t.Errorf("Expected correlation ID %s, got %s", args.correlationID, mid)
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Failed to Publish Event",
			fields: fields{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx:           testCtx,
				msg:           message.NewMessage(testCorrelationID, nil),
				tagNumber:     testTagNumber,
				correlationID: testCorrelationID,
			},
			wantErr: true,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, args.correlationID)
				f.eventBus.EXPECT().
					Publish(args.ctx, userevents.LeaderboardTagAvailabilityCheckRequest, gomock.Any()).
					Return(errors.New("publish error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UserServiceImpl{
				UserDB:   tt.fields.UserDB,
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}

			// Call setup function to configure mocks before each test case
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			if err := s.CheckTagAvailability(tt.args.ctx, tt.args.msg, tt.args.tagNumber); (err != nil) != tt.wantErr {
				t.Errorf("UserServiceImpl.CheckTagAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
