package userservice

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
)

// GetUser retrieves user data and returns a response payload.
func (s *UserServiceImpl) GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserOperationResult, error) {
	if userID == "" {
		s.logger.WarnContext(ctx, "Attempted to get user with empty Discord ID")
		return UserOperationResult{}, errors.New("GetUser: Discord ID cannot be empty")
	}

	operationName := "GetUser"

	result, err := s.serviceWrapper(ctx, operationName, userID, func(ctx context.Context) (UserOperationResult, error) {
		user, dbErr := s.UserDB.GetUserByUserID(ctx, userID, guildID)
		if dbErr != nil {
			return UserOperationResult{}, dbErr
		}

		return UserOperationResult{
			Success: &userevents.GetUserResponsePayloadV1{
				GuildID: guildID,
				User: &usertypes.UserData{
					ID:     user.ID,
					UserID: user.UserID,
					Role:   user.Role,
				},
			},
		}, nil
	})
	if err != nil {
		if errors.Is(err, userdb.ErrUserNotFound) {
			s.logger.InfoContext(ctx, "User not found",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)

			return UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayloadV1{
					GuildID: guildID,
					UserID:  userID,
					Reason:  "user not found",
				},
				Error: nil,
			}, nil
		}

		s.logger.ErrorContext(ctx, "Failed to get user due to technical error",
			attr.Error(err),
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return UserOperationResult{
			Success: nil,
			Failure: &userevents.GetUserFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "failed to retrieve user from database",
			},
			Error: err,
		}, err
	}

	s.metrics.RecordUserRetrievalSuccess(ctx, userID)

	return result, nil
}

func (s *UserServiceImpl) GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserOperationResult, error) {
	operationName := "GetUserRole"

	innerOp := func(ctx context.Context) (UserOperationResult, error) {
		role, dbErr := s.UserDB.GetUserRole(ctx, userID, guildID)
		if dbErr != nil {
			s.logger.ErrorContext(ctx, "Failed to get user role from DB",
				attr.Error(dbErr),
				attr.String("userID", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			return UserOperationResult{}, dbErr
		}

		return UserOperationResult{
			Success: &userevents.GetUserRoleResponsePayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Role:    role,
			},
		}, nil
	}

	result, err := s.serviceWrapper(ctx, operationName, userID, innerOp)
	if err != nil {
		if errors.Is(err, userdb.ErrUserNotFound) {
			s.logger.InfoContext(ctx, "User not found for role lookup",
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)

			return UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayloadV1{
					GuildID: guildID,
					UserID:  userID,
					Reason:  "user not found",
				},
				Error: nil,
			}, nil
		}

		s.logger.ErrorContext(ctx, "Technical error during GetUserRole operation",
			attr.Error(err),
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return UserOperationResult{
			Success: nil,
			Failure: &userevents.GetUserRoleFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "failed to retrieve user role from database",
			},
			Error: err,
		}, err
	}

	if result.Success == nil {
		s.logger.ErrorContext(ctx, "serviceWrapper returned nil error but result.Success is nil for GetUserRole",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		internalErr := errors.New("internal service error: unexpected nil success payload")
		return UserOperationResult{
			Failure: &userevents.GetUserRoleFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "internal service error",
			},
			Error: internalErr,
		}, internalErr
	}

	successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayloadV1)
	if !ok {
		s.logger.ErrorContext(ctx, "serviceWrapper returned nil error but result.Success has unexpected type for GetUserRole",
			attr.String("user_id", string(userID)),
			attr.String("guild_id", string(guildID)),
		)
		internalErr := errors.New("internal service error: unexpected success payload type")
		return UserOperationResult{
			Failure: &userevents.GetUserRoleFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "internal service error",
			},
			Error: internalErr,
		}, internalErr
	}

	if !successPayload.Role.IsValid() {
		s.logger.ErrorContext(ctx, "Retrieved invalid role for user",
			attr.String("userID", string(userID)),
			attr.String("guild_id", string(guildID)),
			attr.String("role", string(successPayload.Role)),
		)
		s.metrics.RecordUserRetrievalFailure(ctx, userID)

		return UserOperationResult{
			Success: nil,
			Failure: &userevents.GetUserRoleFailedPayloadV1{
				GuildID: guildID,
				UserID:  userID,
				Reason:  "user found but has invalid role",
			},
			Error: nil,
		}, nil
	}

	s.metrics.RecordUserRetrievalSuccess(ctx, userID)

	return result, nil
}
