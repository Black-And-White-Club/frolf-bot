package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleScheduledRoundTagSync(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	tn1 := sharedtypes.TagNumber(1)
	tn2 := sharedtypes.TagNumber(13)

	testPayload := &sharedevents.SyncRoundsTagRequestPayloadV1{
		GuildID: testGuildID,
		ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
			sharedtypes.DiscordID("user1"): tn1,
			sharedtypes.DiscordID("user2"): tn2,
		},
	}

	logger := loggerfrolfbot.NoOpLogger

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *sharedevents.SyncRoundsTagRequestPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle scheduled round tag update with changes",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundsWithNewTagsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
					return results.SuccessResult[*roundtypes.ScheduledRoundsSyncResult, error](&roundtypes.ScheduledRoundsSyncResult{
						GuildID: testGuildID,
						Updates: []roundtypes.RoundUpdate{
							{
								RoundID:        testRoundID,
								EventMessageID: "msg-123",
								Participants:   []roundtypes.Participant{},
							},
						},
						TotalChecked: 1,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScheduledRoundsSyncedV1,
		},
		{
			name: "Service returns failure",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundsWithNewTagsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
					return results.FailureResult[*roundtypes.ScheduledRoundsSyncResult, error](errors.New("tag update failed")), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.RoundUpdateErrorV1,
		},
		{
			name: "Service returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundsWithNewTagsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
					return roundservice.UpdateScheduledRoundsWithNewTagsResult{}, errors.New("database error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "database error",
		},
		{
			name: "Service returns empty result",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundsWithNewTagsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
					return roundservice.UpdateScheduledRoundsWithNewTagsResult{}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0,
		},
		{
			name: "Success with no affected rounds returns result with empty updates",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateScheduledRoundsWithNewTagsFunc = func(ctx context.Context, req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest) (roundservice.UpdateScheduledRoundsWithNewTagsResult, error) {
					return results.SuccessResult[*roundtypes.ScheduledRoundsSyncResult, error](&roundtypes.ScheduledRoundsSyncResult{
						GuildID:      testGuildID,
						Updates:      []roundtypes.RoundUpdate{},
						TotalChecked: 1,
					}), nil
				}
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScheduledRoundsSyncedV1,
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
			results, err := h.HandleScheduledRoundTagSync(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScheduledRoundTagSync() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScheduledRoundTagSync() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScheduledRoundTagSync() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScheduledRoundTagSync() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
