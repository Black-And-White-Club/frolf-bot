package roundhandlers

import (
	"context"
	"errors"
	"testing"
	"time"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
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
		fakeSetup       func(*FakeService, *FakeUserService, *FakeChallengeLookup)
		payload         *roundevents.CreateRoundRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
		assertPayload   func(t *testing.T, payload *roundevents.RoundEntityCreatedPayloadV1)
	}{
		{
			name: "Successfully handle CreateRoundRequest",
			fakeSetup: func(fake *FakeService, _ *FakeUserService, _ *FakeChallengeLookup) {
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
			name: "Successfully handle CreateRoundRequest preserves request source",
			fakeSetup: func(fake *FakeService, _ *FakeUserService, _ *FakeChallengeLookup) {
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
			payload: &roundevents.CreateRoundRequestedPayloadV1{
				Title:         testTitle,
				Description:   &testDescription,
				Location:      testLocation,
				StartTime:     testStartTimeString,
				UserID:        testUserID,
				RequestSource: func() *string { v := "pwa"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEntityCreatedV1,
			assertPayload: func(t *testing.T, payload *roundevents.RoundEntityCreatedPayloadV1) {
				t.Helper()
				if payload.RequestSource == nil || *payload.RequestSource != "pwa" {
					t.Fatalf("expected request source pwa, got %+v", payload.RequestSource)
				}
			},
		},
		{
			name: "Challenge-linked create canonicalizes guild id and seeds participants",
			fakeSetup: func(fake *FakeService, users *FakeUserService, lookup *FakeChallengeLookup) {
				challengeID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
				clubID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
				challengerID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
				defenderID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

				lookup.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, gotChallengeID uuid.UUID) (*clubdb.ClubChallenge, error) {
					if gotChallengeID != challengeID {
						t.Fatalf("expected challenge id %s, got %s", challengeID, gotChallengeID)
					}
					return &clubdb.ClubChallenge{
						UUID:               challengeID,
						ClubUUID:           clubID,
						Status:             clubtypes.ChallengeStatusAccepted,
						ChallengerUserUUID: challengerID,
						DefenderUserUUID:   defenderID,
					}, nil
				}
				lookup.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, gotClubID uuid.UUID) (*clubdb.Club, error) {
					if gotClubID != clubID {
						t.Fatalf("expected club id %s, got %s", clubID, gotClubID)
					}
					discordGuildID := "discord-guild-123"
					return &clubdb.Club{
						UUID:           clubID,
						DiscordGuildID: &discordGuildID,
					}, nil
				}
				users.GetDiscordIDByUUIDFunc = func(ctx context.Context, userUUID uuid.UUID) (sharedtypes.DiscordID, error) {
					switch userUUID {
					case challengerID:
						return sharedtypes.DiscordID("challenger-discord"), nil
					case defenderID:
						return sharedtypes.DiscordID("defender-discord"), nil
					default:
						t.Fatalf("unexpected user uuid %s", userUUID)
						return "", nil
					}
				}
				users.GetUUIDByDiscordIDFunc = func(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
					if discordID != testUserID {
						t.Fatalf("expected actor discord id %q, got %q", testUserID, discordID)
					}
					return challengerID, nil
				}
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					if req.GuildID != sharedtypes.GuildID("discord-guild-123") {
						t.Fatalf("expected canonical guild id discord-guild-123, got %q", req.GuildID)
					}
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
			payload: &roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "club-uuid-from-pwa",
				Title:       testTitle,
				Description: &testDescription,
				Location:    testLocation,
				StartTime:   testStartTimeString,
				UserID:      testUserID,
				ChallengeID: func() *string { v := "11111111-1111-1111-1111-111111111111"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEntityCreatedV1,
			assertPayload: func(t *testing.T, payload *roundevents.RoundEntityCreatedPayloadV1) {
				t.Helper()
				if payload.GuildID != sharedtypes.GuildID("discord-guild-123") {
					t.Fatalf("expected canonical guild id, got %q", payload.GuildID)
				}
				if payload.Round.GuildID != sharedtypes.GuildID("discord-guild-123") {
					t.Fatalf("expected round guild id to be canonicalized, got %q", payload.Round.GuildID)
				}
				if payload.ChallengeID == nil || *payload.ChallengeID != "11111111-1111-1111-1111-111111111111" {
					t.Fatalf("expected challenge id to be preserved, got %+v", payload.ChallengeID)
				}
				if len(payload.Round.Participants) != 2 {
					t.Fatalf("expected 2 seeded challenge participants, got %d", len(payload.Round.Participants))
				}
				if payload.Round.Participants[0].UserID != "challenger-discord" || payload.Round.Participants[1].UserID != "defender-discord" {
					t.Fatalf("unexpected seeded participants: %+v", payload.Round.Participants)
				}
			},
		},
		{
			name: "Challenge-linked create falls back to club uuid when discord guild mapping is absent",
			fakeSetup: func(fake *FakeService, users *FakeUserService, lookup *FakeChallengeLookup) {
				challengeID := uuid.MustParse("12111111-1111-1111-1111-111111111111")
				clubID := uuid.MustParse("23222222-2222-2222-2222-222222222222")
				challengerID := uuid.MustParse("34333333-3333-3333-3333-333333333333")
				defenderID := uuid.MustParse("45444444-4444-4444-4444-444444444444")

				lookup.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, gotChallengeID uuid.UUID) (*clubdb.ClubChallenge, error) {
					if gotChallengeID != challengeID {
						t.Fatalf("expected challenge id %s, got %s", challengeID, gotChallengeID)
					}
					return &clubdb.ClubChallenge{
						UUID:               challengeID,
						ClubUUID:           clubID,
						Status:             clubtypes.ChallengeStatusAccepted,
						ChallengerUserUUID: challengerID,
						DefenderUserUUID:   defenderID,
					}, nil
				}
				lookup.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, gotClubID uuid.UUID) (*clubdb.Club, error) {
					if gotClubID != clubID {
						t.Fatalf("expected club id %s, got %s", clubID, gotClubID)
					}
					return &clubdb.Club{UUID: clubID}, nil
				}
				users.GetDiscordIDByUUIDFunc = func(ctx context.Context, userUUID uuid.UUID) (sharedtypes.DiscordID, error) {
					switch userUUID {
					case challengerID:
						return sharedtypes.DiscordID("challenger-discord"), nil
					case defenderID:
						return sharedtypes.DiscordID("defender-discord"), nil
					default:
						t.Fatalf("unexpected user uuid %s", userUUID)
						return "", nil
					}
				}
				users.GetUUIDByDiscordIDFunc = func(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
					if discordID != testUserID {
						t.Fatalf("expected actor discord id %q, got %q", testUserID, discordID)
					}
					return challengerID, nil
				}
				users.GetUserRoleFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserRoleResult, error) {
					t.Fatalf("expected participant scheduling to bypass role lookup, got guild %q user %q", guildID, userID)
					return userservice.UserRoleResult{}, nil
				}
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					if req.GuildID != sharedtypes.GuildID(clubID.String()) {
						t.Fatalf("expected club uuid scope %q, got %q", clubID.String(), req.GuildID)
					}
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
			payload: &roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "club-uuid-from-pwa",
				Title:       testTitle,
				Description: &testDescription,
				Location:    testLocation,
				StartTime:   testStartTimeString,
				UserID:      testUserID,
				ChallengeID: func() *string { v := "12111111-1111-1111-1111-111111111111"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEntityCreatedV1,
			assertPayload: func(t *testing.T, payload *roundevents.RoundEntityCreatedPayloadV1) {
				t.Helper()
				expectedGuildID := sharedtypes.GuildID("23222222-2222-2222-2222-222222222222")
				if payload.GuildID != expectedGuildID {
					t.Fatalf("expected fallback guild id %q, got %q", expectedGuildID, payload.GuildID)
				}
				if payload.Round.GuildID != expectedGuildID {
					t.Fatalf("expected round guild id to use club uuid scope, got %q", payload.Round.GuildID)
				}
				if len(payload.Round.Participants) != 2 {
					t.Fatalf("expected seeded challenge participants, got %d", len(payload.Round.Participants))
				}
			},
		},
		{
			name: "Challenge-linked create rejects non-participant actors before storing",
			fakeSetup: func(fake *FakeService, users *FakeUserService, lookup *FakeChallengeLookup) {
				challengeID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
				clubID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
				challengerID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
				defenderID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
				otherActorID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

				lookup.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, gotChallengeID uuid.UUID) (*clubdb.ClubChallenge, error) {
					if gotChallengeID != challengeID {
						t.Fatalf("expected challenge id %s, got %s", challengeID, gotChallengeID)
					}
					return &clubdb.ClubChallenge{
						UUID:               challengeID,
						ClubUUID:           clubID,
						Status:             clubtypes.ChallengeStatusAccepted,
						ChallengerUserUUID: challengerID,
						DefenderUserUUID:   defenderID,
					}, nil
				}
				lookup.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, gotClubID uuid.UUID) (*clubdb.Club, error) {
					if gotClubID != clubID {
						t.Fatalf("expected club id %s, got %s", clubID, gotClubID)
					}
					discordGuildID := "discord-guild-456"
					return &clubdb.Club{
						UUID:           clubID,
						DiscordGuildID: &discordGuildID,
					}, nil
				}
				users.GetUUIDByDiscordIDFunc = func(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
					if discordID != testUserID {
						t.Fatalf("expected actor discord id %q, got %q", testUserID, discordID)
					}
					return otherActorID, nil
				}
				users.GetUserRoleFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserRoleResult, error) {
					if guildID != sharedtypes.GuildID("discord-guild-456") {
						t.Fatalf("expected canonical guild id discord-guild-456, got %q", guildID)
					}
					if userID != testUserID {
						t.Fatalf("expected actor discord id %q, got %q", testUserID, userID)
					}
					return results.SuccessResult[sharedtypes.UserRoleEnum, error](sharedtypes.UserRoleUser), nil
				}
				fake.ValidateRoundCreationWithClockFunc = func(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (roundservice.CreateRoundResult, error) {
					t.Fatal("expected validation to fail before round creation service call")
					return roundservice.CreateRoundResult{}, nil
				}
			},
			payload: &roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "club-uuid-from-pwa",
				Title:       testTitle,
				Description: &testDescription,
				Location:    testLocation,
				StartTime:   testStartTimeString,
				UserID:      testUserID,
				ChallengeID: func() *string { v := "55555555-5555-5555-5555-555555555555"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundValidationFailedV1,
		},
		{
			name: "Service failure returns validation error",
			fakeSetup: func(fake *FakeService, _ *FakeUserService, _ *FakeChallengeLookup) {
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
			fakeSetup: func(fake *FakeService, _ *FakeUserService, _ *FakeChallengeLookup) {
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
			fakeSetup: func(fake *FakeService, _ *FakeUserService, _ *FakeChallengeLookup) {
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
			fakeUserService := NewFakeUserService()
			fakeChallengeLookup := &FakeChallengeLookup{}
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService, fakeUserService, fakeChallengeLookup)
			}

			h := &RoundHandlers{
				service:         fakeService,
				userService:     fakeUserService,
				challengeLookup: fakeChallengeLookup,
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
			if tt.assertPayload != nil && len(results) > 0 {
				payload, ok := results[0].Payload.(*roundevents.RoundEntityCreatedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundEntityCreatedPayloadV1, got %T", results[0].Payload)
				}
				tt.assertPayload(t, payload)
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
		name            string
		fakeSetup       func(*FakeService, *FakeUserService)
		payload         *roundevents.RoundEntityCreatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
		assertPayload   func(t *testing.T, resultPayload any)
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
			wantResultLen:   3, // Original + Guild Scoped + Club Scoped
			wantResultTopic: roundevents.RoundCreatedV2,
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
				RequestSource:    func() *string { v := "discord"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   3,
			wantResultTopic: roundevents.RoundCreatedV2,
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
		},
		{
			name: "PWA request source schedules directly even when discord channel id is already present",
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
				fake.ScheduleRoundEventsFunc = func(ctx context.Context, req *roundtypes.ScheduleRoundEventsRequest) (roundservice.ScheduleRoundEventsResult, error) {
					if req.GuildID != guildID {
						t.Fatalf("expected guild id %q, got %q", guildID, req.GuildID)
					}
					if req.RoundID != testRoundID {
						t.Fatalf("expected round id %q, got %q", testRoundID, req.RoundID)
					}
					if req.ChannelID != "pwa-provided-channel-id" {
						t.Fatalf("expected channel id %q, got %q", "pwa-provided-channel-id", req.ChannelID)
					}
					if req.NativeEventPlanned == nil || *req.NativeEventPlanned {
						t.Fatalf("expected NativeEventPlanned=false, got %+v", req.NativeEventPlanned)
					}
					return results.SuccessResult[*roundtypes.ScheduleRoundEventsResult, error](&roundtypes.ScheduleRoundEventsResult{}), nil
				}
			},
			payload: &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          guildID,
				Round:            testRound,
				DiscordChannelID: "pwa-provided-channel-id",
				DiscordGuildID:   "test-guild-id",
				RequestSource:    func() *string { v := "pwa"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   3,
			wantResultTopic: roundevents.RoundCreatedV2,
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
		},
		{
			name: "Successfully handle RoundEntityCreated preserves challenge id",
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
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload: &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          guildID,
				Round:            testRound,
				DiscordChannelID: "test-channel-id",
				DiscordGuildID:   "test-guild-id",
				ChallengeID:      func() *string { v := "challenge-6"; return &v }(),
			},
			wantErr:         false,
			wantResultLen:   4,
			wantResultTopic: roundevents.RoundCreatedV2,
			assertPayload: func(t *testing.T, resultPayload any) {
				t.Helper()
				createdPayload, ok := resultPayload.(*roundevents.RoundCreatedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.RoundCreatedPayloadV1, got %T", resultPayload)
				}
				if createdPayload.ChallengeID == nil || *createdPayload.ChallengeID != "challenge-6" {
					t.Fatalf("expected challenge id challenge-6, got %+v", createdPayload.ChallengeID)
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
			if tt.payload != nil && tt.payload.ChallengeID != nil {
				foundChallengeLink := false
				for _, result := range results {
					if result.Topic == clubevents.ChallengeRoundLinkRequestedV1 {
						foundChallengeLink = true
						break
					}
				}
				if !foundChallengeLink {
					t.Fatalf("expected challenge round link request result when challenge_id is present")
				}
			}
			if tt.assertPayload != nil && len(results) > 0 {
				tt.assertPayload(t, results[0].Payload)
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
			name: "Propagates challenge id to the scheduled payload",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateRoundMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return testRound, nil
				}
			},
			payload: &roundevents.RoundMessageIDUpdatePayloadV1{
				GuildID:     guildID,
				RoundID:     testRoundID,
				ChallengeID: func() *string { v := "challenge-8"; return &v }(),
			},
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundEventMessageIDUpdatedV1,
			assertPayload: func(t *testing.T, payload roundevents.RoundScheduledPayloadV1) {
				t.Helper()
				if payload.ChallengeID == nil || *payload.ChallengeID != "challenge-8" {
					t.Fatalf("expected challenge id challenge-8, got %+v", payload.ChallengeID)
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
