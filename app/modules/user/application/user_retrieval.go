package userservice

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// GetUser retrieves user data and returns a response payload.
func (s *UserService) GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult, error) {
	if userID == "" {
		s.logger.WarnContext(ctx, "Attempted to get user with empty Discord ID")
		return results.FailureResult(&userevents.GetUserFailedPayloadV1{GuildID: guildID, UserID: userID, Reason: "GetUser: Discord ID cannot be empty"}), errors.New("GetUser: Discord ID cannot be empty")
	}

	operationName := "GetUser"

	result, err := s.withTelemetry(ctx, operationName, userID, func(ctx context.Context) (results.OperationResult, error) {
		user, dbErr := s.repo.GetUserByUserID(ctx, userID, guildID)
		if dbErr != nil {
			// If record not found, return domain failure with nil error so caller can publish
			if errors.Is(dbErr, userdb.ErrNotFound) {
				s.logger.InfoContext(ctx, "User not found in DB (GetUser inner op)",
					attr.String("user_id", string(userID)),
					attr.String("guild_id", string(guildID)),
				)
				s.metrics.RecordUserRetrievalFailure(ctx, userID)
				return results.FailureResult(&userevents.GetUserFailedPayloadV1{
					GuildID: guildID,
					UserID:  userID,
					Reason:  "user not found",
				}), nil
			}

			// Technical DB error -> return failure payload but do not propagate top-level error
			s.logger.ErrorContext(ctx, "Failed to get user from DB",
				attr.Error(dbErr),
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)
			return results.FailureResult(&userevents.GetUserFailedPayloadV1{GuildID: guildID, UserID: userID, Reason: "failed to retrieve user from database"}), nil
		}

		if user == nil {
			// Domain-level not-found -> return failure payload with nil error so caller can publish
			s.logger.InfoContext(ctx, "User not found in DB (GetUser inner op)",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)
			return results.FailureResult(&userevents.GetUserFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "user not found",
			}), nil
		}

		return results.SuccessResult(&userevents.GetUserResponsePayloadV1{
			GuildID: guildID,
			User: &usertypes.UserData{
				ID:     user.User.ID,
				UserID: user.User.UserID,
				Role:   user.Role,
			},
		}), nil
	})
	if err != nil {
		// Technical error from inner operation
		s.logger.ErrorContext(ctx, "Failed to get user due to technical error",
			attr.Error(err),
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return results.FailureResult(&userevents.GetUserFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "failed to retrieve user from database",
		}), err
	}

	// If wrapper returned nil error but result indicates domain failure, propagate it
	// Only treat as internal error if BOTH Success and Failure are nil (unexpected)
	if result.Success == nil && result.Failure == nil {
		return results.FailureResult(&userevents.GetUserFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "internal service error",
		}), errors.New("internal service error: unexpected nil success payload")
	}

	s.metrics.RecordUserRetrievalSuccess(ctx, userID)
	return result, nil
}

func (s *UserService) GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult, error) {
	operationName := "GetUserRole"

	innerOp := func(ctx context.Context) (results.OperationResult, error) {
		role, dbErr := s.repo.GetUserRole(ctx, userID, guildID)
		if dbErr != nil {
			// If not found, return domain failure so caller can publish a not-found event
			if errors.Is(dbErr, userdb.ErrNotFound) {
				s.logger.InfoContext(ctx, "User role not found in DB",
					attr.String("user_id", string(userID)),
					attr.String("guild_id", string(guildID)),
				)
				return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
					GuildID: guildID,
					UserID:  userID,
					Reason:  "user not found",
				}), nil
			}

			s.logger.ErrorContext(ctx, "Failed to get user role from DB",
				attr.Error(dbErr),
				attr.String("userID", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			// Return failure payload but do not propagate top-level error
			return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{GuildID: guildID, UserID: userID, Reason: "failed to retrieve user role from database"}), nil
		}

		return results.SuccessResult(&userevents.GetUserRoleResponsePayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Role:    role,
		}), nil
	}

	result, err := s.withTelemetry(ctx, operationName, userID, innerOp)
	if err != nil {
		s.logger.ErrorContext(ctx, "Technical error during GetUserRole operation",
			attr.Error(err),
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "failed to retrieve user role from database",
		}), err
	}

	// If wrapper returned nil error but result indicates domain failure, propagate it
	// Only treat as internal error if BOTH Success and Failure are nil (unexpected)
	if result.Success == nil && result.Failure == nil {
		s.logger.ErrorContext(ctx, "serviceWrapper returned nil error and no payload for GetUserRole",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		internalErr := errors.New("internal service error: unexpected nil success payload")
		// Return nil error so handler publishes failure event
		return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "internal service error",
		}), internalErr
	}

	// If operation returned a failure payload, propagate it to the caller (no top-level error)
	if result.Failure != nil {
		return result, nil
	}

	successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayloadV1)
	if !ok {
		s.logger.ErrorContext(ctx, "serviceWrapper returned nil error but result.Success has unexpected type for GetUserRole",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		internalErr := errors.New("internal service error: unexpected success payload type")
		// Return nil error so handler publishes failure event
		return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "internal service error",
		}), internalErr
	}

	if !successPayload.Role.IsValid() {
		s.logger.ErrorContext(ctx, "Retrieved invalid role for user",
			attr.String("userID", string(userID)),
			attr.String("guild_id", string(guildID)),
			attr.String("role", string(successPayload.Role)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return results.FailureResult(&userevents.GetUserRoleFailedPayloadV1{
			GuildID: guildID,
			UserID:  userID,
			Reason:  "user found but has invalid role",
		}), nil
	}

	s.metrics.RecordUserRetrievalSuccess(ctx, userID)

	return result, nil
}
