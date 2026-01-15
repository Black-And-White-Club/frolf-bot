package saga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

type SwapSagaCoordinator struct {
	kv      jetstream.KeyValue
	logger  *slog.Logger
	service leaderboardservice.Service
}

func NewSwapSagaCoordinator(kv jetstream.KeyValue, service leaderboardservice.Service, logger *slog.Logger) *SwapSagaCoordinator {
	return &SwapSagaCoordinator{
		kv:      kv,
		service: service,
		logger:  logger,
	}
}

// ProcessIntent saves a new intent and checks if it completes an N-way swap chain.
func (s *SwapSagaCoordinator) ProcessIntent(ctx context.Context, intent SwapIntent) error {
	key := fmt.Sprintf("intents.%s.%s", intent.GuildID, intent.UserID)
	data, _ := json.Marshal(intent)

	if _, err := s.kv.Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to store swap intent: %w", err)
	}

	allIntents, err := s.getGuildIntents(ctx, intent.GuildID)
	if err != nil {
		return err
	}

	cycle := s.findCycle(intent.UserID, allIntents)
	if len(cycle) == 0 {
		s.logger.InfoContext(ctx, "Intent stored, no cycle detected yet",
			attr.String("user_id", string(intent.UserID)))
		return nil
	}

	return s.executeCycle(ctx, intent.GuildID, cycle, allIntents)
}

func (s *SwapSagaCoordinator) findCycle(start sharedtypes.DiscordID, intents map[sharedtypes.DiscordID]SwapIntent) []sharedtypes.DiscordID {
	visited := make(map[sharedtypes.DiscordID]bool)
	path := []sharedtypes.DiscordID{}
	curr := start

	for curr != "" {
		if visited[curr] {
			for i, u := range path {
				if u == curr {
					return path[i:]
				}
			}
			return nil
		}
		visited[curr] = true
		path = append(path, curr)

		targetTag := intents[curr].TargetTag
		var nextUser sharedtypes.DiscordID
		for _, potential := range intents {
			if potential.CurrentTag == targetTag {
				nextUser = potential.UserID
				break
			}
		}
		curr = nextUser
	}
	return nil
}

func (s *SwapSagaCoordinator) executeCycle(ctx context.Context, guildID sharedtypes.GuildID, cycle []sharedtypes.DiscordID, intents map[sharedtypes.DiscordID]SwapIntent) error {

	s.logger.InfoContext(ctx, "Circular swap detected, executing batch",
		attr.Int("chain_length", len(cycle)),
		attr.Any("cycle_path", cycle))

	requests := make([]sharedtypes.TagAssignmentRequest, 0, len(cycle))
	for _, userID := range cycle {
		intent := intents[userID]
		requests = append(requests, sharedtypes.TagAssignmentRequest{
			UserID:    intent.UserID,
			TagNumber: intent.TargetTag,
		})
	}

	batchID := sharedtypes.RoundID(uuid.New())

	_, err := s.service.ExecuteBatchTagAssignment(
		ctx,
		guildID,
		requests,
		batchID,
		sharedtypes.ServiceUpdateSourceTagSwap,
	)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to execute saga cycle batch",
			attr.Error(err),
			attr.String("batch_id", batchID.String()))
		return err
	}

	// Cleanup KV entries
	for _, userID := range cycle {
		key := fmt.Sprintf("intents.%s.%s", guildID, userID)
		if err := s.kv.Delete(ctx, key); err != nil {
			s.logger.WarnContext(ctx, "Failed to delete processed intent",
				attr.String("key", key),
				attr.Error(err))
		}
	}

	return nil
}

func (s *SwapSagaCoordinator) getGuildIntents(ctx context.Context, guildID sharedtypes.GuildID) (map[sharedtypes.DiscordID]SwapIntent, error) {

	intents := make(map[sharedtypes.DiscordID]SwapIntent)
	prefix := fmt.Sprintf("intents.%s.", guildID)

	keys, err := s.kv.Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return intents, nil
		}
		return nil, err
	}

	for _, k := range keys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		entry, err := s.kv.Get(ctx, k)
		if err != nil {
			continue
		}

		var intent SwapIntent
		if err := json.Unmarshal(entry.Value(), &intent); err == nil {
			intents[intent.UserID] = intent
		}
	}

	return intents, nil
}
