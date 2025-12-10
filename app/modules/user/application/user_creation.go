package userservice

import (
	"context"
	"errors"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// Common domain errors
var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidDiscordID  = errors.New("invalid Discord ID")
	ErrNegativeTagNumber = errors.New("tag number cannot be negative")
	ErrNilContext        = errors.New("context cannot be nil")
)

// normalizeUDiscName normalizes a UDisc name for matching (lowercase, trimmed).
func normalizeUDiscName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// CreateUser creates a user and returns a success or failure payload.
func (s *UserServiceImpl) CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (UserOperationResult, error) {
	if ctx == nil {
		return UserOperationResult{
			Error: ErrNilContext,
		}, ErrNilContext
	}

	// Handle empty Discord ID
	if userID == "" {
		return createFailureResult(guildID, userID, tag, ErrInvalidDiscordID), ErrInvalidDiscordID
	}

	// Handle negative tag numbers
	if tag != nil && *tag < 0 {
		return createFailureResult(guildID, userID, tag, ErrNegativeTagNumber), ErrNegativeTagNumber
	}

	startTime := time.Now()
	userType := "base"
	source := "user"

	if tag != nil {
		s.metrics.RecordUserCreationByTag(ctx, *tag)
	}
	s.metrics.RecordUserCreationAttempt(ctx, userType, source)

	result, err := s.serviceWrapper(ctx, "CreateUser", userID, func(ctx context.Context) (UserOperationResult, error) {
		user := userdb.User{
			UserID:  userID,
			GuildID: guildID,
		}

		// Normalize and set UDisc fields if provided
		if udiscUsername != nil && *udiscUsername != "" {
			normalized := normalizeUDiscName(*udiscUsername)
			user.UDiscUsername = &normalized
		}
		if udiscName != nil && *udiscName != "" {
			normalized := normalizeUDiscName(*udiscName)
			user.UDiscName = &normalized
		}

		dbStart := time.Now()
		err := s.UserDB.CreateUser(ctx, &user)
		dbDuration := time.Since(dbStart)
		s.metrics.RecordDBQueryDuration(ctx, dbDuration)

		if err != nil {
			// Standardize common database errors
			domainErr := translateDBError(err)
			logLevel := "error"

			// For expected errors like duplicate users, log as info or warn
			if errors.Is(domainErr, ErrUserAlreadyExists) {
				logLevel = "warn"
			}

			// Log with appropriate level
			if logLevel == "error" {
				s.logger.ErrorContext(ctx, "Failed to create user",
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
			// Return UserOperationResult with standardized failure
			return UserOperationResult{
				Failure: &userevents.UserCreationFailedPayload{
					GuildID:   guildID,
					UserID:    userID,
					TagNumber: tag,
					Reason:    domainErr.Error(),
				},
				Error: domainErr,
			}, domainErr
		}

		// Success
		s.metrics.RecordUserCreationSuccess(ctx, userType, source)

		return UserOperationResult{
			Success: &userevents.UserCreatedPayload{
				GuildID:   guildID,
				UserID:    userID,
				TagNumber: tag,
			},
		}, nil
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

	// Add more error translations as needed

	// Default case: return the original error
	return err
}

// createFailureResult is a helper to create standardized failure results
func createFailureResult(guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, err error) UserOperationResult {
	return UserOperationResult{
		Success: nil,
		Failure: &userevents.UserCreationFailedPayload{
			GuildID:   guildID,
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		},
		Error: err,
	}
}
