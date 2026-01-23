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

	basePayload := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
		GuildID:   testGuildID,
		RoundID:   testRoundID,
		Scores:    []sharedtypes.ScoreInfo{{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber}},
		Overwrite: true,
	}

	tests := []struct {
		name           string
		payload        *sharedevents.ProcessRoundScoresRequestedPayloadV1
		mockSetup      func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1)
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle ProcessRoundScoresRequest",
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeSingles
				return &p
			}(),
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					payload.GuildID,
					payload.RoundID,
					payload.Scores,
					payload.Overwrite,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &sharedevents.ProcessRoundScoresSucceededPayloadV1{
							GuildID: payload.GuildID,
							RoundID: payload.RoundID,
							TagMappings: []sharedtypes.TagMapping{
								{DiscordID: testUserID, TagNumber: testTagNumber},
							},
						},
					}, nil,
				)
			},
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
			name:    "Nil payload",
			payload: nil,
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				// No expectations
			},
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "Service failure in ProcessRoundScores",
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeSingles
				return &p
			}(),
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				failurePayload := &sharedevents.ProcessRoundScoresFailedPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Reason:  "internal service error",
				}
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					payload.GuildID,
					payload.RoundID,
					payload.Scores,
					payload.Overwrite,
				).Return(scoreservice.ScoreOperationResult{Failure: failurePayload}, nil)
			},
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
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeSingles
				return &p
			}(),
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					payload.GuildID,
					payload.RoundID,
					payload.Scores,
					payload.Overwrite,
				).Return(scoreservice.ScoreOperationResult{
					Success: "wrong type", // Simulate an unexpected success type
				}, nil)
			},
			wantErr:        true,
			expectedErrMsg: "unexpected success payload type",
		},
		{
			name: "Service returns direct error",
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeSingles
				return &p
			}(),
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					payload.GuildID,
					payload.RoundID,
					payload.Scores,
					payload.Overwrite,
				).Return(scoreservice.ScoreOperationResult{}, fmt.Errorf("direct service error"))
			},
			wantErr:        true,
			expectedErrMsg: "direct service error",
		},
		{
			name: "Service returns wrong failure payload type",
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeSingles
				return &p
			}(),
			mockSetup: func(mockScoreService *scoremocks.MockService, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) {
				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					payload.GuildID,
					payload.RoundID,
					payload.Scores,
					payload.Overwrite,
				).Return(scoreservice.ScoreOperationResult{Failure: "wrong type"}, nil)
			},
			wantErr:        true,
			expectedErrMsg: "unexpected failure payload type from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScoreService := scoremocks.NewMockService(ctrl)
			if tt.mockSetup != nil && tt.payload != nil {
				tt.mockSetup(mockScoreService, tt.payload)
			}

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
