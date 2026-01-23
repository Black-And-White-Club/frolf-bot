package roundhandlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
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
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.UpdateRoundRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundUpdateValidatedPayloadV1{
							GuildID: testGuildID,
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
								GuildID: testGuildID,
								RoundID: testRoundID,
								Title:   &testTitle,
							},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateValidatedV1,
		},
		{
			name: "Service failure returns update error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							GuildID: testGuildID,
							Error:   "invalid update request",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				service: mockRoundService,
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
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundUpdateValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle without rescheduling",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundEntityUpdatedPayloadV1{
							GuildID: testGuildID,
							Round: roundtypes.Round{
								ID:    testRoundID,
								Title: testTitle,
							},
						},
					},
					nil,
				)
			},
			payload:         testPayloadNoReschedule,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdatedV1,
		},
		{
			name: "Successfully handle with rescheduling",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundEntityUpdatedPayloadV1{
							GuildID: testGuildID,
							Round: roundtypes.Round{
								ID:        testRoundID,
								Title:     testTitle,
								StartTime: &testStartTime,
							},
						},
					},
					nil,
				)
			},
			payload:         testPayloadWithReschedule,
			wantErr:         false,
			wantResultLen:   2,
			wantResultTopic: roundevents.RoundUpdatedV1,
		},
		{
			name: "Service failure returns update error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							GuildID: testGuildID,
							Error:   "update failed",
						},
					},
					nil,
				)
			},
			payload:         testPayloadNoReschedule,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("database error"),
				)
			},
			payload:        testPayloadNoReschedule,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayloadNoReschedule,
			wantErr:       true,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				service: mockRoundService,
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
			if tt.wantResultLen > 1 && results[1].Topic != roundevents.RoundScheduleUpdatedV1 {
				t.Errorf("HandleRoundUpdateValidated() second result topic = %v, want %v", results[1].Topic, roundevents.RoundScheduleUpdatedV1)
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
		mockSetup      func(*roundmocks.MockService)
		payload        *roundevents.RoundEntityUpdatedPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedErrMsg string
	}{
		{
			name: "Successfully update scheduled round events",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundUpdateSuccessPayloadV1{},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Service failure returns update error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							GuildID: testGuildID,
							Error:   "schedule update failed",
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("scheduling service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "scheduling service error",
		},
		{
			name: "Unknown result returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Payload uses fallback GuildID from Round",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundUpdateSuccessPayloadV1{},
					},
					nil,
				)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				service: mockRoundService,
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
