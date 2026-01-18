package userservice

import (
	"context"
	"errors"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// normalizeUDiscName normalizes a UDisc name for matching (lowercase, trimmed).
func normalizeUDiscName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// CreateUser creates a user and returns a success or failure payload.
func (s *UserService) CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (results.OperationResult, error) {
	if ctx == nil {
		return results.FailureResult(&userevents.UserCreationFailedPayloadV1{GuildID: guildID, UserID: userID, TagNumber: tag, Reason: ErrNilContext.Error()}), ErrNilContext
	}

	// Handle empty Discord ID - domain failure, not execution error
	if userID == "" {
		return results.FailureResult(&userevents.UserCreationFailedPayloadV1{GuildID: guildID, UserID: userID, TagNumber: tag, Reason: ErrInvalidDiscordID.Error()}), nil
	}

	// Handle negative tag numbers - domain failure, not execution error
	if tag != nil && *tag < 0 {
		return results.FailureResult(&userevents.UserCreationFailedPayloadV1{GuildID: guildID, UserID: userID, TagNumber: tag, Reason: ErrNegativeTagNumber.Error()}), nil
	}

	startTime := time.Now()
	userType := "base"
	source := "user"

	if tag != nil {
		s.metrics.RecordUserCreationByTag(ctx, *tag)
	}
	s.metrics.RecordUserCreationAttempt(ctx, userType, source)

	result, err := s.withTelemetry(ctx, "CreateUser", userID, func(ctx context.Context) (results.OperationResult, error) {
		// Step 1: Check if user exists globally
		globalUser, err := s.repo.GetUserGlobal(ctx, userID)
		userExists := err == nil && globalUser != nil

		if userExists {
			// User exists globally - check if they're already in this guild
			membership, err := s.repo.GetGuildMembership(ctx, userID, guildID)
			if err == nil && membership != nil {
				// Already a member of this guild - domain failure, not execution error
				return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
					GuildID:   guildID,
					UserID:    userID,
					TagNumber: tag,
					Reason:    "user already exists in this guild",
				}), nil
			}

			// User exists but not in this guild - create membership only
			membership = &userdb.GuildMembership{
				UserID:  userID,
				GuildID: guildID,
				Role:    sharedtypes.UserRoleUser,
			}
			if err := s.repo.CreateGuildMembership(ctx, membership); err != nil {
				s.logger.ErrorContext(ctx, "Failed to create guild membership",
					attr.String("user_id", string(userID)),
					attr.String("guild_id", string(guildID)),
					attr.Error(err),
				)
				s.metrics.RecordUserCreationFailure(ctx, userType, source)
				return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
					GuildID:   guildID,
					UserID:    userID,
					TagNumber: tag,
					Reason:    err.Error(),
				}), err
			}

			// Success - returning user to new guild
			s.metrics.RecordUserCreationSuccess(ctx, userType, source)

			return results.SuccessResult(&userevents.UserCreatedPayloadV1{
				GuildID:         guildID,
				UserID:          userID,
				TagNumber:       tag,
				IsReturningUser: true,
			}), nil
		}

		// Step 2: User doesn't exist - create global user
		newUser := &userdb.User{
			UserID: userID,
		}

		// Normalize and set UDisc fields if provided
		if udiscUsername != nil && *udiscUsername != "" {
			normalized := normalizeUDiscName(*udiscUsername)
			newUser.UDiscUsername = &normalized
		}
		if udiscName != nil && *udiscName != "" {
			normalized := normalizeUDiscName(*udiscName)
			newUser.UDiscName = &normalized
		}

		dbStart := time.Now()
		if err := s.repo.CreateGlobalUser(ctx, newUser); err != nil {
			dbDuration := time.Since(dbStart)
			s.metrics.RecordDBQueryDuration(ctx, dbDuration)

			domainErr := translateDBError(err)
			logLevel := "error"

			// For expected errors like duplicate users, log as info or warn
			if errors.Is(domainErr, ErrUserAlreadyExists) {
				logLevel = "warn"
			}

			// Log with appropriate level
			if logLevel == "error" {
				s.logger.ErrorContext(ctx, "Failed to create global user",
					attr.String("user_id", string(userID)),
					attr.String("guild_id", string(guildID)),
					attr.Error(domainErr),
				)
			} else {
				s.logger.WarnContext(ctx, domainErr.Error(),
					attr.String("user_id", string(userID)),
					attr.String("guild_id", string(guildID)),
					attr.String("original_error", err.Error()),
				)
			}

			s.metrics.RecordUserCreationFailure(ctx, userType, source)
			// Duplicate user is a domain-level failure (idempotent semantics);
			// return a failure payload and NO top-level error so handlers publish
			// the domain failure instead of retrying the message.
			if errors.Is(domainErr, ErrUserAlreadyExists) {
				return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
					GuildID:   guildID,
					UserID:    userID,
					TagNumber: tag,
					Reason:    domainErr.Error(),
				}), nil
			}
			return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
				GuildID:   guildID,
				UserID:    userID,
				TagNumber: tag,
				Reason:    domainErr.Error(),
			}), domainErr
		}
		dbDuration := time.Since(dbStart)
		s.metrics.RecordDBQueryDuration(ctx, dbDuration)

		// Step 3: Create guild membership
		membership := &userdb.GuildMembership{
			UserID:  userID,
			GuildID: guildID,
			Role:    sharedtypes.UserRoleUser,
		}

		if err := s.repo.CreateGuildMembership(ctx, membership); err != nil {
			s.logger.ErrorContext(ctx, "Failed to create guild membership",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordUserCreationFailure(ctx, userType, source)
			return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
				GuildID:   guildID,
				UserID:    userID,
				TagNumber: tag,
				Reason:    err.Error(),
			}), err
		}

		// Success - new user
		s.metrics.RecordUserCreationSuccess(ctx, userType, source)

		return results.SuccessResult(&userevents.UserCreatedPayloadV1{
			GuildID:         guildID,
			UserID:          userID,
			TagNumber:       tag,
			IsReturningUser: false,
		}), nil
	})

	s.metrics.RecordUserCreationDuration(ctx, userType, source, time.Since(startTime))

	return result, err
}

// translateDBError converts database-specific errors to domain errors
func translateDBError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Detect PostgreSQL unique constraint violations
	if strings.Contains(errMsg, "SQLSTATE 23505") ||
		strings.Contains(errMsg, "duplicate key value") {
		return ErrUserAlreadyExists
	}

	// Default case: return the original error
	return err
}

// createFailureResult is a helper to create standardized failure results
func createFailureResult(guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, err error) results.OperationResult {
	return results.FailureResult(&userevents.UserCreationFailedPayloadV1{
		GuildID:   guildID,
		UserID:    userID,
		TagNumber: tag,
		Reason:    err.Error(),
	})
}
