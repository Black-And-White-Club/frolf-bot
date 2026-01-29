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
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundStartRequested_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")

	testPayload := &roundevents.RoundStartRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundStartRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundStarted",
			fakeSetup: func(fake *FakeService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:    testRoundID,
						Title: "Test Round",
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1")},
							{UserID: sharedtypes.DiscordID("user2")},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundStartedDiscordV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return roundservice.StartRoundResult{}, errors.New("database connection failed")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection failed",
		},
		{
			name: "Service returns failure result",
			fakeSetup: func(fake *FakeService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("start failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundStartFailedV1,
		},
		{
			name: "Service returns empty result (unknown)",
			fakeSetup: func(fake *FakeService) {
				fake.StartRoundFunc = func(ctx context.Context, req *roundtypes.StartRoundRequest) (roundservice.StartRoundResult, error) {
					return roundservice.StartRoundResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
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

			results, err := h.HandleRoundStartRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStartRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundStartRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundStartRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundStartRequested() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
