package adapters

import (
	"context"
	"errors"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// StubRoundService is a test stub for roundservice.Service
// It embeds everything to satisfy the interface, but we only implement what we need.
type StubRoundService struct {
	roundservice.Service // Embedded to satisfy interface for methods we don't implement

	GetRoundFunc func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error)
}

func (s *StubRoundService) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
	if s.GetRoundFunc != nil {
		return s.GetRoundFunc(ctx, guildID, roundID)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil // Default return
}

func TestRoundLookupAdapter_GetRound(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	roundID := sharedtypes.RoundID(uuid.New())
	testRound := &roundtypes.Round{ID: roundID}

	t.Run("Success", func(t *testing.T) {
		stubService := &StubRoundService{
			GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
				return results.OperationResult[*roundtypes.Round, error]{
					Success: &testRound,
				}, nil
			},
		}
		adapter := NewRoundLookupAdapter(stubService)

		result, err := adapter.GetRound(ctx, guildID, roundID)
		assert.NoError(t, err)
		assert.Equal(t, testRound, result)
	})

	t.Run("ServiceError", func(t *testing.T) {
		expectedErr := errors.New("service error")
		stubService := &StubRoundService{
			GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
				return results.OperationResult[*roundtypes.Round, error]{}, expectedErr
			},
		}
		adapter := NewRoundLookupAdapter(stubService)

		result, err := adapter.GetRound(ctx, guildID, roundID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("FailureResult", func(t *testing.T) {
		failureErr := errors.New("not found")
		stubService := &StubRoundService{
			GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
				return results.OperationResult[*roundtypes.Round, error]{
					Failure: &failureErr,
				}, nil
			},
		}
		adapter := NewRoundLookupAdapter(stubService)

		result, err := adapter.GetRound(ctx, guildID, roundID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "round lookup failed: not found")
	})

	t.Run("NilFailureResult", func(t *testing.T) {
		// Case: Success is nil, Failure is nil (unlikely but possible return)
		// Simulates "IsFailure() is true but failure is nil" OR "IsFailure() is false but Success is nil"
		// The adapter logic we added:
		// 1. Checks IsFailure()
		// 2. Checks Success == nil

		// To test the "Failure is nil when IsFailure() returns true" branch, we need a Result that returns true for IsFailure() but has nil Failure.
		// Usually results.OperationResult IsFailure() checks `Failure != nil`.
		// If so, that branch is unreachable unless we mock OperationResult behavior (which we can't easily do for a struct).
		// So we focus on the "Success is nil" branch.

		stubService := &StubRoundService{
			GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
				return results.OperationResult[*roundtypes.Round, error]{}, nil
			},
		}
		adapter := NewRoundLookupAdapter(stubService)

		result, err := adapter.GetRound(ctx, guildID, roundID)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
