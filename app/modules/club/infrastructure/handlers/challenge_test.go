package clubhandlers

import (
	"context"
	"log/slog"
	"testing"
	"time"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleChallengeOpenRequestedFansOutToBaseClubAndGuildTopics(t *testing.T) {
	clubUUID := uuid.New()
	guildID := "guild-123"
	actorUUID := uuid.New()
	challengeID := uuid.New()

	fakeService := NewFakeClubService()
	fakeService.OpenChallengeFunc = func(ctx context.Context, req clubservice.ChallengeOpenRequest) (*clubtypes.ChallengeDetail, error) {
		assert.NotNil(t, req.Scope.ClubUUID)
		assert.Equal(t, clubUUID, *req.Scope.ClubUUID)
		assert.Equal(t, guildID, req.Scope.GuildID)
		assert.NotNil(t, req.Actor.UserUUID)
		assert.Equal(t, actorUUID, *req.Actor.UserUUID)
		assert.Equal(t, "discord-user-2", req.Target.ExternalID)

		return &clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:                 challengeID.String(),
				ClubUUID:           clubUUID.String(),
				DiscordGuildID:     &guildID,
				Status:             clubtypes.ChallengeStatusOpen,
				ChallengerUserUUID: actorUUID.String(),
				DefenderUserUUID:   uuid.NewString(),
				OpenedAt:           time.Now().UTC(),
			},
		}, nil
	}

	handler := NewClubHandlers(fakeService, slog.Default(), noop.NewTracerProvider().Tracer("test"))
	results, err := handler.HandleChallengeOpenRequested(context.Background(), &clubevents.ChallengeOpenRequestedPayloadV1{
		ClubUUID:         clubUUID.String(),
		GuildID:          guildID,
		ActorUserUUID:    actorUUID.String(),
		ActorExternalID:  "discord-user-1",
		TargetExternalID: "discord-user-2",
	})

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, clubevents.ChallengeOpenedV1, results[0].Topic)
	assert.Equal(t, clubevents.ChallengeOpenedV1+"."+clubUUID.String(), results[1].Topic)
	assert.Equal(t, clubevents.ChallengeOpenedV1+"."+guildID, results[2].Topic)
}

func TestHandleChallengeListRequestRepliesOnReplyTopic(t *testing.T) {
	clubUUID := uuid.New()

	fakeService := NewFakeClubService()
	fakeService.ListChallengesFunc = func(ctx context.Context, req clubservice.ChallengeListRequest) ([]clubtypes.ChallengeSummary, error) {
		assert.NotNil(t, req.Scope.ClubUUID)
		assert.Equal(t, clubUUID, *req.Scope.ClubUUID)
		return []clubtypes.ChallengeSummary{
			{
				ID:                 uuid.NewString(),
				ClubUUID:           clubUUID.String(),
				Status:             clubtypes.ChallengeStatusOpen,
				ChallengerUserUUID: uuid.NewString(),
				DefenderUserUUID:   uuid.NewString(),
				OpenedAt:           time.Now().UTC(),
			},
		}, nil
	}

	handler := NewClubHandlers(fakeService, slog.Default(), noop.NewTracerProvider().Tracer("test"))
	ctx := context.WithValue(context.Background(), handlerwrapper.CtxKeyReplyTo, "_INBOX.challenge-list")

	results, err := handler.HandleChallengeListRequest(ctx, &clubevents.ChallengeListRequestPayloadV1{
		ClubUUID: clubUUID.String(),
	})

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "_INBOX.challenge-list", results[0].Topic)

	responsePayload, ok := results[0].Payload.(*clubevents.ChallengeListResponsePayloadV1)
	assert.True(t, ok)
	assert.Len(t, responsePayload.Challenges, 1)
	assert.Equal(t, clubtypes.ChallengeStatusOpen, responsePayload.Challenges[0].Status)
}

func TestHandleLeaderboardTagUpdatedPrefersClubUUIDFromPayload(t *testing.T) {
	clubUUID := uuid.New()
	guildID := "legacy-guild"
	userID := "member-123"
	var capturedReq clubservice.ChallengeRefreshRequest

	fakeService := NewFakeClubService()
	fakeService.RefreshChallengesForMembersFunc = func(ctx context.Context, req clubservice.ChallengeRefreshRequest) ([]clubtypes.ChallengeDetail, error) {
		capturedReq = req
		return []clubtypes.ChallengeDetail{{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:                 uuid.NewString(),
				ClubUUID:           clubUUID.String(),
				DiscordGuildID:     &guildID,
				Status:             clubtypes.ChallengeStatusAccepted,
				ChallengerUserUUID: uuid.NewString(),
				DefenderUserUUID:   uuid.NewString(),
				OpenedAt:           time.Now().UTC(),
			},
		}}, nil
	}

	handler := NewClubHandlers(fakeService, slog.Default(), noop.NewTracerProvider().Tracer("test"))
	clubUUIDString := clubUUID.String()

	results, err := handler.HandleLeaderboardTagUpdated(context.Background(), &leaderboardevents.LeaderboardTagUpdatedPayloadV1{
		GuildID:  sharedtypes.GuildID(guildID),
		ClubUUID: &clubUUIDString,
		UserID:   sharedtypes.DiscordID(userID),
	})

	require.NoError(t, err)
	require.NotNil(t, capturedReq.ClubUUID)
	assert.Equal(t, clubUUID, *capturedReq.ClubUUID)
	assert.Equal(t, guildID, capturedReq.GuildID)
	assert.Equal(t, []string{userID}, capturedReq.ExternalIDs)
	assert.Len(t, results, 3)
	assert.Equal(t, clubevents.ChallengeRefreshedV1, results[0].Topic)
	assert.Equal(t, clubevents.ChallengeRefreshedV1+"."+clubUUID.String(), results[1].Topic)
	assert.Equal(t, clubevents.ChallengeRefreshedV1+"."+guildID, results[2].Topic)
}
