package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserService handles user-related logic.
type UserService struct {
	repo    userdb.Repository
	logger  *slog.Logger
	metrics usermetrics.UserMetrics
	tracer  trace.Tracer
}

// NewUserService creates a new UserService.
func NewUserService(
	repo userdb.Repository,
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
) *UserService {
	return &UserService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

// operationFunc is the signature for service operation functions.
type operationFunc func(ctx context.Context) (results.OperationResult, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
// This standardizes observability across all service methods.
func (s *UserService) withTelemetry(
	ctx context.Context,
	operationName string,
	userID sharedtypes.DiscordID,
	op operationFunc,
) (result results.OperationResult, err error) {

	if ctx == nil {
		ctx = context.Background()
	}

	// Panic recovery MUST be registered early
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)
			s.metrics.RecordOperationFailure(ctx, operationName, userID)
			result = results.OperationResult{}
		}
	}()

	// Start span
	ctx, span := s.tracer.Start(ctx, operationName,
		trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("user_id", string(userID)),
		),
	)
	defer span.End()

	s.metrics.RecordOperationAttempt(ctx, operationName, userID)

	startTime := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration(ctx, operationName, time.Since(startTime), userID)
	}()

	result, err = op(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		s.metrics.RecordOperationFailure(ctx, operationName, userID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	s.metrics.RecordOperationSuccess(ctx, operationName, userID)
	return result, nil
}

// MatchParsedScorecard performs exact matching of parsed scorecard player names to Discord users.
// It first attempts normalized username, then normalized display name, and always returns a confirmed payload.
// Unmatched players are skipped (no admin confirmation round-trip) and capped to a reasonable limit to guard against abuse.
func (s *UserService) MatchParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (results.OperationResult, error) {
	const (
		maxPlayers  = 512 // guardrail against oversized payloads
		maxNameRune = 128 // avoid pathological player-name sizes
	)

	return s.withTelemetry(ctx, "MatchParsedScorecard", payload.UserID, func(ctx context.Context) (results.OperationResult, error) {
		if payload.ParsedData == nil {
			return results.FailureResult(&struct{ Reason string }{Reason: "parsed_data is nil"}), errors.New("parsed_data is nil")
		}

		if len(payload.ParsedData.PlayerScores) > maxPlayers {
			return results.FailureResult(&struct{ Reason string }{Reason: fmt.Sprintf("too many players in payload: %d > %d", len(payload.ParsedData.PlayerScores), maxPlayers)}), nil
		}

		var mappings []userevents.UDiscConfirmedMappingV1
		var unmatched []string

		for _, player := range payload.ParsedData.PlayerScores {
			name := strings.TrimSpace(player.PlayerName)
			if name == "" {
				continue
			}

			if len([]rune(name)) > maxNameRune {
				// Skip absurdly long names to prevent log/processing abuse
				s.logger.WarnContext(ctx, "Skipping player with overlength name",
					attr.String("user_id", string(payload.GuildID)),
					attr.String("round_id", payload.RoundID.String()),
				)
				unmatched = append(unmatched, name[:maxNameRune])
				continue
			}
			norm := strings.ToLower(name)

			// Try username first
			user, err := s.repo.FindByUDiscUsername(ctx, payload.GuildID, norm)
			if err != nil {
				// Treat not-found as a benign absence and fall through to name fallback
				if errors.Is(err, userdb.ErrNotFound) {
					user = nil
				} else {
					return results.FailureResult(err), err
				}
			}
			if user == nil {
				// Try name fallback
				user, err = s.repo.FindByUDiscName(ctx, payload.GuildID, norm)
				if err != nil {
					if errors.Is(err, userdb.ErrNotFound) {
						user = nil
					} else {
						return results.FailureResult(err), err
					}
				}
				if user == nil {
					unmatched = append(unmatched, name)
					continue
				}
			}

			mappings = append(mappings, userevents.UDiscConfirmedMappingV1{
				PlayerName:    player.PlayerName,
				DiscordUserID: user.User.UserID,
			})
		}

		if len(unmatched) > 0 {
			s.logger.InfoContext(ctx, "Skipping unmatched UDisc players (no admin confirmation flow)",
				attr.String("unmatched_players", strings.Join(unmatched, ",")),
				attr.String("import_id", payload.ImportID),
				attr.String("user_id", string(payload.GuildID)),
				attr.String("round_id", payload.RoundID.String()),
			)
		}

		return results.SuccessResult(&userevents.UDiscMatchConfirmedPayloadV1{
			ImportID:     payload.ImportID,
			GuildID:      payload.GuildID,
			RoundID:      payload.RoundID,
			UserID:       payload.UserID,
			ChannelID:    payload.ChannelID,
			Timestamp:    time.Now().UTC(),
			Mappings:     mappings,
			ParsedScores: &payload, // Include the parsed scorecard for round module ingestion
		}), nil
	})
}

// UpdateUDiscIdentity sets UDisc username/name for a user (stores normalized, applies globally).
func (s *UserService) UpdateUDiscIdentity(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "UpdateUDiscIdentity", userID, func(ctx context.Context) (results.OperationResult, error) {
		// Normalize before passing to DB
		normalizedUsername := normalizeStringPointer(username)
		normalizedName := normalizeStringPointer(name)

		if err := s.repo.UpdateUDiscIdentityGlobal(ctx, userID, normalizedUsername, normalizedName); err != nil {
			return results.FailureResult(&struct{ Reason string }{Reason: fmt.Sprintf("failed to update udisc identity: %v", err)}), err
		}

		s.logger.InfoContext(ctx, "Updated UDisc identity globally",
			attr.String("user_id", string(userID)),
			attr.String("udisc_username", safeString(normalizedUsername)),
			attr.String("udisc_name", safeString(normalizedName)),
		)

		return results.SuccessResult(true), nil
	})
}

// FindByUDiscUsername attempts to find a user by UDisc username within a guild.
func (s *UserService) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "FindByUDiscUsername", sharedtypes.DiscordID(""), func(ctx context.Context) (results.OperationResult, error) {
		user, err := s.repo.FindByUDiscUsername(ctx, guildID, username)
		if err != nil {
			return results.FailureResult(err), err
		}
		return results.SuccessResult(user), nil
	})
}

// FindByUDiscName attempts to find a user by UDisc name within a guild.
func (s *UserService) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "FindByUDiscName", sharedtypes.DiscordID(""), func(ctx context.Context) (results.OperationResult, error) {
		user, err := s.repo.FindByUDiscName(ctx, guildID, name)
		if err != nil {
			return results.FailureResult(err), err
		}
		return results.SuccessResult(user), nil
	})
}

func normalizeStringPointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}

func safeString(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
