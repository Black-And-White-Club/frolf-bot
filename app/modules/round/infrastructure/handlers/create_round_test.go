package roundhandlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleCreateRoundRequest(t *testing.T) {
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("This is a test round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now())
	testStartTimeString := "2024-01-01T12:00:00Z"
	testCreateRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.CreateRoundRequestedPayloadV1{
		Title:       testTitle,
		Description: testDescription,
		Location:    testLocation,
		StartTime:   testStartTimeString,
		UserID:      testUserID,
	}

	tests := []struct {
		name           string
		mockSetup      func(*roundmocks.MockService)
		payload        *roundevents.CreateRoundRequestedPayloadV1
		wantErr        bool
		wantResultLen  int
		wantResultTopic string
		expectedErrMsg string
	}{
		{
			name: "Successfully handle CreateRoundRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testCreateRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEntityCreatedV1,
		},
		{
			name: "Service failure returns validation error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundValidationFailedPayloadV1{
							UserID:        testUserID,
							ErrorMessages: []string{"validation failed"},
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundValidationFailedV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					fmt.Errorf("internal error"),
				)
			},
			payload:         testPayload,
			wantErr:         true,
			expectedErrMsg:  "internal error",
		},
		{
			name: "Unknown result returns empty results",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:        testPayload,
			wantErr:        false,
			wantResultLen:  0,
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
			}

			ctx := context.Background()
			results, err := h.HandleCreateRoundRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCreateRoundRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCreateRoundRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleCreateRoundRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleCreateRoundRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundEntityCreated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("This is a test round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testRound := roundtypes.Round{
		ID:          testRoundID,
		Title:       testTitle,
		Description: &testDescription,
		Location:    &testLocation,
		StartTime:   &testStartTime,
		CreatedBy:   testUserID,
	}

	guildID := sharedtypes.GuildID("guild-123")
	testPayload := &roundevents.RoundEntityCreatedPayloadV1{
		GuildID:          guildID,
		Round:            testRound,
		DiscordChannelID: "test-channel-id",
		DiscordGuildID:   "test-guild-id",
	}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundEntityCreatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundEntityCreated",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundCreatedV1,
		},
		{
			name: "Service failure returns creation failed",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{
						Failure: &roundevents.RoundCreationFailedPayloadV1{
							ErrorMessage: "creation failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundCreationFailedV1,
		},
		{
			name: "Service error returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().StoreRound(
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
			name: "Unknown result returns empty results",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					results.OperationResult{},
					nil,
				)
			},
			payload:       testPayload,
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
			}

			ctx := context.Background()
			results, err := h.HandleRoundEntityCreated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundEntityCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundEntityCreated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundEntityCreated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundEntityCreated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundEventMessageIDUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("This is a test round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	guildID := sharedtypes.GuildID("guild-123")

	testPayload := &roundevents.RoundMessageIDUpdatePayloadV1{
		GuildID: guildID,
		RoundID: testRoundID,
	}

	testRound := &roundtypes.Round{
		ID:          testRoundID,
		Title:       testTitle,
		Description: &testDescription,
		Location:    &testLocation,
		StartTime:   &testStartTime,
		CreatedBy:   testUserID,
	}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.RoundMessageIDUpdatePayloadV1
		ctx             context.Context
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully update message ID",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundMessageID(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					"msg-123",
				).Return(testRound, nil)
			},
			payload:         testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEventMessageIDUpdatedV1,
		},
		{
			name: "Missing discord_message_id in context",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				// No setup needed
			},
			payload:        testPayload,
			ctx:            context.Background(),
			wantErr:        true,
			expectedErrMsg: "discord_message_id missing from context",
		},
		{
			name: "Service returns error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundMessageID(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					"msg-123",
				).Return(nil, fmt.Errorf("database error"))
			},
			payload:        testPayload,
			ctx:            context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns nil round",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().UpdateRoundMessageID(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					"msg-123",
				).Return(nil, nil)
			},
			payload:        testPayload,
			ctx:            context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:        true,
			expectedErrMsg: "updated round object is nil",
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
			}

			results, err := h.HandleRoundEventMessageIDUpdate(tt.ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundEventMessageIDUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundEventMessageIDUpdate() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundEventMessageIDUpdate() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundEventMessageIDUpdate() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
