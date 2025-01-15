package roundservice

import (
	"context"
	"os"
	"testing"

	"log/slog"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	types "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	rounddbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestRoundService_CreateRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		input   events.RoundCreateRequestPayload
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			input: events.RoundCreateRequestPayload{
				Title:     "Test Round",
				Location:  "Test Location",
				EventType: nil,
				DateTime:  types.DateTimeInput{Date: "2025-01-15", Time: "10:00"},
				State:     string(types.RoundStateUpcoming),
				Participants: []types.ParticipantInput{
					{DiscordID: "user-123", TagNumber: &[]int{1}[0], Response: string(types.ResponseAccept)},
				},
			},
			setup: func() {
				mockRoundDB.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
					Return(nil)

				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundCreated, gomock.Any()).
					Return(nil)

				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundReminder, gomock.Any()).
					Return(nil).Times(2)

				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundStarted, gomock.Any()).
					Return(nil)

				// Add expectation for RoundCreateResponse
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundCreateResponse, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error - Empty Title",
			input: events.RoundCreateRequestPayload{
				Title: "", // Empty title
				DateTime: types.DateTimeInput{
					Date: "2025-01-15", Time: "10:00",
				},
			},
			wantErr: true,
		},
		{
			name: "Error - Invalid DateTime",
			input: events.RoundCreateRequestPayload{
				Title:    "Invalid Round",
				DateTime: types.DateTimeInput{Date: "invalid-date", Time: "invalid-time"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			s := &RoundService{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			}

			err := s.CreateRound(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RoundService.CreateRound() expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RoundService.CreateRound() expected no error, but got %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		event   *events.RoundUpdatedPayload
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			event: &events.RoundUpdatedPayload{
				RoundID:  "round-123",
				Title:    &[]string{"Updated Title"}[0],
				Location: &[]string{"Updated Location"}[0],
				// ... other fields ...
			},
			setup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddbtypes.Round{
						ID: "round-123",
						// ... other round data ...
					}, nil)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), "round-123", gomock.Any()).
					Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		// Add more test cases for error scenarios
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			s := &RoundService{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			}

			if err := s.UpdateRound(context.Background(), tt.event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_DeleteRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		event   *events.RoundDeletedPayload
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			event: &events.RoundDeletedPayload{
				RoundID: "round-123",
			},
			setup: func() {
				mockRoundDB.EXPECT().
					DeleteRound(gomock.Any(), "round-123").
					Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundDeleted, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		// Add more test cases for error scenarios
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			s := &RoundService{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			}

			if err := s.DeleteRound(context.Background(), tt.event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.DeleteRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_StartRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		event   *events.RoundStartedPayload
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			event: &events.RoundStartedPayload{
				RoundID: "round-123",
			},
			setup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddbtypes.Round{ // Using rounddbtypes
						ID: "round-123",
						Participants: []rounddbtypes.Participant{ // Using rounddbtypes
							{DiscordID: "user-1", Response: rounddbtypes.ResponseAccept},    // Using rounddbtypes
							{DiscordID: "user-2", Response: rounddbtypes.ResponseTentative}, // Using rounddbtypes
							// ... other participants
						},
						// ... other round data ...
					}, nil)
				mockRoundDB.EXPECT().
					UpdateRoundState(gomock.Any(), "round-123", rounddbtypes.RoundStateInProgress). // Using rounddbtypes
					Return(nil)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundStarted, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		// Add more test cases for error scenarios
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			s := &RoundService{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			}

			if err := s.StartRound(context.Background(), tt.event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.StartRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
