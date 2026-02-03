package userservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserService handles user-related logic.
type UserService struct {
	repo    userdb.Repository
	logger  *slog.Logger
	metrics usermetrics.UserMetrics
	tracer  trace.Tracer
	db      *bun.DB
}

// NewUserService creates a new UserService.
func NewUserService(
	repo userdb.Repository,
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
	db *bun.DB,
) *UserService {
	return &UserService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		db:      db,
	}
}

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func withTelemetry[S any, F any](
	s *UserService,
	ctx context.Context,
	operationName string,
	userID sharedtypes.DiscordID,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {
	if ctx == nil {
		return results.FailureResult[S, F](any(errors.New("context cannot be nil")).(F)), errors.New("context cannot be nil")
	}

	// Start span
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("user_id", string(userID)),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	// Record attempt
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, userID)
	}

	// Track duration
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, time.Since(startTime), userID)
		}
	}()

	// Log operation start
	s.logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, userID)
			}
			span.RecordError(err)
			result = results.OperationResult[S, F]{}
		}
	}()

	// Execute operation
	result, err = op(ctx)

	// Handle Infrastructure Error
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, userID)
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Handle Domain Failure
	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
			attr.Any("failure_payload", *result.Failure),
		)
	}

	// Handle Success
	if result.IsSuccess() {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
		)
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, userID)
	}

	return result, nil
}

func (s *UserService) MatchParsedScorecard(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	playerNames []string,
) (results.OperationResult[*MatchResult, error], error) {

	matchOp := func(ctx context.Context, db bun.IDB) (results.OperationResult[*MatchResult, error], error) {
		return s.executeMatchParsedScorecard(ctx, db, guildID, userID, playerNames)
	}

	result, err := withTelemetry(s, ctx, "MatchParsedScorecard", userID, func(ctx context.Context) (results.OperationResult[*MatchResult, error], error) {
		return matchOp(ctx, s.db)
	})

	if err != nil {
		return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("MatchParsedScorecard failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeMatchParsedScorecard(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	_ sharedtypes.DiscordID,
	playerNames []string,
) (results.OperationResult[*MatchResult, error], error) {
	const (
		maxPlayers  = 512
		maxNameRune = 128
	)

	if len(playerNames) > maxPlayers {
		return results.FailureResult[*MatchResult](fmt.Errorf("too many players: %d", len(playerNames))), nil
	}

	matchResult := &MatchResult{
		Mappings:  []userevents.UDiscConfirmedMappingV1{},
		Unmatched: []string{},
	}

	for _, rawName := range playerNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		if len([]rune(name)) > maxNameRune {
			matchResult.Unmatched = append(matchResult.Unmatched, name[:maxNameRune])
			continue
		}
		norm := strings.ToLower(name)

		// Try Username
		user, err := s.repo.FindByUDiscUsername(ctx, db, guildID, norm)
		if err != nil && !errors.Is(err, userdb.ErrNotFound) {
			return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("error finding by username: %w", err)
		}

		// Try Name if not found
		if user == nil || errors.Is(err, userdb.ErrNotFound) {
			user, err = s.repo.FindByUDiscName(ctx, db, guildID, norm)
			if err != nil && !errors.Is(err, userdb.ErrNotFound) {
				return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("error finding by name: %w", err)
			}
		}

		if user != nil {
			matchResult.Mappings = append(matchResult.Mappings, userevents.UDiscConfirmedMappingV1{
				PlayerName:    rawName,
				DiscordUserID: user.User.GetUserID(),
			})
		} else {
			matchResult.Unmatched = append(matchResult.Unmatched, rawName)
		}
	}

	return results.SuccessResult[*MatchResult, error](matchResult), nil
}

// UpdateUDiscIdentity sets UDisc username/name for a user globally.
func (s *UserService) UpdateUDiscIdentity(
	ctx context.Context,
	userID sharedtypes.DiscordID,
	username *string,
	name *string,
) (results.OperationResult[bool, error], error) {

	updateTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		return s.executeUpdateUDiscIdentity(ctx, db, userID, username, name)
	}

	result, err := withTelemetry(s, ctx, "UpdateUDiscIdentity", userID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, updateTx)
	})

	if err != nil {
		return results.FailureResult[bool](err), fmt.Errorf("UpdateUDiscIdentity failed: %w", err)
	}

	return result, nil
}

// UpdateUserProfile updates user's display name and avatar hash.
func (s *UserService) UpdateUserProfile(
	ctx context.Context,
	userID sharedtypes.DiscordID,
	guildID sharedtypes.GuildID,
	displayName string,
	avatarHash string,
) error {

	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		// Update global profile
		if err := s.repo.UpdateProfile(ctx, db, userID, displayName, avatarHash); err != nil {
			return results.OperationResult[bool, error]{}, err
		}

		// Update club membership if guildID is provided
		if guildID != "" {
			userUUID, err := s.repo.GetUUIDByDiscordID(ctx, db, userID)
			if err != nil {
				return results.OperationResult[bool, error]{}, err
			}
			clubUUID, err := s.repo.GetClubUUIDByDiscordGuildID(ctx, db, guildID)
			if err != nil {
				return results.OperationResult[bool, error]{}, err
			}

			// Upsert club membership with Discord as source
			avatarURL := ""
			if avatarHash != "" {
				avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", string(userID), avatarHash)
			}

			membership := &userdb.ClubMembership{
				UserUUID:    userUUID,
				ClubUUID:    clubUUID,
				DisplayName: &displayName,
				AvatarURL:   &avatarURL,
				Source:      "discord",
				ExternalID:  pointer(string(userID)),
			}

			// Get existing to preserve role
			existing, err := s.repo.GetClubMembership(ctx, db, userUUID, clubUUID)
			if err == nil && existing != nil {
				membership.Role = existing.Role
			} else {
				membership.Role = sharedtypes.UserRoleUser
			}

			if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
				return results.OperationResult[bool, error]{}, err
			}
		}

		return results.SuccessResult[bool, error](true), nil
	}

	_, err := withTelemetry(s, ctx, "UpdateUserProfile", userID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, op)
	})

	return err
}

func (s *UserService) UpdateClubMembership(
	ctx context.Context,
	clubUUID uuid.UUID,
	userUUID uuid.UUID,
	displayName *string,
	avatarURL *string,
	role *sharedtypes.UserRoleEnum,
) (results.OperationResult[bool, error], error) {
	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		// Get existing to determine role if not provided
		existing, err := s.repo.GetClubMembership(ctx, db, userUUID, clubUUID)
		if err != nil && !errors.Is(err, userdb.ErrNotFound) {
			return results.OperationResult[bool, error]{}, err
		}

		membership := &userdb.ClubMembership{
			UserUUID:    userUUID,
			ClubUUID:    clubUUID,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
		}

		if role != nil {
			membership.Role = *role
		} else if existing != nil {
			membership.Role = existing.Role
		} else {
			membership.Role = sharedtypes.UserRoleUser
		}

		if err := s.repo.UpsertClubMembership(ctx, db, membership); err != nil {
			return results.OperationResult[bool, error]{}, err
		}
		return results.SuccessResult[bool, error](true), nil
	}

	return withTelemetry(s, ctx, "UpdateClubMembership", "", func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, op)
	})
}

func (s *UserService) LookupProfiles(
	ctx context.Context,
	userIDs []sharedtypes.DiscordID,
) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
	userIdForTelemetry := sharedtypes.DiscordID("bulk")
	if len(userIDs) > 0 {
		userIdForTelemetry = userIDs[0] + "..."
	}

	op := func(ctx context.Context, db bun.IDB) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
		return s.executeLookupProfiles(ctx, db, userIDs)
	}

	return withTelemetry(s, ctx, "LookupProfiles", userIdForTelemetry, func(ctx context.Context) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
		return runInTx(s, ctx, op)
	})
}

func (s *UserService) executeLookupProfiles(
	ctx context.Context,
	db bun.IDB,
	userIDs []sharedtypes.DiscordID,
) (results.OperationResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error], error) {
	if len(userIDs) == 0 {
		return results.SuccessResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error](make(map[sharedtypes.DiscordID]*usertypes.UserProfile)), nil
	}

	users, err := s.repo.GetByUserIDs(ctx, db, userIDs)
	if err != nil {
		return results.FailureResult[map[sharedtypes.DiscordID]*usertypes.UserProfile](err), nil
	}

	result := make(map[sharedtypes.DiscordID]*usertypes.UserProfile, len(users))
	for _, u := range users {
		result[u.GetUserID()] = &usertypes.UserProfile{
			UserID:      u.GetUserID(),
			DisplayName: u.GetDisplayName(),
			AvatarURL:   u.AvatarURL(64), // 64px for list views
		}
	}

	// For users not in DB, generate default profiles
	for _, id := range userIDs {
		if _, exists := result[id]; !exists {
			result[id] = s.generateDefaultProfile(id)
		}
	}

	return results.SuccessResult[map[sharedtypes.DiscordID]*usertypes.UserProfile, error](result), nil
}

func (s *UserService) generateDefaultProfile(userID sharedtypes.DiscordID) *usertypes.UserProfile {
	idStr := string(userID)
	displayName := "User"
	if len(idStr) > 6 {
		displayName = "User ..." + idStr[len(idStr)-6:]
	}

	// Default avatar
	userIDInt, _ := strconv.ParseUint(idStr, 10, 64)
	index := (userIDInt >> 22) % 6
	avatarURL := fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)

	return &usertypes.UserProfile{
		UserID:      userID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}

func (s *UserService) executeUpdateUDiscIdentity(
	ctx context.Context,
	db bun.IDB,
	userID sharedtypes.DiscordID,
	username *string,
	name *string,
) (results.OperationResult[bool, error], error) {

	if userID == "" {
		return results.FailureResult[bool](ErrInvalidDiscordID), nil
	}

	updates := &userdb.UserUpdateFields{
		UDiscUsername: normalizeStringPointer(username),
		UDiscName:     normalizeStringPointer(name),
	}

	if updates.IsEmpty() {
		return results.SuccessResult[bool, error](true), nil
	}

	if err := s.repo.UpdateGlobalUser(ctx, db, userID, updates); err != nil {
		if errors.Is(err, userdb.ErrNoRowsAffected) {
			return results.FailureResult[bool](userdb.ErrNotFound), nil
		}
		return results.OperationResult[bool, error]{}, fmt.Errorf("failed to update global user: %w", err)
	}

	return results.SuccessResult[bool, error](true), nil
}

func normalizeStringPointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}

// runInTx ensures the operation runs within a database transaction.
// It matches the pattern used in GuildService but adapted for UserService.
func runInTx[S any, F any](
	s *UserService,
	ctx context.Context,
	fn func(ctx context.Context, db bun.IDB) (results.OperationResult[S, F], error),
) (results.OperationResult[S, F], error) {

	// Update this block to match GuildService
	if s.db == nil {
		return fn(ctx, nil)
	}

	var result results.OperationResult[S, F]

	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var txErr error
		result, txErr = fn(ctx, tx)
		return txErr
	})

	return result, err
}

func (s *UserService) GetUUIDByDiscordID(ctx context.Context, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	return s.repo.GetUUIDByDiscordID(ctx, s.db, discordID)
}

func (s *UserService) GetClubUUIDByDiscordGuildID(ctx context.Context, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	return s.repo.GetClubUUIDByDiscordGuildID(ctx, s.db, guildID)
}

func pointer[T any](v T) *T {
	return &v
}
