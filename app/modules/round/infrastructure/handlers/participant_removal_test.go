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

func TestRoundHandlers_HandleParticipantRemovalRequest_Basic(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("user-123")
	testClubUUID := uuid.New()

	testPayload := &roundevents.ParticipantRemovalRequestPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		UserID:  testUserID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService, *FakeUserService)
		wantErr        bool
		expectedErrMsg string
		wantResultLen  int
		expectedTopics []string
		assertPayload  func(t *testing.T, payload any)
	}{
		{
			name: "Successfully handles participant removal with parallel identity topics",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.ParticipantRemovalFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						EventMessageID: "event-msg-1",
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("accept-1"), Response: roundtypes.ResponseAccept},
							{UserID: sharedtypes.DiscordID("decline-1"), Response: roundtypes.ResponseDecline},
							{UserID: sharedtypes.DiscordID("tentative-1"), Response: roundtypes.ResponseTentative},
						},
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			wantErr:       false,
			wantResultLen: 3, // Original + Guild Scoped + Club Scoped
			expectedTopics: []string{
				roundevents.RoundParticipantRemovedV2,
				roundevents.RoundParticipantRemovedV2 + "." + string(testGuildID),
				roundevents.RoundParticipantRemovedV2 + "." + testClubUUID.String(),
			},
			assertPayload: func(t *testing.T, payload any) {
				t.Helper()
				removedPayload, ok := payload.(*roundevents.ParticipantRemovedPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.ParticipantRemovedPayloadV1, got %T", payload)
				}
				if removedPayload.GuildID != testGuildID {
					t.Fatalf("expected guild_id %q, got %q", testGuildID, removedPayload.GuildID)
				}
				if removedPayload.RoundID != testRoundID {
					t.Fatalf("expected round_id %q, got %q", testRoundID, removedPayload.RoundID)
				}
				if removedPayload.UserID != testUserID {
					t.Fatalf("expected user_id %q, got %q", testUserID, removedPayload.UserID)
				}
				if len(removedPayload.AcceptedParticipants) != 1 {
					t.Fatalf("expected 1 accepted participant, got %d", len(removedPayload.AcceptedParticipants))
				}
				if len(removedPayload.DeclinedParticipants) != 1 {
					t.Fatalf("expected 1 declined participant, got %d", len(removedPayload.DeclinedParticipants))
				}
				if len(removedPayload.TentativeParticipants) != 1 {
					t.Fatalf("expected 1 tentative participant, got %d", len(removedPayload.TentativeParticipants))
				}
			},
		},
		{
			name: "Skips club scoped topic when club uuid lookup fails",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.ParticipantRemovalFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:             testRoundID,
						GuildID:        testGuildID,
						EventMessageID: "event-msg-2",
					}), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return uuid.Nil, errors.New("lookup failed")
				}
			},
			wantErr:       false,
			wantResultLen: 2, // Original + Guild Scoped
			expectedTopics: []string{
				roundevents.RoundParticipantRemovedV2,
				roundevents.RoundParticipantRemovedV2 + "." + string(testGuildID),
			},
		},
		{
			name: "Returns removal error event when service reports failure",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.ParticipantRemovalFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("removal failed")), nil
				}
			},
			wantErr:       false,
			wantResultLen: 1,
			expectedTopics: []string{
				roundevents.RoundParticipantRemovalErrorV1,
			},
			assertPayload: func(t *testing.T, payload any) {
				t.Helper()
				errorPayload, ok := payload.(*roundevents.ParticipantRemovalErrorPayloadV1)
				if !ok {
					t.Fatalf("expected *roundevents.ParticipantRemovalErrorPayloadV1, got %T", payload)
				}
				if errorPayload.Error != "removal failed" {
					t.Fatalf("expected error %q, got %q", "removal failed", errorPayload.Error)
				}
			},
		},
		{
			name: "Returns error when service call fails",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.ParticipantRemovalFunc = func(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("database error")
				}
			},
			wantErr:        true,
			expectedErrMsg: "database error",
			wantResultLen:  0,
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
				logger:      logger,
			}

			results, err := h.HandleParticipantRemovalRequest(context.Background(), testPayload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantRemovalRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantRemovalRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleParticipantRemovalRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			for i, topic := range tt.expectedTopics {
				if i >= len(results) {
					t.Fatalf("expected topic index %d = %q but only got %d results", i, topic, len(results))
				}
				if results[i].Topic != topic {
					t.Fatalf("result topic at index %d = %q, want %q", i, results[i].Topic, topic)
				}
			}

			if tt.assertPayload != nil && len(results) > 0 {
				tt.assertPayload(t, results[0].Payload)
			}
		})
	}
}
