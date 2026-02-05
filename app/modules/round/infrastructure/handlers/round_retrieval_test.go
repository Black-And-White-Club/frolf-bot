package roundhandlers

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleGetRoundRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("Test Description")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
	testUserID := sharedtypes.DiscordID("user-123")

	testPayload := &roundevents.GetRoundRequestPayloadV1{
		GuildID: testGuildID,
		// ClubUUID: testClubUUID, // Not yet in shared library struct
		RoundID: testRoundID,
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.GetRoundRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully retrieve round",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:          testRoundID,
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   &testStartTime,
						CreatedBy:   testUserID,
						GuildID:     testGuildID,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
		},
		{
			name: "Service returns failure - round not found",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.FailureResult[*roundtypes.Round, error](errors.New("round not found")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievalFailedV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, errors.New("database connection error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database connection error",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.OperationResult[*roundtypes.Round, error]{}, nil
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "unexpected empty result from GetRound service",
		},
		{
			name: "Service returns unexpected payload type",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
		},
		{
			name: "Successfully retrieve minimal round data",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:      testRoundID,
						Title:   testTitle,
						GuildID: testGuildID,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
		},
		{
			name: "Successfully retrieve round with participants",
			fakeSetup: func(fake *FakeService) {
				testScore := sharedtypes.Score(65)
				fake.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
					return results.SuccessResult[*roundtypes.Round, error](&roundtypes.Round{
						ID:          testRoundID,
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   &testStartTime,
						CreatedBy:   testUserID,
						GuildID:     testGuildID,
						Participants: []roundtypes.Participant{
							{
								UserID:   sharedtypes.DiscordID("user1"),
								Response: roundtypes.ResponseAccept,
								Score:    &testScore,
							},
							{
								UserID:   sharedtypes.DiscordID("user2"),
								Response: roundtypes.ResponseDecline,
							},
						},
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundRetrievedV1,
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
			}

			ctx := context.Background()
			results, err := h.HandleGetRoundRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetRoundRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetRoundRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleGetRoundRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleGetRoundRequest() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
