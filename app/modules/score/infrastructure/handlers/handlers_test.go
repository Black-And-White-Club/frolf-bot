package scorehandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreService := scoreservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type args struct {
		scoreService *scoreservice.MockService
		logger       *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want *ScoreHandlers
	}{
		{
			name: "Successful Creation",
			args: args{
				scoreService: mockScoreService,
				logger:       logger,
			},
			want: &ScoreHandlers{
				scoreService: mockScoreService,
				logger:       logger,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewScoreHandlers(tt.args.scoreService, tt.args.logger)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewScoreHandlers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreHandlers_HandleProcessRoundScoresRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreService := scoreservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		scoreService *scoreservice.MockService
		logger       *slog.Logger
	}

	type args struct {
		msg *message.Message
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(f fields, a args)
	}{
		{
			name: "Successful process round scores request handling",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), scoreevents.ProcessRoundScoresRequestPayload{
					RoundID: "some-round-id",
					Scores:  []scoreevents.ParticipantScore{},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.scoreService.EXPECT().ProcessRoundScores(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), "invalid-payload"),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				// No expectations on the service layer as unmarshalling should fail first
			},
		},
		{
			name: "Service layer error",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), scoreevents.ProcessRoundScoresRequestPayload{
					RoundID: "some-round-id",
					Scores:  []scoreevents.ParticipantScore{},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.scoreService.EXPECT().ProcessRoundScores(gomock.Any(), gomock.Any()).Return(fmt.Errorf("service error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ScoreHandlers{
				scoreService: tt.fields.scoreService,
				logger:       tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields, tt.args)
			}

			if err := h.HandleProcessRoundScoresRequest(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("ScoreHandlers.HandleProcessRoundScoresRequest() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestScoreHandlers_HandleScoreUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreService := scoreservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		scoreService *scoreservice.MockService
		logger       *slog.Logger
	}

	type args struct {
		msg *message.Message
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(f fields, a args)
	}{
		{
			name: "Successful score update request handling",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					TagNumber:   123,
					Score:       func() *int { i := 10; return &i }(),
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.scoreService.EXPECT().CorrectScore(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), "invalid-payload"),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				// No expectations on the service layer as unmarshalling should fail first
			},
		},
		{
			name: "Service layer error",
			fields: fields{
				scoreService: mockScoreService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), scoreevents.ScoreUpdateRequestPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					TagNumber:   123,
					Score:       func() *int { i := 10; return &i }(),
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.scoreService.EXPECT().CorrectScore(gomock.Any(), gomock.Any()).Return(fmt.Errorf("service error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ScoreHandlers{
				scoreService: tt.fields.scoreService,
				logger:       tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields, tt.args)
			}

			if err := h.HandleScoreUpdateRequest(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("ScoreHandlers.HandleScoreUpdateRequest() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

// Helper function to create a test message with a payload
func createTestMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	return msg
}
