package userservice

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// UpdateUserRoleInDatabase updates a user's role in the database and returns an operation result.
func (s *UserService) UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult, error) {
	operationName := "HandleUpdateUserRole"

	result, err := s.withTelemetry(ctx, operationName, userID, func(ctx context.Context) (results.OperationResult, error) {
		if !newRole.IsValid() {
			validationErr := errors.New("invalid role")

			s.logger.ErrorContext(ctx, "Role validation failed",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
				attr.String("new_role", string(newRole)),
				attr.Error(validationErr),
			)

			s.metrics.RecordRoleUpdateFailure(ctx, userID, "validation_failed", newRole)
			return results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Role:    newRole, // Include the invalid role in the response
				Success: false,
				Reason:  "invalid role",
			}), nil
		}

		dbErr := s.repo.UpdateUserRole(ctx, userID, guildID, newRole)
		if dbErr != nil {
			// If user not found, return domain failure (nil error) so caller can publish failure
			if errors.Is(dbErr, userdb.ErrNoRowsAffected) || errors.Is(dbErr, userdb.ErrNotFound) {
				s.metrics.RecordRoleUpdateFailure(ctx, userID, "not_found", newRole)
				return results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{
					GuildID: guildID,
					UserID:  userID,
					Role:    newRole,
					Success: false,
					Reason:  "user not found",
				}), nil
			}

			s.logger.ErrorContext(ctx, "Failed to update userrole",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
				attr.String("new_role", string(newRole)),
				attr.Error(dbErr),
			)

			s.metrics.RecordRoleUpdateFailure(ctx, userID, "database_error", newRole)
			// Return failure payload but do not propagate top-level error so caller can publish
			return results.FailureResult(&userevents.UserRoleUpdateResultPayloadV1{GuildID: guildID, UserID: userID, Role: newRole, Success: false, Reason: "failed to update user role"}), nil
		}

		s.logger.InfoContext(ctx, "User role updated successfully",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
			attr.String("new_role", string(newRole)),
		)

		s.metrics.RecordRoleUpdateSuccess(ctx, userID, "database_success", newRole)

		return results.SuccessResult(&userevents.UserRoleUpdateResultPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Role:    newRole,
			Success: true,
			Reason:  "",
		}), nil
	})

	// Return the result and the wrapped error from the wrapper.
	return result, err
}
