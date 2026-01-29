package roundhandlers

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
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
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.CreateRoundRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle CreateRoundRequest",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					return results.SuccessResult[*roundtypes.CreateRoundResult, error](&roundtypes.CreateRoundResult{
						Round: &roundtypes.Round{
							ID:          testCreateRoundID,
							Title:       testTitle,
							Description: testDescription,
							Location:    testLocation,
							StartTime:   &testStartTime,
							CreatedBy:   testUserID,
						},
						ChannelID: "test-channel-id",
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEntityCreatedV1,
		},
		{
			name: "Service failure returns validation error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					return results.FailureResult[*roundtypes.CreateRoundResult, error](errors.New("validation failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundValidationFailedV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					return roundservice.CreateRoundResult{}, errors.New("internal error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "internal error",
		},
		{
			name: "Unknown result returns empty results",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					return roundservice.CreateRoundResult{}, nil
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
		Description: testDescription,
		Location:    testLocation,
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
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundEntityCreatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundEntityCreated",
			fakeSetup: func(fake *FakeService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return results.SuccessResult[*roundtypes.CreateRoundResult, error](&roundtypes.CreateRoundResult{
						Round: &roundtypes.Round{
							ID:          testRoundID,
							Title:       testTitle,
							Description: testDescription,
							Location:    testLocation,
							StartTime:   &testStartTime,
							CreatedBy:   testUserID,
						},
						ChannelID: "test-channel-id",
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   2, // Now returns original + guild-scoped event
			wantResultTopic: roundevents.RoundCreatedV1,
		},
		{
			name: "Service failure returns creation failed",
			fakeSetup: func(fake *FakeService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return results.FailureResult[*roundtypes.CreateRoundResult, error](errors.New("creation failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundCreationFailedV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return roundservice.CreateRoundResult{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns empty results",
			fakeSetup: func(fake *FakeService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return roundservice.CreateRoundResult{}, nil
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
		Description: testDescription,
		Location:    testLocation,
		StartTime:   &testStartTime,
		CreatedBy:   testUserID,
	}

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundMessageIDUpdatePayloadV1
		ctx             context.Context
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully update message ID",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return testRound, nil
				}
			},
			payload:         testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEventMessageIDUpdatedV1,
		},
		{
			name: "Missing discord_message_id in context",
			fakeSetup: func(fake *FakeService) {
				// No setup needed
			},
			payload:        testPayload,
			ctx:            context.Background(),
			wantErr:        true,
			expectedErrMsg: "discord_message_id missing from context",
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return nil, errors.New("database error")
				}
			},
			payload:        testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns nil round",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return nil, nil
				}
			},
			payload:        testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:        true,
			expectedErrMsg: "updated round object is nil",
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
