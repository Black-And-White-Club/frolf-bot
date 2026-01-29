package roundhandlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testReminderType := "24-hour"
	testUserIDs := []sharedtypes.DiscordID{"user1", "user2", "user3"}

	testPayload := &roundevents.DiscordReminderPayloadV1{
		RoundID:      testRoundID,
		GuildID:      testGuildID,
		ReminderType: testReminderType,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.DiscordReminderPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle round reminder with participants",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return results.SuccessResult[roundtypes.ProcessRoundReminderResult, error](roundtypes.ProcessRoundReminderResult{
						RoundID:      testRoundID,
						GuildID:      testGuildID,
						ReminderType: testReminderType,
						UserIDs:      testUserIDs,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderSentV1,
		},
		{
			name: "Successfully handle round reminder with no participants",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return results.SuccessResult[roundtypes.ProcessRoundReminderResult, error](roundtypes.ProcessRoundReminderResult{
						RoundID:      testRoundID,
						GuildID:      testGuildID,
						ReminderType: testReminderType,
						UserIDs:      []sharedtypes.DiscordID{}, // No participants
					}), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0, // Returns empty results when no participants
		},
		{
			name: "Service returns failure",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return results.FailureResult[roundtypes.ProcessRoundReminderResult, error](errors.New("round not found")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderFailedV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return roundservice.ProcessRoundReminderResult{}, fmt.Errorf("database connection failed")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection failed",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return roundservice.ProcessRoundReminderResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service returns unexpected payload type",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					// In the original test, this was achieved by OperationResult{Success: &roundevents.RoundCreatedPayloadV1{}}
					// With generics, we'd need to bypass type safety to return a wrong Success type.
					// We'll skip mimicking this exact behavior if it's too complex, but returning an empty success result
					// would trigger the "service returned neither success nor failure" case in the handler.
					return roundservice.ProcessRoundReminderResult{}, nil
				}
			},
			payload: testPayload,
			wantErr: true,
		},
		{
			name: "Successfully handle with single participant",
			fakeSetup: func(fake *FakeService) {
				fake.ProcessRoundReminderFunc = func(ctx context.Context, req *roundtypes.ProcessRoundReminderRequest) (roundservice.ProcessRoundReminderResult, error) {
					return results.SuccessResult[roundtypes.ProcessRoundReminderResult, error](roundtypes.ProcessRoundReminderResult{
						RoundID:      testRoundID,
						GuildID:      testGuildID,
						ReminderType: testReminderType,
						UserIDs:      []sharedtypes.DiscordID{"user1"},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundReminderSentV1,
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

			ctx := context.Background()
			results, err := h.HandleRoundReminder(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundReminder() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundReminder() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundReminder() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
