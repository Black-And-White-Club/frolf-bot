package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleTagNumberFound_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testTagNumber := sharedtypes.TagNumber(42)
	testOriginalResponse := roundtypes.ResponseAccept
	testOriginalJoinedLate := true

	testPayload := &sharedevents.RoundTagLookupResultPayloadV1{
		RoundID:            testRoundID,
		UserID:             testUserID,
		TagNumber:          &testTagNumber,
		OriginalResponse:   testOriginalResponse,
		OriginalJoinedLate: &testOriginalJoinedLate,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *sharedevents.RoundTagLookupResultPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle TagNumberFound",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.UpdateParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:             testRoundID,
						EventMessageID: "msg-id",
						Participants:   []roundtypes.Participant{},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundParticipantJoinedV1,
		},
		{
			name: "Handle UpdateParticipantStatus error",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.UpdateParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("service error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service: fakeService,
				logger:  logger,
			}

			results, err := h.HandleTagNumberFound(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagNumberFound() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagNumberFound() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleTagNumberFound() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleTagNumberFound() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantDeclined_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.ParticipantDeclinedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		UserID:  testUserID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ParticipantDeclinedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ParticipantDeclined",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.UpdateParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						GuildID:        testGuildID,
						ID:             testRoundID,
						EventMessageID: "msg-id",
						Participants: []roundtypes.Participant{
							{UserID: testUserID, Response: roundtypes.ResponseDecline},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   2, // Now returns original + guild-scoped event
			wantResultTopic: roundevents.RoundParticipantJoinedV1,
		},
		{
			name: "Handle UpdateParticipantStatus error",
			fakeSetup: func(fakeService *FakeService) {
				fakeService.UpdateParticipantStatusFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("service error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service: fakeService,
				logger:  logger,
			}

			results, err := h.HandleParticipantDeclined(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantDeclined() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantDeclined() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleParticipantDeclined() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleParticipantDeclined() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
