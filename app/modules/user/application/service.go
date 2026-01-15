package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB         userdb.UserDB
	eventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        usermetrics.UserMetrics
	tracer         trace.Tracer
	serviceWrapper func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error)
}

// NewUserService creates a new UserService.
func NewUserService(
	db userdb.UserDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
) Service {
	return &UserServiceImpl{
		UserDB:   db,
		eventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		serviceWrapper: func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (result UserOperationResult, err error) {
			return serviceWrapper(ctx, operationName, userID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper is a helper function that wraps service operations with common logic.
func serviceWrapper(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error), logger *slog.Logger, metrics usermetrics.UserMetrics, tracer trace.Tracer) (result UserOperationResult, err error) {
	if ctx == nil {
		err := errors.New("context cannot be nil")
		return UserOperationResult{
			Success: nil,
			Failure: nil,
			Error:   err,
		}, err
	}

	if serviceFunc == nil {
		return UserOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("user_id", string(userID)),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, userID)

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, duration, userID)
	}()

	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(userID)),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, userID)
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = UserOperationResult{
				Success: nil,
				Failure: nil,
				Error:   fmt.Errorf("%s", errorMsg),
			}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	result, err = serviceFunc(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, userID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)
	metrics.RecordOperationSuccess(ctx, operationName, userID)

	return result, nil
}

// UserOperationResult represents a generic result from a user operation
type UserOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// MatchParsedScorecard performs exact matching of parsed scorecard player names to Discord users.
// It first attempts normalized username, then normalized display name, and always returns a confirmed payload.
// Unmatched players are skipped (no admin confirmation round-trip) and capped to a reasonable limit to guard against abuse.
func (s *UserServiceImpl) MatchParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (UserOperationResult, error) {
	const (
		maxPlayers  = 512 // guardrail against oversized payloads
		maxNameRune = 128 // avoid pathological player-name sizes
	)

	return s.serviceWrapper(ctx, "MatchParsedScorecard", payload.UserID, func(ctx context.Context) (UserOperationResult, error) {
		if payload.ParsedData == nil {
			return UserOperationResult{Failure: errors.New("parsed_data is nil")}, errors.New("parsed_data is nil")
		}

		if len(payload.ParsedData.PlayerScores) > maxPlayers {
			return UserOperationResult{Failure: fmt.Errorf("too many players in payload: %d > %d", len(payload.ParsedData.PlayerScores), maxPlayers)}, nil
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
					attr.String("guild_id", string(payload.GuildID)),
					attr.String("round_id", payload.RoundID.String()),
				)
				unmatched = append(unmatched, name[:maxNameRune])
				continue
			}
			norm := strings.ToLower(name)

			// Try username first
			user, err := s.UserDB.FindByUDiscUsername(ctx, payload.GuildID, norm)
			if err != nil {
				if err != userdb.ErrUserNotFound {
					return UserOperationResult{Failure: err}, err
				}
				// Try name fallback
				user, err = s.UserDB.FindByUDiscName(ctx, payload.GuildID, norm)
				if err != nil {
					if err != userdb.ErrUserNotFound {
						return UserOperationResult{Failure: err}, err
					}
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
				attr.String("guild_id", string(payload.GuildID)),
				attr.String("round_id", payload.RoundID.String()),
			)
		}

		return UserOperationResult{Success: &userevents.UDiscMatchConfirmedPayloadV1{
			ImportID:     payload.ImportID,
			GuildID:      payload.GuildID,
			RoundID:      payload.RoundID,
			UserID:       payload.UserID,
			ChannelID:    payload.ChannelID,
			Timestamp:    time.Now().UTC(),
			Mappings:     mappings,
			ParsedScores: &payload, // Include the parsed scorecard for round module ingestion
		}}, nil
	})
}

// UpdateUDiscIdentity sets UDisc username/name for a user (stores normalized, applies globally).
func (s *UserServiceImpl) UpdateUDiscIdentity(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, username *string, name *string) (UserOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateUDiscIdentity", userID, func(ctx context.Context) (UserOperationResult, error) {
		// Normalize before passing to DB
		normalizedUsername := normalizeStringPointer(username)
		normalizedName := normalizeStringPointer(name)

		if err := s.UserDB.UpdateUDiscIdentityGlobal(ctx, userID, normalizedUsername, normalizedName); err != nil {
			return UserOperationResult{Failure: fmt.Errorf("failed to update udisc identity: %w", err)}, err
		}

		s.logger.InfoContext(ctx, "Updated UDisc identity globally",
			attr.String("user_id", string(userID)),
			attr.String("udisc_username", safeString(normalizedUsername)),
			attr.String("udisc_name", safeString(normalizedName)),
		)

		return UserOperationResult{Success: true}, nil
	})
}

// FindByUDiscUsername attempts to find a user by UDisc username within a guild.
func (s *UserServiceImpl) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (UserOperationResult, error) {
	return s.serviceWrapper(ctx, "FindByUDiscUsername", sharedtypes.DiscordID(""), func(ctx context.Context) (UserOperationResult, error) {
		user, err := s.UserDB.FindByUDiscUsername(ctx, guildID, username)
		if err != nil {
			return UserOperationResult{Failure: err}, err
		}
		return UserOperationResult{Success: user}, nil
	})
}

// FindByUDiscName attempts to find a user by UDisc name within a guild.
func (s *UserServiceImpl) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (UserOperationResult, error) {
	return s.serviceWrapper(ctx, "FindByUDiscName", sharedtypes.DiscordID(""), func(ctx context.Context) (UserOperationResult, error) {
		user, err := s.UserDB.FindByUDiscName(ctx, guildID, name)
		if err != nil {
			return UserOperationResult{Failure: err}, err
		}
		return UserOperationResult{Success: user}, nil
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

func normalizeNullable(val *string) string {
	if val == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*val))
}
