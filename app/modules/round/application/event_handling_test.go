package roundservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundService_getTagNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	s := &RoundService{
		RoundDB:  mockRoundDB,
		eventBus: mockEventBus,
		logger:   logger,
	}

	discordID := "12345"
	tagNumber := 100

	testCases := []struct {
		name        string
		setupMocks  func()
		timeout     time.Duration
		want        *int
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Success",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.GetTagNumberRequest, gomock.Any()).
					Return(nil)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.LeaderboardStreamName, events.GetTagNumberResponse, gomock.Any()).
					DoAndReturn(func(ctx context.Context, stream, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						// Simulate receiving a response message
						go func() {
							time.Sleep(50 * time.Millisecond) // Simulate delay
							resp := events.GetTagNumberResponsePayload{
								DiscordID: discordID,
								TagNumber: tagNumber,
							}
							data, _ := json.Marshal(resp)
							msg := message.NewMessage(watermill.NewUUID(), data)
							_ = handler(ctx, msg)
						}()
						return nil
					})
			},
			timeout:     1 * time.Second,
			want:        &tagNumber,
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "Timeout",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.GetTagNumberRequest, gomock.Any()).
					Return(nil)

				mockEventBus.EXPECT().
					Subscribe(gomock.Any(), events.LeaderboardStreamName, events.GetTagNumberResponse, gomock.Any()).
					Return(nil) // No handler to simulate timeout
			},
			timeout:     1 * time.Millisecond, // Very short timeout to force immediate timeout
			want:        nil,
			wantErr:     true,
			expectedErr: "timeout waiting for response", // Ensure this matches the actual error message
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupMocks != nil {
				tc.setupMocks()
			}

			got, err := s.getTagNumber(context.Background(), discordID, tc.timeout)

			if (err != nil) != tc.wantErr {
				t.Errorf("RoundService.getTagNumber() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.expectedErr != "" && err.Error() != tc.expectedErr {
				t.Errorf("Expected error: %q, Got: %q", tc.expectedErr, err.Error())
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("RoundService.getTagNumber() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRoundService_scheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	s := &RoundService{
		RoundDB:  mockRoundDB,
		eventBus: mockEventBus,
		logger:   logger,
	}

	ctx := context.Background()
	roundID := "test-round-id"
	startTime := time.Now().Add(2 * time.Hour)

	testCases := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		expectedErr error
	}{
		{
			name: "Success",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundReminder, gomock.Any()).
					Times(2).
					Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundStarted, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "OneHourReminderError",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundReminder, gomock.Any()).
					Return(errors.New("publish error"))
			},
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to schedule one-hour reminder: %w", errors.New("publish error")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupMocks != nil {
				tc.setupMocks()
			}

			err := s.scheduleRoundEvents(ctx, roundID, startTime)

			if (err != nil) != tc.wantErr {
				t.Errorf("RoundService.scheduleRoundEvents() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestRoundService_publishEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	s := &RoundService{
		RoundDB:  mockRoundDB,
		eventBus: mockEventBus,
		logger:   logger,
	}

	ctx := context.Background()
	subject := "test-subject"
	event := &events.RoundCreatedPayload{RoundID: "test-round-id"}

	testCases := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		expectedErr error
	}{
		{
			name: "Success",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), subject, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "PublishError",
			setupMocks: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), subject, gomock.Any()).
					Return(errors.New("publish error"))
			},
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to publish event: %w", errors.New("publish error")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupMocks != nil {
				tc.setupMocks()
			}

			err := s.publishEvent(ctx, subject, event)

			if (err != nil) != tc.wantErr {
				t.Errorf("RoundService.publishEvent() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tc.expectedErr, err)
			}
		})
	}
}
