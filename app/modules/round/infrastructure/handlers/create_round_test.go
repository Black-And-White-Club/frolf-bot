package roundhandlers

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
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
		Description: &testDescription,
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
				service:     fakeService,
				userService: NewFakeUserService(),
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
	testClubUUID := uuid.New()

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
		name              string
		fakeSetup         func(*FakeService, *FakeUserService)
		payload           *roundevents.RoundEntityCreatedPayloadV1
		wantErr           bool
		wantResultLen     int
		wantResultTopic   string
		wantLastTopic     string
		expectedErrMsg    string
		assertPayload     func(t *testing.T, resultPayload any)
		assertLastPayload func(t *testing.T, resultPayload any)
	}{
		{
			name: "Successfully handle RoundEntityCreated",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
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
						ChannelID: "ignored-service-channel-id",
						GuildConfig: &guildtypes.GuildConfig{
							GuildID:        guildID,
							EventChannelID: "configured-events-channel-id",
						},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   4, // Original + Guild Scoped + Club Scoped + scheduling trigger
			wantResultTopic: roundevents.RoundCreatedV2,
			wantLastTopic:   roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				createdPayload, ok := resultPayload.(*roundevents.RoundCreatedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundCreatedPayloadV1, got %T", resultPayload)
				}
				if createdPayload.ChannelID != "test-channel-id" {
					t.Fatalf("expected original payload channel_id to win, got %q", createdPayload.ChannelID)
				}
				if createdPayload.Config == nil {
					t.Fatalf("expected config fragment to be populated")
				}
				if createdPayload.Config.EventChannelID != "configured-events-channel-id" {
					t.Fatalf(
						"expected config fragment event_channel_id %q, got %q",
						"configured-events-channel-id",
						createdPayload.Config.EventChannelID,
					)
				}
			},
			assertLastPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				scheduledPayload, ok := resultPayload.(*roundevents.RoundScheduledPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundScheduledPayloadV1, got %T", resultPayload)
				}
				if scheduledPayload.ChannelID != "test-channel-id" {
					t.Fatalf("expected scheduled channel_id %q, got %q", "test-channel-id", scheduledPayload.ChannelID)
				}
			},
		},
		{
			name: "Successfully handle RoundEntityCreated falls back to guild event channel when payload channel is empty",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return results.SuccessResult[*roundtypes.CreateRoundResult, error](&roundtypes.CreateRoundResult{
						Round:     &roundtypes.Round{ID: testRoundID},
						ChannelID: "service-channel-id-ignored",
						GuildConfig: &guildtypes.GuildConfig{
							GuildID:        guildID,
							EventChannelID: "guild-fallback-channel-id",
						},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload: &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          guildID,
				Round:            testRound,
				DiscordChannelID: "",
				DiscordGuildID:   "test-guild-id",
			},
			wantErr:         false,
			wantResultLen:   4,
			wantResultTopic: roundevents.RoundCreatedV2,
			wantLastTopic:   roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				createdPayload, ok := resultPayload.(*roundevents.RoundCreatedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundCreatedPayloadV1, got %T", resultPayload)
				}
				if createdPayload.ChannelID != "guild-fallback-channel-id" {
					t.Fatalf(
						"expected fallback channel_id %q, got %q",
						"guild-fallback-channel-id",
						createdPayload.ChannelID,
					)
				}
			},
			assertLastPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				scheduledPayload, ok := resultPayload.(*roundevents.RoundScheduledPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundScheduledPayloadV1, got %T", resultPayload)
				}
				if scheduledPayload.ChannelID != "guild-fallback-channel-id" {
					t.Fatalf("expected scheduled channel_id %q, got %q", "guild-fallback-channel-id", scheduledPayload.ChannelID)
				}
			},
		},
		{
			name: "RoundEntityCreated schedules directly even when discord channel id is already present",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.StoreRoundFunc = func(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (roundservice.CreateRoundResult, error) {
					return results.SuccessResult[*roundtypes.CreateRoundResult, error](&roundtypes.CreateRoundResult{
						Round:     &roundtypes.Round{ID: testRoundID},
						ChannelID: "service-channel-id-ignored",
						GuildConfig: &guildtypes.GuildConfig{
							GuildID:        guildID,
							EventChannelID: "guild-fallback-channel-id",
						},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload: &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          guildID,
				Round:            testRound,
				DiscordChannelID: "pwa-provided-channel-id",
				DiscordGuildID:   "test-guild-id",
			},
			wantErr:         false,
			wantResultLen:   4,
			wantResultTopic: roundevents.RoundCreatedV2,
			wantLastTopic:   roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				createdPayload, ok := resultPayload.(*roundevents.RoundCreatedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundCreatedPayloadV1, got %T", resultPayload)
				}
				if createdPayload.ChannelID != "pwa-provided-channel-id" {
					t.Fatalf(
						"expected payload channel_id %q, got %q",
						"pwa-provided-channel-id",
						createdPayload.ChannelID,
					)
				}
			},
			assertLastPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				scheduledPayload, ok := resultPayload.(*roundevents.RoundScheduledPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundScheduledPayloadV1, got %T", resultPayload)
				}
				if scheduledPayload.ChannelID != "pwa-provided-channel-id" {
					t.Fatalf("expected scheduled channel_id %q, got %q", "pwa-provided-channel-id", scheduledPayload.ChannelID)
				}
			},
		},
		{
			name: "Service failure returns creation failed",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
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
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
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
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
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
			fakeUserService := NewFakeUserService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService, fakeUserService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: fakeUserService,
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
			if tt.wantLastTopic != "" && len(results) > 0 && results[len(results)-1].Topic != tt.wantLastTopic {
				t.Errorf("HandleRoundEntityCreated() last result topic = %v, want %v", results[len(results)-1].Topic, tt.wantLastTopic)
			}
			if tt.assertPayload != nil && len(results) > 0 {
				tt.assertPayload(t, results[0].Payload)
			}
			if tt.assertLastPayload != nil && len(results) > 0 {
				tt.assertLastPayload(t, results[len(results)-1].Payload)
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
		assertPayload   func(t *testing.T, payload roundevents.RoundScheduledPayloadV1)
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
			name: "Includes the stored Discord message id in the scheduled payload",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return testRound, nil
				}
			},
			payload: &roundevents.RoundMessageIDUpdatePayloadV1{
				GuildID: guildID,
				RoundID: testRoundID,
			},
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, payload roundevents.RoundScheduledPayloadV1) {
				t.Helper()
				if payload.EventMessageID != "msg-123" {
					t.Fatalf("expected EventMessageID msg-123, got %s", payload.EventMessageID)
				}
			},
		},
		{
			name: "Propagates native event ownership to the scheduled payload",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return testRound, nil
				}
			},
			payload: &roundevents.RoundMessageIDUpdatePayloadV1{
				GuildID:            guildID,
				RoundID:            testRoundID,
				NativeEventPlanned: func() *bool { v := true; return &v }(),
			},
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, payload roundevents.RoundScheduledPayloadV1) {
				t.Helper()
				if payload.NativeEventPlanned == nil || !*payload.NativeEventPlanned {
					t.Fatalf("expected NativeEventPlanned=true, got %+v", payload.NativeEventPlanned)
				}
			},
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
			ctx:            context.WithValue(context.Background(), "discord_message_id", "msg-123"),
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
			ctx:            context.WithValue(context.Background(), "discord_message_id", "msg-123"),
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
				service:     fakeService,
				userService: NewFakeUserService(),
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
			if tt.assertPayload != nil && len(results) > 0 {
				payload, ok := results[0].Payload.(roundevents.RoundScheduledPayloadV1)
				if !ok {
					t.Fatalf("expected RoundScheduledPayloadV1, got %T", results[0].Payload)
				}
				tt.assertPayload(t, payload)
			}
		})
	}
}
