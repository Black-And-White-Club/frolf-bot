package scorehandlers

import (
	"context"
	"fmt"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleCorrectScoreRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &scoreevents.ScoreUpdateRequestedPayloadV1{
		GuildID:   testGuildID,
		RoundID:   testRoundID,
		UserID:    testUserID,
		Score:     testScore,
		TagNumber: &testTagNumber,
	}

	// Mock dependencies
	mockScoreService := scoremocks.NewMockService(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &scoremetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		payload        *scoreevents.ScoreUpdateRequestedPayloadV1
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle CorrectScoreRequest",
			mockSetup: func() {
				successPayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: successPayload,
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				// Expect GetScoresForRound to be called for reprocessing
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber},
				}, nil)
			},
			payload: testPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 2 {
					t.Fatalf("expected 2 results, got %d", len(results))
				}
				// First result: ScoreUpdated
				if results[0].Topic != scoreevents.ScoreUpdatedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ScoreUpdatedV1, results[0].Topic)
				}
				successPayload, ok := results[0].Payload.(*scoreevents.ScoreUpdatedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: got %T", results[0].Payload)
				}
				if successPayload.UserID != testUserID {
					t.Errorf("expected UserID %s, got %s", testUserID, successPayload.UserID)
				}
				// Second result: Reprocess request
				if results[1].Topic != scoreevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ProcessRoundScoresRequestedV1, results[1].Topic)
				}
			},
		},
		{
			name: "Nil payload",
			mockSetup: func() {
				// No expectations - handler returns early
			},
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "Service failure in CorrectScore",
			mockSetup: func() {
				failurePayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Reason:  "internal service error",
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: nil,
						Failure: failurePayload,
						Error:   nil,
					},
					nil,
				)
			},
			payload: testPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != scoreevents.ScoreUpdateFailedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ScoreUpdateFailedV1, results[0].Topic)
				}
				failurePayload, ok := results[0].Payload.(*scoreevents.ScoreUpdateFailedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: got %T", results[0].Payload)
				}
				if failurePayload.Reason != "internal service error" {
					t.Errorf("expected reason 'internal service error', got %s", failurePayload.Reason)
				}
			},
		},
		{
			name: "Service returns error",
			mockSetup: func() {
				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
		{
			name: "GetScoresForRound fails - returns only success result",
			mockSetup: func() {
				successPayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: successPayload,
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(nil, fmt.Errorf("db error"))
			},
			payload: testPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result (no reprocess due to error), got %d", len(results))
				}
				if results[0].Topic != scoreevents.ScoreUpdatedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ScoreUpdatedV1, results[0].Topic)
				}
			},
		},
		{
			name: "Unknown result from CorrectScore",
			mockSetup: func() {
				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(), testGuildID, testRoundID, testUserID, testScore, &testTagNumber,
				).Return(scoreservice.ScoreOperationResult{}, nil)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service: neither success nor failure",
		},
		{
			name: "Wrong failure payload type",
			mockSetup: func() {
				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: "wrong type",
					},
					nil,
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "unexpected failure payload type from service",
		},
		{
			name: "Wrong success payload type",
			mockSetup: func() {
				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: "wrong type",
					},
					nil,
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "unexpected success payload type from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &ScoreHandlers{
				scoreService: mockScoreService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			ctx := context.Background()
			got, err := h.HandleCorrectScoreRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCorrectScoreRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
