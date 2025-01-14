package leaderboardhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	leaderboardmocks "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
	}{
		{
			name:               "Create New Leaderboard Handlers",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLeaderboardHandlers(tt.leaderboardService, tt.eventBus, tt.logger)

			if got == nil {
				t.Fatal("NewLeaderboardHandlers() returned nil") // Use t.Fatal to stop execution
			}

			if got.LeaderboardService != tt.leaderboardService {
				t.Error("LeaderboardService not set correctly")
			}
			if got.EventBus != tt.eventBus {
				t.Error("EventBus not set correctly")
			}
			if got.logger != tt.logger {
				t.Error("Logger not set correctly")
			}
		})
	}
}

func TestLeaderboardHandlers_HandleLeaderboardUpdate(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
	}{
		{
			name:               "Successful Update",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"scores": [{"discord_id": "123", "tag_number": "1", "score": -2}]}`)),
			wantErr:            false,
		},
		{
			name:               "Unmarshal Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr:            true,
		},
		{
			name:               "Update Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"scores": [{"discord_id": "123", "tag_number": "1", "score": -2}]}`)),
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Successful Update" {
				tt.leaderboardService.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any()).Return(nil)
			} else if tt.name == "Update Error" {
				tt.leaderboardService.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any()).Return(errors.New("update error"))
			}

			if err := h.HandleLeaderboardUpdate(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleLeaderboardUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagAssigned(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
	}{
		{
			name:               "Successful Assign",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id": "123", "tag_number": 123}`)),
			wantErr:            false,
		},
		// ... Add more test cases for HandleTagAssigned
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			if tt.name == "Successful Assign" {
				tt.leaderboardService.EXPECT().AssignTag(gomock.Any(), gomock.Any()).Return(nil)
			}

			if err := h.HandleTagAssigned(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByDiscordIDRequest(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
		wantTagNumber      int    // Expected tag number
		discordID          string // Discord ID to request
	}{
		{
			name:               "Successful Get",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id": "123"}`)),
			wantErr:            false,
			wantTagNumber:      42,
			discordID:          "123",
		},
		{
			name:               "Unmarshal Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr:            true,
		},
		{
			name:               "Get Tag Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id": "456"}`)),
			wantErr:            true,
			discordID:          "456",
		},
		{
			name:               "Publish Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id": "789"}`)),
			wantErr:            true,
			wantTagNumber:      99,
			discordID:          "789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Successful Get" {
				tt.leaderboardService.EXPECT().GetTagByDiscordID(gomock.Any(), tt.discordID).Return(tt.wantTagNumber, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.LeaderboardStreamName, gomock.Any()).Return(nil)
			} else if tt.name == "Get Tag Error" {
				tt.leaderboardService.EXPECT().GetTagByDiscordID(gomock.Any(), tt.discordID).Return(0, errors.New("get tag error"))
			} else if tt.name == "Publish Error" {
				tt.leaderboardService.EXPECT().GetTagByDiscordID(gomock.Any(), tt.discordID).Return(tt.wantTagNumber, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.LeaderboardStreamName, gomock.Any()).Return(errors.New("publish error"))
			}

			if err := h.HandleGetTagByDiscordIDRequest(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleGetTagByDiscordIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleCheckTagAvailabilityRequest(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
		wantIsAvailable    bool // Expected availability status
		tagNumber          int  // Tag number to check
	}{
		{
			name:               "Tag Available",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number": 123}`)),
			wantErr:            false,
			wantIsAvailable:    true,
			tagNumber:          123,
		},
		{
			name:               "Tag Not Available",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number": 456}`)),
			wantErr:            false,
			wantIsAvailable:    false,
			tagNumber:          456,
		},
		{
			name:               "Unmarshal Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr:            true,
		},
		{
			name:               "Check Availability Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number": 789}`)),
			wantErr:            true,
			tagNumber:          789,
		},
		{
			name:               "Publish Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number": 999}`)),
			wantErr:            true,
			wantIsAvailable:    true,
			tagNumber:          999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Tag Available" || tt.name == "Tag Not Available" {
				tt.leaderboardService.EXPECT().CheckTagAvailability(gomock.Any(), tt.tagNumber).Return(tt.wantIsAvailable, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.UserStreamName, gomock.Any()).Return(nil)
			} else if tt.name == "Check Availability Error" {
				tt.leaderboardService.EXPECT().CheckTagAvailability(gomock.Any(), tt.tagNumber).Return(false, errors.New("check availability error"))
			} else if tt.name == "Publish Error" {
				tt.leaderboardService.EXPECT().CheckTagAvailability(gomock.Any(), tt.tagNumber).Return(tt.wantIsAvailable, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.UserStreamName, gomock.Any()).Return(errors.New("publish error"))
			}

			if err := h.HandleCheckTagAvailabilityRequest(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleCheckTagAvailabilityRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagSwapRequest(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
	}{
		{
			name:               "Successful Swap",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"requestor_id": "123", "target_id": "456"}`)),
			wantErr:            false,
		},
		{
			name:               "Unmarshal Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr:            true,
		},
		{
			name:               "Swap Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{"requestor_id": "123", "target_id": "456"}`)),
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Successful Swap" {
				tt.leaderboardService.EXPECT().SwapTags(gomock.Any(), "123", "456").Return(nil)
			} else if tt.name == "Swap Error" {
				tt.leaderboardService.EXPECT().SwapTags(gomock.Any(), "123", "456").Return(errors.New("swap error"))
			}

			if err := h.HandleTagSwapRequest(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagSwapRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		msg                *message.Message
		wantErr            bool
		wantLeaderboard    []events.LeaderboardEntry // Expected leaderboard entries
	}{
		{
			name:               "Successful Get",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{}`)), // Empty message payload
			wantErr:            false,
			wantLeaderboard:    []events.LeaderboardEntry{{TagNumber: "1", DiscordID: "123"}},
		},
		{
			name:               "Get Leaderboard Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{}`)),
			wantErr:            true,
		},
		{
			name:               "Publish Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			msg:                message.NewMessage(watermill.NewUUID(), []byte(`{}`)),
			wantErr:            true,
			wantLeaderboard:    []events.LeaderboardEntry{{TagNumber: "1", DiscordID: "123"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Successful Get" {
				tt.leaderboardService.EXPECT().GetLeaderboard(gomock.Any()).Return(tt.wantLeaderboard, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.LeaderboardStreamName, gomock.Any()).Return(nil)
			} else if tt.name == "Get Leaderboard Error" {
				tt.leaderboardService.EXPECT().GetLeaderboard(gomock.Any()).Return(nil, errors.New("get leaderboard error"))
			} else if tt.name == "Publish Error" {
				tt.leaderboardService.EXPECT().GetLeaderboard(gomock.Any()).Return(tt.wantLeaderboard, nil)
				tt.eventBus.EXPECT().Publish(gomock.Any(), events.LeaderboardStreamName, gomock.Any()).Return(errors.New("publish error"))
			}

			if err := h.HandleGetLeaderboardRequest(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_publishEvent(t *testing.T) {
	tests := []struct {
		name               string
		leaderboardService *leaderboardmocks.MockService
		eventBus           *eventbusmock.MockEventBus
		logger             *slog.Logger
		ctx                context.Context
		subject            string
		streamName         string
		payload            []byte
		wantErr            bool
	}{
		{
			name:               "Successful Publish",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			ctx:                context.Background(),
			subject:            "test.subject",
			streamName:         "test.stream",
			payload:            []byte(`{"key": "value"}`),
			wantErr:            false,
		},
		{
			name:               "Publish Error",
			leaderboardService: leaderboardmocks.NewMockService(gomock.NewController(t)),
			eventBus:           eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
			ctx:                context.Background(),
			subject:            "test.subject",
			streamName:         "test.stream",
			payload:            []byte(`{"key": "value"}`),
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				LeaderboardService: tt.leaderboardService,
				EventBus:           tt.eventBus,
				logger:             tt.logger,
			}

			// Set expectations based on the test case
			if tt.name == "Successful Publish" {
				tt.eventBus.EXPECT().Publish(tt.ctx, tt.streamName, gomock.Any()).Return(nil)
			} else if tt.name == "Publish Error" {
				tt.eventBus.EXPECT().Publish(tt.ctx, tt.streamName, gomock.Any()).Return(errors.New("publish error"))
			}

			if err := h.publishEvent(tt.ctx, tt.subject, tt.streamName, tt.payload); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.publishEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
