package leaderboardhandlers

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/stretchr/testify/assert"
)

func TestHandleEndSeason(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")

	t.Run("Success", func(t *testing.T) {
		service := NewFakeService()
		service.EndSeasonFunc = func(ctx context.Context, gID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
			assert.Equal(t, guildID, gID)
			return results.SuccessResult[bool, error](true), nil
		}

		handlers := &LeaderboardHandlers{service: service}
		payload := &leaderboardevents.EndSeasonPayloadV1{GuildID: guildID}

		res, err := handlers.HandleEndSeason(ctx, payload)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, leaderboardevents.LeaderboardEndSeasonSuccessV1, res[0].Topic)
		successPayload, ok := res[0].Payload.(*leaderboardevents.EndSeasonSuccessPayloadV1)
		assert.True(t, ok)
		assert.Equal(t, guildID, successPayload.GuildID)
	})

	t.Run("Service Error", func(t *testing.T) {
		service := NewFakeService()
		service.EndSeasonFunc = func(ctx context.Context, gID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
			return results.OperationResult[bool, error]{}, errors.New("pipeline error")
		}

		handlers := &LeaderboardHandlers{service: service}
		payload := &leaderboardevents.EndSeasonPayloadV1{GuildID: guildID}

		res, err := handlers.HandleEndSeason(ctx, payload)

		assert.NoError(t, err) // Handlers return events, not errors usually
		assert.Len(t, res, 1)
		assert.Equal(t, leaderboardevents.LeaderboardEndSeasonFailedV1, res[0].Topic)
		failPayload, ok := res[0].Payload.(*leaderboardevents.AdminFailedPayloadV1)
		assert.True(t, ok)
		assert.Equal(t, guildID, failPayload.GuildID)
		assert.Contains(t, failPayload.Reason, "pipeline error")
	})

	t.Run("Operation Failure", func(t *testing.T) {
		service := NewFakeService()
		service.EndSeasonFunc = func(ctx context.Context, gID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
			return results.FailureResult[bool, error](errors.New("business logic error")), nil
		}

		handlers := &LeaderboardHandlers{service: service}
		payload := &leaderboardevents.EndSeasonPayloadV1{GuildID: guildID}

		res, err := handlers.HandleEndSeason(ctx, payload)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, leaderboardevents.LeaderboardEndSeasonFailedV1, res[0].Topic)
		failPayload, ok := res[0].Payload.(*leaderboardevents.AdminFailedPayloadV1)
		assert.True(t, ok)
		assert.Equal(t, guildID, failPayload.GuildID)
		assert.Contains(t, failPayload.Reason, "business logic error")
	})
}
