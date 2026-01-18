package scorehandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleProcessRoundScoresRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("user1")
	testScore := sharedtypes.Score(72)
	testTagNumber := sharedtypes.TagNumber(1)

	testProcessRoundScoresRequestedPayloadV1 := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Scores: []sharedtypes.ScoreInfo{
			{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber},
		},
	}

	// Mock dependencies
	mockScoreService := scoremocks.NewMockService(ctrl)

	// no-op observability in handler tests

	tests := []struct {
		name           string
		mockSetup      func()
		payload        *sharedevents.ProcessRoundScoresRequestedPayloadV1
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle ProcessRoundScoresRequest",
			mockSetup: func() {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testProcessRoundScoresRequestedPayloadV1.Scores,
					gomock.Any(),
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &sharedevents.ProcessRoundScoresSucceededPayloadV1{
							GuildID: testGuildID,
							RoundID: testRoundID,
							TagMappings: []sharedtypes.TagMapping{
								{DiscordID: testUserID, TagNumber: testTagNumber},
							},
						},
					},
					nil,
				)
			},
			payload: testProcessRoundScoresRequestedPayloadV1,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.LeaderboardBatchTagAssignmentRequestedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.LeaderboardBatchTagAssignmentRequestedV1, results[0].Topic)
				}
				batchPayload, ok := results[0].Payload.(*sharedevents.BatchTagAssignmentRequestedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: got %T", results[0].Payload)
				}
				if batchPayload.RequestingUserID != "score-service" {
					t.Errorf("expected RequestingUserID 'score-service', got %s", batchPayload.RequestingUserID)
				}
				if len(batchPayload.Assignments) != 1 {
					t.Fatalf("expected 1 assignment, got %d", len(batchPayload.Assignments))
				}
				if batchPayload.Assignments[0].UserID != testUserID {
					t.Errorf("expected UserID %s, got %s", testUserID, batchPayload.Assignments[0].UserID)
				}
				if batchPayload.Assignments[0].TagNumber != testTagNumber {
					t.Errorf("expected TagNumber %d, got %d", testTagNumber, batchPayload.Assignments[0].TagNumber)
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
			name: "Service failure in ProcessRoundScores",
			mockSetup: func() {
				failurePayload := &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Reason:  "internal service error",
				}

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testProcessRoundScoresRequestedPayloadV1.Scores,
					gomock.Any(),
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: failurePayload,
					},
					nil,
				)
			},
			payload: testProcessRoundScoresRequestedPayloadV1,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ProcessRoundScoresFailedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ProcessRoundScoresFailedV1, results[0].Topic)
				}
				failurePayload, ok := results[0].Payload.(*sharedevents.ProcessRoundScoresFailedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: got %T", results[0].Payload)
				}
				if failurePayload.Reason != "internal service error" {
					t.Errorf("expected reason 'internal service error', got %s", failurePayload.Reason)
				}
			},
		},
		{
			name: "Unknown result from ProcessRoundScores",
			mockSetup: func() {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testProcessRoundScoresRequestedPayloadV1.Scores,
					gomock.Any(),
				).Return(
					scoreservice.ScoreOperationResult{}, // Neither success nor failure
					nil,
				)
			},
			payload:        testProcessRoundScoresRequestedPayloadV1,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service: expected ProcessRoundScoresSucceededPayloadV1",
		},
		{
			name: "Service returns direct error",
			mockSetup: func() {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testProcessRoundScoresRequestedPayloadV1.Scores,
					gomock.Any(),
				).Return(
					scoreservice.ScoreOperationResult{}, // No failure payload
					fmt.Errorf("direct service error"),  // Direct error
				)
			},
			payload:        testProcessRoundScoresRequestedPayloadV1,
			wantErr:        true,
			expectedErrMsg: "direct service error",
		},
		{
			name: "Service returns wrong failure payload type",
			mockSetup: func() {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testProcessRoundScoresRequestedPayloadV1.Scores,
					gomock.Any(),
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: "wrong type", // Wrong type, should be *ProcessRoundScoresFailedPayloadV1
					},
					nil,
				)
			},
			payload:        testProcessRoundScoresRequestedPayloadV1,
			wantErr:        true,
			expectedErrMsg: "unexpected failure payload type from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &ScoreHandlers{
				service: mockScoreService,
				helpers: nil,
			}

			ctx := context.Background()
			got, err := h.HandleProcessRoundScoresRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
