package roundservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundService_JoinRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := mocks.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	type fields struct {
		RoundDB  rounddb.RoundDB
		eventBus shared.EventBus
		logger   *slog.Logger
	}
	type args struct {
		ctx   context.Context
		event *events.ParticipantResponsePayload
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func()
	}{
		{
			name: "Success",
			fields: fields{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx: context.Background(),
				event: &events.ParticipantResponsePayload{
					RoundID:     "round-123",
					Participant: "user-456",
					Response:    "accept",
				},
			},
			wantErr: false,
			setup: func() {
				// Mock Publish for GetTagNumberRequest
				mockEventBus.EXPECT().Publish(gomock.Any(), events.GetTagNumberRequest, gomock.Any()).Return(nil)

				// Mock Subscribe for GetTagNumberResponse
				mockEventBus.EXPECT().Subscribe(gomock.Any(), events.LeaderboardStreamName, events.GetTagNumberResponse, gomock.Any()).
					DoAndReturn(func(ctx context.Context, stream, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						go func() {
							msg := message.NewMessage(watermill.NewUUID(), nil)
							response := events.GetTagNumberResponsePayload{
								DiscordID: "user-456",
								TagNumber: 123,
							}
							responseData, _ := json.Marshal(response)
							msg.Payload = responseData
							handler(ctx, msg)
						}()
						return nil
					})

				// Mock UpdateParticipant
				mockRoundDB.EXPECT().UpdateParticipant(gomock.Any(), "round-123", rounddb.Participant{
					DiscordID: "user-456",
					TagNumber: &[]int{123}[0],
					Response:  rounddb.Response("ACCEPT"), // Consistent case
				}).Return(nil)

				// Mock Publish for ParticipantJoined
				mockEventBus.EXPECT().Publish(gomock.Any(), events.ParticipantJoined, gomock.Any()).Return(nil)
			},
		},
		{
			name: "GetTagNumberError",
			fields: fields{
				RoundDB:  mockRoundDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			args: args{
				ctx: context.Background(),
				event: &events.ParticipantResponsePayload{
					RoundID:     "round-123",
					Participant: "user-456",
					Response:    "accept",
				},
			},
			wantErr: true,
			setup: func() {
				mockEventBus.EXPECT().Publish(gomock.Any(), events.GetTagNumberRequest, gomock.Any()).
					Return(errors.New("get tag number error"))
			},
		},
		// ... Add more test cases for other error scenarios in JoinRound
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			s := &RoundService{
				RoundDB:  tt.fields.RoundDB,
				eventBus: tt.fields.eventBus,
				logger:   tt.fields.logger,
			}
			if err := s.JoinRound(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.JoinRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_UpdateScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := mocks.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				// Mock UpdateParticipantScore
				mockRoundDB.EXPECT().
					UpdateParticipantScore(gomock.Any(), "round-123", "user-456", 10).Return(nil)

				// Mock GetParticipantsWithResponses (NOT all participants have scores)
				mockRoundDB.EXPECT().
					GetParticipantsWithResponses(gomock.Any(), "round-123", rounddb.ResponseAccept, rounddb.ResponseTentative).
					Return([]rounddb.Participant{
						{DiscordID: "user-456", Score: &[]int{10}[0]},
						{DiscordID: "user-789"}, // This participant has no score
					}, nil)

				// Mock GetRound (called by updateScoreInternal)
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddb.Round{
						ID: "round-123",
						Participants: []rounddb.Participant{
							{DiscordID: "user-456", Score: &[]int{10}[0]},
							// ... other participants if needed for sendRoundDataToScoreModule ...
						},
						// ... other round data if needed ...
					}, nil)

				// Mock LogRound (called by updateScoreInternal)
				mockRoundDB.EXPECT().
					LogRound(gomock.Any(), gomock.Any(), rounddb.ScoreUpdateTypeManual).
					Return(nil)

				// Mock sendRoundDataToScoreModule (called by updateScoreInternal)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ProcessRoundScoresRequest, gomock.Any()).
					Return(nil)

				// Mock Publish for ScoreUpdated (called by updateScoreInternal)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ScoreUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		// ... (other test cases) ...
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

			event := &events.ScoreUpdatedPayload{
				RoundID:     "round-123",
				Participant: "user-456",
				Score:       10,
			}

			if err := s.UpdateScore(context.Background(), event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_UpdateScoreAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := mocks.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				// Mock getUserRole
				mockEventBus.EXPECT().Publish(gomock.Any(), events.GetUserRoleRequest, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), events.UserStreamName, events.GetUserRoleResponse, gomock.Any()).DoAndReturn(
					func(ctx context.Context, stream, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						go func() {
							msg := message.NewMessage(watermill.NewUUID(), nil)
							response := events.GetUserRoleResponsePayload{
								DiscordID: "user-456",
								Role:      "Admin",
							}
							responseData, _ := json.Marshal(response)
							msg.Payload = responseData
							handler(ctx, msg)
						}()
						return nil
					})

				// Mock GetRound (called by updateScoreInternal)
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddb.Round{
						ID: "round-123",
						Participants: []rounddb.Participant{
							{DiscordID: "user-456", Score: &[]int{10}[0]},
							// ... other participants if needed for sendRoundDataToScoreModule ...
						},
						// ... other round data if needed ...
					}, nil)

				// Mock LogRound (called by updateScoreInternal)
				mockRoundDB.EXPECT().
					LogRound(gomock.Any(), gomock.Any(), rounddb.ScoreUpdateTypeManual). // Match any Round object
					Return(nil)

				// Mock sendRoundDataToScoreModule (called by updateScoreInternal)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ProcessRoundScoresRequest, gomock.Any()).
					Return(nil)

				// Mock Publish for ScoreUpdated (called by updateScoreInternal)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ScoreUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "GetUserRoleError",
			setup: func() {
				mockEventBus.EXPECT().Publish(gomock.Any(), events.GetUserRoleRequest, gomock.Any()).Return(errors.New("get user role error"))
			},
			wantErr: true,
		},
		// ... Add more test cases for other error scenarios in UpdateScoreAdmin
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
			event := &events.ScoreUpdatedPayload{
				RoundID:     "round-123",
				Participant: "user-456",
				Score:       10,
			}
			if err := s.UpdateScoreAdmin(context.Background(), event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScoreAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_updateScoreInternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockRoundDB := mocks.NewMockRoundDB(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				// Mock GetRound
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), "round-123").
					Return(&rounddb.Round{
						ID: "round-123",
						Participants: []rounddb.Participant{
							{DiscordID: "user-456", Score: &[]int{10}[0]},
							// ... other participants if needed for sendRoundDataToScoreModule ...
						},
						// ... other round data if needed ...
					}, nil)

				// Mock LogRound
				mockRoundDB.EXPECT().
					LogRound(gomock.Any(), gomock.Any(), rounddb.ScoreUpdateTypeManual).
					Return(nil)

				// Mock sendRoundDataToScoreModule
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ProcessRoundScoresRequest, gomock.Any()).
					Return(nil)

				// Mock Publish for ScoreUpdated
				mockEventBus.EXPECT().
					Publish(gomock.Any(), events.ScoreUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		// ... Add more test cases for error scenarios (e.g., GetRound error, LogRound error, etc.) ...
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
			event := &events.ScoreUpdatedPayload{
				RoundID:     "round-123",
				Participant: "user-456",
				Score:       10,
			}
			if err := s.updateScoreInternal(context.Background(), event); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.updateScoreInternal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
