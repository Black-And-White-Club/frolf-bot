package userservice

import (
	"context"
	"errors"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// CreateUser  creates a user and returns a success or failure payload.
func (s *UserServiceImpl) CreateUser(ctx context.Context, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber) (*userevents.UserCreatedPayload, *userevents.UserCreationFailedPayload, error) {
	// Handle nil context
	if ctx == nil {
		return nil, nil, errors.New("context cannot be nil")
	}

	// Handle empty Discord ID
	if userID == "" {
		err := errors.New("invalid Discord ID")
		return nil, &userevents.UserCreationFailedPayload{
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		}, err
	}

	// Handle negative tag numbers
	if tag != nil && *tag < 0 {
		err := errors.New("tag number cannot be negative")
		return nil, &userevents.UserCreationFailedPayload{
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		}, err
	}

	startTime := time.Now()

	// Record usercreation attempt
	if tag != nil {
		s.metrics.RecordUserCreationByTag(ctx, *tag)
	}

	userType := "base"
	source := "user"

	s.metrics.RecordUserCreationAttempt(ctx, userType, source)

	result, err := s.serviceWrapper(ctx, "CreateUser ", userID, func(ctx context.Context) (UserOperationResult, error) {
		user := userdb.User{UserID: userID}

		// Time the database operation
		dbStart := time.Now()

		err := s.UserDB.CreateUser(ctx, &user)
		dbDuration := time.Duration(time.Since(dbStart).Seconds())
		s.metrics.RecordDBQueryDuration(ctx, dbDuration)

		if err != nil {
			s.logger.ErrorContext(ctx, "Database error during user creation",
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)

			s.metrics.RecordUserCreationFailure(ctx, userType, source)

			return UserOperationResult{
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    userID,
					TagNumber: tag,
					Reason:    err.Error(),
				},
			}, err
		}

		s.metrics.RecordUserCreationSuccess(ctx, userType, source)

		return UserOperationResult{
			Success: &userevents.UserCreatedPayload{
				UserID:    userID,
				TagNumber: tag,
			},
		}, nil
	})

	// Record total usercreation duration
	s.metrics.RecordUserCreationDuration(ctx, userType, source, time.Duration(time.Since(startTime).Seconds()))

	if err != nil {
		if result.Failure != nil {
			return nil, result.Failure.(*userevents.UserCreationFailedPayload), err
		}
		return nil, &userevents.UserCreationFailedPayload{
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		}, err
	}

	s.logger.InfoContext(ctx, "User  successfully created",
		attr.String("user_id", string(userID)),
		attr.Float64("creation_duration_seconds", time.Since(startTime).Seconds()),
		attr.String("user_type", userType),
		attr.String("source", source),
	)

	return result.Success.(*userevents.UserCreatedPayload), nil, nil
}
