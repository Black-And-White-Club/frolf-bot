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

// UpdateUserRoleInDatabase updates a user's role in the database and returns an operation result.
func (s *UserServiceImpl) UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (UserOperationResult, error) {
	operationName := "HandleUpdateUserRole"

	result, err := s.serviceWrapper(ctx, operationName, userID, func(ctx context.Context) (UserOperationResult, error) {
		if !newRole.IsValid() {
			validationErr := errors.New("invalid role")

			s.logger.ErrorContext(ctx, "Role validation failed",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
				attr.String("new_role", string(newRole)),
				attr.Error(validationErr),
			)

			s.metrics.RecordRoleUpdateFailure(ctx, userID, "validation_failed", newRole)

			return UserOperationResult{
				Failure: &userevents.UserRoleUpdateResultPayload{
					UserID:  userID,
					Role:    newRole, // Include the invalid role in the response
					Success: false,
					Error:   "invalid role",
				},
				Error: validationErr,
			}, nil
		}

		dbErr := s.UserDB.UpdateUserRole(ctx, userID, guildID, newRole)
		if dbErr != nil {
			s.logger.ErrorContext(ctx, "Failed to update userrole",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
				attr.String("new_role", string(newRole)),
				attr.Error(dbErr),
			)

			failureReason := "failed to update user role"
			if errors.Is(dbErr, userdb.ErrUserNotFound) {
				failureReason = "user not found"
			}

			s.metrics.RecordRoleUpdateFailure(ctx, userID, "database_error", newRole)

			return UserOperationResult{
				Failure: &userevents.UserRoleUpdateResultPayload{
					UserID:  userID,
					Role:    newRole,
					Success: false,
					Error:   failureReason,
				},
				Error: dbErr,
			}, fmt.Errorf("failed to update user role: %w", dbErr)
		}

		s.logger.InfoContext(ctx, "User role updated successfully",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
			attr.String("new_role", string(newRole)),
		)

		s.metrics.RecordRoleUpdateSuccess(ctx, userID, "database_success", newRole)

		return UserOperationResult{
			Success: &userevents.UserRoleUpdateResultPayload{
				UserID:  userID,
				Role:    newRole,
				Success: true,
				Error:   "",
			},
		}, nil
	})

	// Return the result and the wrapped error from the wrapper.
	return result, err
}
