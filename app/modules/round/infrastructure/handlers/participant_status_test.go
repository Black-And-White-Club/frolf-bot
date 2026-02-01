package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleParticipantJoinRequest_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("user-123")

	testPayload := &roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:  testRoundID,
		GuildID:  testGuildID,
		UserID:   testUserID,
		Response: roundtypes.ResponseAccept,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(fakeService *FakeService)
		payload         *roundevents.ParticipantJoinRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully validate participant join request",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.CheckParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
					return results.SuccessResult[*roundtypes.ParticipantStatusCheckResult, error](&roundtypes.ParticipantStatusCheckResult{
						Action:   "VALIDATE",
						GuildID:  testGuildID,
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundParticipantJoinValidationRequestedV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.CheckParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
					return results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns failure result",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.CheckParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
					return results.FailureResult[*roundtypes.ParticipantStatusCheckResult, error](errors.New("status check failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundParticipantStatusCheckErrorV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
				logger:      logger,
			}

			results, err := h.HandleParticipantJoinRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantJoinRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleParticipantJoinRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleParticipantJoinRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
