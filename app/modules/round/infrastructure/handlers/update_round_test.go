package roundhandlers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Updated Round")

	testPayload := &roundevents.UpdateRoundRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Title:   &testTitle,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.UpdateRoundRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundUpdateRequest",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundUpdateWithClockFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error) {
					return results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
						Round: &roundtypes.Round{
							ID:      testRoundID,
							GuildID: testGuildID,
							Title:   testTitle,
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateValidatedV1,
		},
		{
			name: "Service failure returns update error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundUpdateWithClockFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error) {
					return results.FailureResult[*roundtypes.UpdateRoundResult, error](errors.New("invalid update request")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundUpdateWithClockFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error) {
					return roundservice.UpdateRoundResult{}, fmt.Errorf("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundUpdateWithClockFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.UpdateRoundResult, error) {
					return roundservice.UpdateRoundResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       true,
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

			ctx := context.Background()
			results, err := h.HandleRoundUpdateRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundUpdateRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundUpdateRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundUpdateRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Updated Round")
	testStartTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))

	testPayloadNoReschedule := &roundevents.RoundUpdateValidatedPayloadV1{
		GuildID: testGuildID,
		RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
			GuildID: testGuildID,
			RoundID: testRoundID,
			Title:   &testTitle,
		},
	}

	testPayloadWithReschedule := &roundevents.RoundUpdateValidatedPayloadV1{
		GuildID: testGuildID,
		RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
			GuildID:   testGuildID,
			RoundID:   testRoundID,
			Title:     &testTitle,
			StartTime: &testStartTime,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundUpdateValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle without rescheduling",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundEntityFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
					return results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
						Round: &roundtypes.Round{
							ID:      testRoundID,
							GuildID: testGuildID,
							Title:   testTitle,
						},
					}), nil
				}
			},
			payload:         testPayloadNoReschedule,
			wantErr:         false,
			wantResultLen:   2, // Now returns original + guild-scoped event
			wantResultTopic: roundevents.RoundUpdatedV1,
		},
		{
			name: "Successfully handle with rescheduling",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundEntityFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
					return results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
						Round: &roundtypes.Round{
							ID:        testRoundID,
							GuildID:   testGuildID,
							Title:     testTitle,
							StartTime: &testStartTime,
						},
					}), nil
				}
			},
			payload:         testPayloadWithReschedule,
			wantErr:         false,
			wantResultLen:   3, // Now returns original + guild-scoped + reschedule event
			wantResultTopic: roundevents.RoundUpdatedV1,
		},
		{
			name: "Service failure returns update error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundEntityFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
					return results.FailureResult[*roundtypes.UpdateRoundResult, error](errors.New("update failed")), nil
				}
			},
			payload:         testPayloadNoReschedule,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundEntityFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
					return roundservice.UpdateRoundResult{}, fmt.Errorf("database error")
				}
			},
			payload:        testPayloadNoReschedule,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundEntityFunc = func(ctx context.Context, req *roundtypes.UpdateRoundRequest) (roundservice.UpdateRoundResult, error) {
					return roundservice.UpdateRoundResult{}, nil
				}
			},
			payload:       testPayloadNoReschedule,
			wantErr:       true,
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

			ctx := context.Background()
			results, err := h.HandleRoundUpdateValidated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundUpdateValidated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundUpdateValidated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundUpdateValidated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			// With guild-scoped events: [0] = original, [1] = guild-scoped, [2] = reschedule (if present)
			if tt.wantResultLen == 3 && results[2].Topic != roundevents.RoundScheduleUpdatedV1 {
				t.Errorf("HandleRoundUpdateValidated() third result topic = %v, want %v", results[2].Topic, roundevents.RoundScheduleUpdatedV1)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundScheduleUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Test Round")
	testStartTime := sharedtypes.StartTime(time.Now())

	testPayload := &roundevents.RoundEntityUpdatedPayloadV1{
		GuildID: testGuildID,
		Round: roundtypes.Round{
			ID:        testRoundID,
			Title:     testTitle,
			StartTime: &testStartTime,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService)
		payload        *roundevents.RoundEntityUpdatedPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedErrMsg string
	}{
		{
			name: "Successfully update scheduled round events",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundEventsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
					return results.SuccessResult[bool, error](true), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Service failure returns update error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundEventsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
					return results.FailureResult[bool, error](errors.New("schedule update failed")), nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundEventsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
					return roundservice.UpdateScheduledRoundEventsResult{}, fmt.Errorf("scheduling service error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "scheduling service error",
		},
		{
			name: "Unknown result returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundEventsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
					return roundservice.UpdateScheduledRoundEventsResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Payload uses fallback GuildID from Round",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundEventsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (roundservice.UpdateScheduledRoundEventsResult, error) {
					return results.SuccessResult[bool, error](true), nil
				}
			},
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:        testRoundID,
					Title:     testTitle,
					StartTime: &testStartTime,
					GuildID:   testGuildID,
				},
			},
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

			ctx := context.Background()
			results, err := h.HandleRoundScheduleUpdate(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundScheduleUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundScheduleUpdate() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundScheduleUpdate() result length = %d, want %d", len(results), tt.wantResultLen)
			}
		})
	}
}
