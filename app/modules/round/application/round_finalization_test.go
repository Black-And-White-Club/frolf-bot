package roundservice

import (
	"context"
	"os"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"

	"log/slog"
)

func TestRoundService_FinalizeRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				// Mock UpdateRoundState
				mockRoundDB.EXPECT().
					UpdateRoundState(gomock.Any(), "round-123", rounddbtypes.RoundStateFinalized).
					Return(nil)

				// Mock GetRound
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddbtypes.Round{
						ID: "round-123",
						Participants: []rounddbtypes.Participant{
							{DiscordID: "user-456", Score: &[]int{10}[0], TagNumber: &[]int{123}[0]},
							// ... other participants ...
						},
					}, nil)

				// Mock LogRound
				mockRoundDB.EXPECT().
					LogRound(gomock.Any(), gomock.Any(), rounddbtypes.ScoreUpdateTypeRegular).
					Return(nil)

				// Mock sendRoundDataToScoreModule
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ProcessRoundScoresRequest, gomock.Any()).
					Return(nil)

				// Mock Publish for RoundFinalized
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.RoundFinalized, gomock.Any()).
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

			event := &events.RoundFinalizedPayload{
				RoundID: "round-123",
			}

			if err := s.FinalizeRound(context.Background(), event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.FinalizeRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_logRoundData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundDB := rounddb.NewMockRoundDB(ctrl)

	tests := []struct {
		name       string
		round      *rounddbtypes.Round
		updateType rounddbtypes.ScoreUpdateType
		setup      func()
		wantErr    bool
	}{
		{
			name: "Success",
			round: &rounddbtypes.Round{
				ID: "round-123",
				// ... other round data ...
			},
			updateType: rounddbtypes.ScoreUpdateTypeRegular,
			setup: func() {
				mockRoundDB.EXPECT().
					LogRound(gomock.Any(), gomock.Any(), rounddbtypes.ScoreUpdateTypeRegular).
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
				RoundDB: mockRoundDB,
			}

			if err := s.logRoundData(context.Background(), tt.round, tt.updateType); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.logRoundData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_sendRoundDataToScoreModule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)

	tests := []struct {
		name    string
		round   *rounddbtypes.Round
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			round: &rounddbtypes.Round{
				ID: "round-123",
				Participants: []rounddbtypes.Participant{
					{DiscordID: "user-456", Score: &[]int{10}[0], TagNumber: &[]int{123}[0]},
					// ... other participants ...
				},
				// ... other round data ...
			},
			setup: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ProcessRoundScoresRequest, gomock.Any()).
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
				eventBus: mockEventBus,
			}

			if err := s.sendRoundDataToScoreModule(context.Background(), tt.round); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.sendRoundDataToScoreModule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
