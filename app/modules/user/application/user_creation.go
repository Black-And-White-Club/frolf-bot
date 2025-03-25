package userservice

import (
	"context"
	"errors"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CreateUser creates a user and returns a success or failure payload.
func (s *UserServiceImpl) CreateUser(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, tag *int) (*userevents.UserCreatedPayload, *userevents.UserCreationFailedPayload, error) {
	//  Handle nil context
	if ctx == nil {
		return nil, nil, errors.New("context cannot be nil")
	}

	//  Handle empty Discord ID
	if userID == "" {
		err := errors.New("invalid Discord ID")
		return nil, &userevents.UserCreationFailedPayload{
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		}, err
	}

	//  Handle negative tag numbers
	if tag != nil && *tag < 0 {
		err := errors.New("tag number cannot be negative")
		return nil, &userevents.UserCreationFailedPayload{
			UserID:    userID,
			TagNumber: tag,
			Reason:    err.Error(),
		}, err
	}

	startTime := time.Now()

	// Record user creation attempt
	if tag != nil {
		s.metrics.UserCreationByTag(*tag)
	}

	userType := "base"
	standardRole := usertypes.UserRoleRattler.String()
	source := msg.Metadata.Get("source")
	if source == "" {
		source = "user"
	}

	s.metrics.RecordUserCreation(userType, source, "attempted")
	s.metrics.RecordUserRoleUpdateAttempt(userID, standardRole)

	result, err := s.serviceWrapper(msg, "CreateUser", userID, func() (UserOperationResult, error) {
		ctx, span := s.tracer.StartSpan(ctx, "CreateUser.DatabaseOperation", msg)
		defer span.End()

		user := userdb.User{UserID: userID}

		// Time the database operation
		dbStart := time.Now()

		err := s.UserDB.CreateUser(ctx, &user)
		dbDuration := time.Since(dbStart).Seconds()
		s.metrics.DBQueryDuration(dbDuration)

		if err != nil {
			s.logger.Error("Database error during user creation",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Error(err),
				attr.String("db_operation", "insert"),
			)

			s.metrics.RecordUserCreation(userType, source, "failed")
			s.metrics.RecordUserRoleUpdateFailure(userID, standardRole)

			return UserOperationResult{
				Failure: &userevents.UserCreationFailedPayload{
					UserID:    userID,
					TagNumber: tag,
					Reason:    err.Error(),
				},
			}, err
		}

		s.metrics.RecordUserCreation(userType, source, "success")
		s.metrics.RecordUserRoleUpdateSuccess(userID, standardRole)

		if tag != nil {
			s.metrics.RecordTagAvailabilityCheck(true, *tag)
		}

		return UserOperationResult{
			Success: &userevents.UserCreatedPayload{
				UserID:    userID,
				TagNumber: tag,
			},
		}, nil
	})

	// Record total user creation duration
	s.metrics.UserCreationDuration(time.Since(startTime).Seconds())

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

	s.logger.Info("User successfully created",
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)),
		attr.Float64("creation_duration_seconds", time.Since(startTime).Seconds()),
		attr.String("user_type", userType),
		attr.String("source", source),
	)

	return result.Success.(*userevents.UserCreatedPayload), nil, nil
}
