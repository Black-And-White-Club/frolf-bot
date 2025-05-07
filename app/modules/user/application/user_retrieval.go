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

// GetUser  retrieves user data and returns a response payload.
func (s *UserServiceImpl) GetUser(ctx context.Context, userID sharedtypes.DiscordID) (UserOperationResult, error) {
	operationName := "GetUser"
	result, err := s.serviceWrapper(ctx, operationName, userID, func(ctx context.Context) (UserOperationResult, error) {
		user, err := s.UserDB.GetUserByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, userdb.ErrUserNotFound) {
				s.logger.InfoContext(ctx, "User not found",
					attr.String("user_id", string(userID)),
				)
				s.metrics.RecordUserRetrievalFailure(ctx, userID)

				return UserOperationResult{
					Success: nil,
					Failure: &userevents.GetUserFailedPayload{
						UserID: userID,
						Reason: "user not found",
					},
					Error: errors.New("user not found"),
				}, errors.New("user not found")
			}

			s.logger.ErrorContext(ctx, "Failed to get user",
				attr.Error(err),
				attr.String("user_id", string(userID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)

			return UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserFailedPayload{
					UserID: userID,
					Reason: "failed to retrieve user from database",
				},
				Error: err,
			}, err
		}

		s.metrics.RecordUserRetrievalSuccess(ctx, userID)

		return UserOperationResult{
			Success: &userevents.GetUserResponsePayload{
				User: &usertypes.UserData{
					ID:     user.ID,
					UserID: user.UserID,
					Role:   user.Role,
				},
			},
			Failure: nil,
			Error:   nil,
		}, nil
	})

	return result, err
}

// GetUserRole retrieves a user's role and returns a response payload.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (UserOperationResult, error) {
	operationName := "GetUserRole"

	result, err := s.serviceWrapper(ctx, operationName, userID, func(ctx context.Context) (UserOperationResult, error) {
		role, err := s.UserDB.GetUserRole(ctx, userID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get user role",
				attr.Error(err),
				attr.String("userID", string(userID)),
			)
			s.metrics.RecordUserRetrievalFailure(ctx, userID)

			return UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: userID,
					Reason: "failed to retrieve user role",
				},
				Error: errors.New("failed to retrieve user role"),
			}, errors.New("failed to retrieve user role")
		}

		if !role.IsValid() {
			s.logger.ErrorContext(ctx, "Retrieved invalid role for user",
				attr.String("userID", string(userID)),
				attr.String("role", string(role)),
			)

			return UserOperationResult{
				Success: nil,
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: userID,
					Reason: "retrieved invalid user role",
				},
				Error: errors.New("invalid role in database"),
			}, errors.New("invalid role in database")
		}

		s.metrics.RecordUserRetrievalSuccess(ctx, userID)

		return UserOperationResult{
			Success: &userevents.GetUserRoleResponsePayload{
				UserID: userID,
				Role:   role,
			},
			Failure: nil,
			Error:   nil,
		}, nil
	})

	// Return the result and the wrapped error from the wrapper.
	return result, err
}
