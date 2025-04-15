package userservice

import (
	"context"
	"errors"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// UpdateUserRoleInDatabase updates the user's role in the database.
func (s *UserServiceImpl) UpdateUserRoleInDatabase(ctx context.Context, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (*userevents.UserRoleUpdateResultPayload, *userevents.UserRoleUpdateFailedPayload, error) {
	operationName := "HandleUpdateUserRole"

	result, err := s.serviceWrapper(ctx, operationName, userID, func(ctx context.Context) (UserOperationResult, error) {
		// Validate the new role
		if !newRole.IsValid() {
			err := errors.New("invalid role")

			// Log the validation failure
			s.logger.ErrorContext(ctx, "Role validation failed",
				attr.String("user_id", string(userID)),
				attr.String("new_role", string(newRole)),
				attr.Error(err),
			)

			// Record the validation failure in metrics
			s.metrics.RecordRoleUpdateFailure(ctx, userID, "TODO", newRole)

			return UserOperationResult{
				Failure: &userevents.UserRoleUpdateFailedPayload{
					UserID: userID,
					Reason: "invalid role",
				},
			}, err
		}

		// Update the user's role in the database
		err := s.UserDB.UpdateUserRole(ctx, userID, newRole)
		if err != nil {
			// Log the database error
			s.logger.ErrorContext(ctx, "Failed to update userrole",
				attr.String("user_id", string(userID)),
				attr.String("new_role", string(newRole)),
				attr.Error(err),
			)

			// Record the database error in metrics
			s.metrics.RecordRoleUpdateFailure(ctx, userID, "TODO", newRole)

			if errors.Is(err, userdb.ErrUserNotFound) {
				return UserOperationResult{
					Failure: &userevents.UserRoleUpdateFailedPayload{
						UserID: userID,
						Reason: "user not found",
					},
				}, errors.New("user not found")
			}

			return UserOperationResult{
				Failure: &userevents.UserRoleUpdateFailedPayload{
					UserID: userID,
					Reason: "failed to update user role",
				},
			}, fmt.Errorf("failed to update user role: %w", err)
		}

		// Log the successful role update
		s.logger.InfoContext(ctx, "User role updated successfully",
			attr.String("user_id", string(userID)),
			attr.String("new_role", string(newRole)),
		)

		// Record the operation success in metrics
		s.metrics.RecordRoleUpdateSuccess(ctx, userID, "TODO", newRole)

		// Return success payload
		return UserOperationResult{
			Success: &userevents.UserRoleUpdateResultPayload{
				UserID: userID,
				Role:   newRole,
			},
		}, nil
	})
	if err != nil {
		// Return failure payload if present
		if result.Failure != nil {
			return nil, result.Failure.(*userevents.UserRoleUpdateFailedPayload), err
		}

		// Otherwise, return a generic failure response
		return nil, &userevents.UserRoleUpdateFailedPayload{
			UserID: userID,
			Reason: err.Error(),
		}, err
	}

	// Return success payload
	return result.Success.(*userevents.UserRoleUpdateResultPayload), nil, nil
}
