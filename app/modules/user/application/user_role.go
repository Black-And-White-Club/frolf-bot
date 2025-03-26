package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UpdateUserRoleInDatabase updates the user's role in the database.
func (s *UserServiceImpl) UpdateUserRoleInDatabase(ctx context.Context, msg *message.Message, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (*userevents.UserRoleUpdateResultPayload, *userevents.UserRoleUpdateFailedPayload, error) {
	operationName := "UpdateUserRole"
	// Start a new span for the operation
	ctx, span := s.tracer.StartSpan(ctx, operationName, msg)
	defer span.End()

	// Log the start of the operation
	s.logger.Info("Starting "+operationName,
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)),
		attr.String("new_role", string(newRole)),
	)

	// Record the operation attempt
	s.metrics.RecordOperationAttempt(operationName, userID)

	// Measure the duration of the operation
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.RecordOperationDuration(operationName, duration)
	}()

	// Validate the new role
	if !newRole.IsValid() {
		err := errors.New("invalid role")

		// Log the validation failure
		s.logger.Error("Role validation failed",
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(userID)),
			attr.String("new_role", string(newRole)),
			attr.Error(err),
		)

		// Record the validation failure in metrics
		s.metrics.RecordOperationFailure(operationName, userID)

		// Record the error in the span
		span.RecordError(err)

		return nil, &userevents.UserRoleUpdateFailedPayload{
			UserID: userID,
			Reason: "invalid role",
		}, err
	}

	// Update the user's role in the database
	err := s.UserDB.UpdateUserRole(ctx, userID, newRole)
	if err != nil {
		// Log the database error
		s.logger.Error("Failed to update user role",
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(userID)),
			attr.String("new_role", string(newRole)),
			attr.Error(err),
		)

		// Record the database error in metrics
		s.metrics.RecordOperationFailure(operationName, userID)

		// Record the error in the span
		span.RecordError(err)

		if errors.Is(err, userdb.ErrUserNotFound) {
			return nil, &userevents.UserRoleUpdateFailedPayload{
				UserID: userID,
				Reason: "user not found",
			}, errors.New("user not found")
		}

		return nil, &userevents.UserRoleUpdateFailedPayload{
			UserID: userID,
			Reason: "failed to update user role",
		}, fmt.Errorf("failed to update user role: %w", err)
	}

	// Log the successful role update
	s.logger.Info("User role updated successfully",
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)),
		attr.String("new_role", string(newRole)),
	)

	// Record the operation success in metrics
	s.metrics.RecordOperationSuccess(operationName, userID)

	// Return success payload
	return &userevents.UserRoleUpdateResultPayload{
		UserID: userID,
		Role:   newRole,
	}, nil, nil
}
