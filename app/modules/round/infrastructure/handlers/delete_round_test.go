package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundDeleteRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              testGuildID,
		RoundID:              testRoundID,
		RequestingUserUserID: testUserID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.RoundDeleteRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle RoundDeleteRequest",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundDeletionFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID: testRoundID,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteValidatedV1,
		},
		{
			name: "Service failure returns delete error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundDeletionFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("unauthorized: only the round creator can delete the round")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteErrorV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundDeletionFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns empty results",
			fakeSetup: func(fake *FakeService) {
				fake.ValidateRoundDeletionFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, nil
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
				logger:      logger,
				helpers:     utils.NewHelper(logger),
			}

			ctx := context.Background()
			results, err := h.HandleRoundDeleteRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteValidatedPayloadV1{
		GuildID: testGuildID,
		RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayloadV1{
			GuildID:              testGuildID,
			RoundID:              testRoundID,
			RequestingUserUserID: testUserID,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		payload         *roundevents.RoundDeleteValidatedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
	}{
		{
			name:            "Successfully transform to RoundDeleteAuthorized",
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteAuthorizedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
				logger:      logger,
				helpers:     utils.NewHelper(logger),
			}

			ctx := context.Background()
			results, err := h.HandleRoundDeleteValidated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteValidated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteValidated() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			if tt.wantResultLen > 0 {
				resultPayload, ok := results[0].Payload.(*roundevents.RoundDeleteAuthorizedPayloadV1)
				if !ok {
					t.Errorf("HandleRoundDeleteValidated() payload type mismatch")
				}
				if resultPayload.GuildID != testGuildID || resultPayload.RoundID != testRoundID {
					t.Errorf("HandleRoundDeleteValidated() payload data mismatch")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteAuthorized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testClubUUID := uuid.New()

	testPayload := &roundevents.RoundDeleteAuthorizedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name             string
		fakeSetup        func(*FakeService, *FakeUserService)
		payload          *roundevents.RoundDeleteAuthorizedPayloadV1
		ctx              context.Context
		wantErr          bool
		wantResultLen    int
		wantResultTopic  string
		expectedErrMsg   string
		checkMetadata    bool
		expectedMetadata map[string]string
	}{
		{
			name: "Successfully delete round",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.DeleteRoundFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
					return results.SuccessResult[bool, error](true), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:         testPayload,
			ctx:             context.Background(),
			wantErr:         false,
			wantResultLen:   3, // Original + Guild Scoped + Club Scoped
			wantResultTopic: roundevents.RoundDeletedV1,
		},
		{
			name: "Successfully delete round with discord message ID in metadata",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.DeleteRoundFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
					return results.SuccessResult[bool, error](true), nil
				}
				u.GetClubUUIDByDiscordGuildIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return testClubUUID, nil
				}
			},
			payload:         testPayload,
			ctx:             context.WithValue(context.Background(), "discord_message_id", "msg-123"),
			wantErr:         false,
			wantResultLen:   3, // Original + Guild Scoped + Club Scoped
			wantResultTopic: roundevents.RoundDeletedV1,
			checkMetadata:   true,
			expectedMetadata: map[string]string{
				"discord_message_id": "msg-123",
			},
		},
		{
			name: "Service failure returns delete error",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.DeleteRoundFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
					return results.FailureResult[bool, error](errors.New("round not found")), nil
				}
			},
			payload:         testPayload,
			ctx:             context.Background(),
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundDeleteErrorV1,
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.DeleteRoundFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			ctx:            context.Background(),
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Unknown result returns empty results",
			fakeSetup: func(fake *FakeService, u *FakeUserService) {
				fake.DeleteRoundFunc = func(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, nil
				}
			},
			payload:       testPayload,
			ctx:           context.Background(),
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
				logger:      logger,
				helpers:     utils.NewHelper(logger),
			}

			results, err := h.HandleRoundDeleteAuthorized(tt.ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleRoundDeleteAuthorized() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleRoundDeleteAuthorized() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
			if tt.checkMetadata && tt.wantResultLen > 0 {
				for k, v := range tt.expectedMetadata {
					if results[0].Metadata[k] != v {
						t.Errorf("HandleRoundDeleteAuthorized() metadata mismatch, key %s: got %s, want %s", k, results[0].Metadata[k], v)
					}
				}
			}
		})
	}
}
