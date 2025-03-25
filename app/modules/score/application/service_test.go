package scoreservice

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	scoredbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewScoreService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreDB := scoredb.NewMockScoreDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type args struct {
		eventBus eventbus.EventBus
		db       scoredb.MockScoreDB
		logger   *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want Service
	}{
		{
			name: "Successful Creation",
			args: args{
				eventBus: mockEventBus,
				db:       *mockScoreDB,
				logger:   logger,
			},
			want: &ScoreService{
				ScoreDB:  mockScoreDB,
				EventBus: mockEventBus,
				logger:   logger,
			},
		},
		// Add more test cases if needed to test behavior with nil inputs
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewScoreService(tt.args.eventBus, &tt.args.db, tt.args.logger)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewScoreService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreDB := scoredb.NewMockScoreDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		ScoreDB   *scoredb.MockScoreDB
		EventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx   context.Context
		event scoreevents.ProcessRoundScoresRequestPayload
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		mockExpects func(f fields)
	}{
		{
			name: "Successful score processing",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ProcessRoundScoresRequestPayload{
					RoundID: "round1",
					Scores: []scoreevents.ParticipantScore{
						{UserID: "user1", TagNumber: 1234, Score: 95.0},
						{UserID: "user2", TagNumber: 5678, Score: 90.0},
					},
				},
			},
			wantErr: false,
			mockExpects: func(f fields) {
				// Expect scores to be logged in the database
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), gomock.Eq("round1"), gomock.Any(), gomock.Eq("auto")).
					DoAndReturn(func(ctx context.Context, roundID string, scores []scoredbtypes.Score, source string) error {
						// You can add more specific validation for the 'scores' argument here if needed
						return nil
					}).Times(1)

				// Expect a leaderboard update event to be published
				f.EventBus.EXPECT().Publish(scoreevents.LeaderboardUpdateRequested, gomock.Any()).
					DoAndReturn(func(eventName string, msg *message.Message) error {
						if eventName != scoreevents.LeaderboardUpdateRequested {
							return fmt.Errorf("unexpected event: %s", eventName)
						}
						// You can add more specific validation for the message payload here if needed
						return nil
					}).Times(1)
			},
		},
		{
			name: "Database error during logging",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ProcessRoundScoresRequestPayload{
					RoundID: "round1",
					Scores: []scoreevents.ParticipantScore{
						{UserID: "user1", TagNumber: 1234, Score: 95.0},
						{UserID: "user2", TagNumber: 5678, Score: 90.0},
					},
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), gomock.Eq("round1"), gomock.Any(), gomock.Eq("auto")).
					Return(fmt.Errorf("database error")).Times(1)
			},
		},
		{
			name: "Error publishing leaderboard update",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ProcessRoundScoresRequestPayload{
					RoundID: "round1",
					Scores: []scoreevents.ParticipantScore{
						{UserID: "user1", TagNumber: 1234, Score: 95.0},
						{UserID: "user2", TagNumber: 5678, Score: 90.0},
					},
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), gomock.Eq("round1"), gomock.Any(), gomock.Eq("auto")).Return(nil).Times(1)
				f.EventBus.EXPECT().Publish(scoreevents.LeaderboardUpdateRequested, gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ScoreService{
				ScoreDB:  tt.fields.ScoreDB,
				EventBus: tt.fields.EventBus,
				logger:   tt.fields.logger,
			}
			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields)
			}

			if err := s.ProcessRoundScores(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.ProcessRoundScores() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScoreService_CorrectScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreDB := scoredb.NewMockScoreDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		ScoreDB   *scoredb.MockScoreDB
		EventBus  *eventbusmocks.MockEventBus
		logger    *slog.Logger
		eventUtil eventutil.EventUtil
	}
	type args struct {
		ctx   context.Context
		event scoreevents.ScoreUpdateRequestPayload
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		mockExpects func(f fields)
	}{
		{
			name: "Successful score correction",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round1",
					Participant: "user1",
					Score:       func() *int { i := 95; return &i }(),
					TagNumber:   1234,
				},
			},
			wantErr: false,
			mockExpects: func(f fields) {
				// Expect the score to be updated/added in the database
				f.ScoreDB.EXPECT().UpdateOrAddScore(gomock.Any(), gomock.Any()).Return(nil).Times(1)

				// Expect to fetch scores for the round
				f.ScoreDB.EXPECT().GetScoresForRound(gomock.Any(), "round1").Return([]scoredbtypes.Score{
					{UserID: "user1", RoundID: "round1", Score: 95, TagNumber: 1234},
				}, nil).Times(1)

				// Expect the updated scores to be logged
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), "round1", gomock.Any(), "manual").Return(nil).Times(1)

				// Expect a leaderboard update event to be published
				f.EventBus.EXPECT().Publish(scoreevents.LeaderboardUpdateRequested, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Error updating/adding score",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round1",
					Participant: "user1",
					Score:       func() *int { i := 95; return &i }(),
					TagNumber:   1234,
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().UpdateOrAddScore(gomock.Any(), gomock.Any()).Return(fmt.Errorf("database error")).Times(1)
			},
		},
		{
			name: "Error fetching scores",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round1",
					Participant: "user1",
					Score:       func() *int { i := 95; return &i }(),
					TagNumber:   1234,
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().UpdateOrAddScore(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				f.ScoreDB.EXPECT().GetScoresForRound(gomock.Any(), "round1").Return(nil, fmt.Errorf("database error")).Times(1)
			},
		},
		{
			name: "Error logging updated scores",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round1",
					Participant: "user1",
					Score:       func() *int { i := 95; return &i }(),
					TagNumber:   1234,
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().UpdateOrAddScore(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				f.ScoreDB.EXPECT().GetScoresForRound(gomock.Any(), "round1").Return([]scoredbtypes.Score{}, nil).Times(1)
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), "round1", gomock.Any(), "manual").Return(fmt.Errorf("database error")).Times(1)
			},
		},
		{
			name: "Error publishing leaderboard update",
			fields: fields{
				ScoreDB:   mockScoreDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			},
			args: args{
				ctx: context.Background(),
				event: scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round1",
					Participant: "user1",
					Score:       func() *int { i := 95; return &i }(),
					TagNumber:   1234,
				},
			},
			wantErr: true, // Expecting an error
			mockExpects: func(f fields) {
				f.ScoreDB.EXPECT().UpdateOrAddScore(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				f.ScoreDB.EXPECT().GetScoresForRound(gomock.Any(), "round1").Return([]scoredbtypes.Score{}, nil).Times(1)
				f.ScoreDB.EXPECT().LogScores(gomock.Any(), "round1", gomock.Any(), "manual").Return(nil).Times(1)
				f.EventBus.EXPECT().Publish(scoreevents.LeaderboardUpdateRequested, gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ScoreService{
				ScoreDB:  tt.fields.ScoreDB,
				EventBus: tt.fields.EventBus,
				logger:   tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields)
			}

			if err := s.CorrectScore(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.CorrectScore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
