package userservice

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetUser retrieves user data and returns a response payload.
func (s *UserServiceImpl) GetUser(ctx context.Context, msg *message.Message, userID usertypes.DiscordID) (*userevents.GetUserResponsePayload, *userevents.GetUserFailedPayload, error) {
	operationName := "GetUser"
	result, err := s.serviceWrapper(msg, operationName, userID, func() (UserOperationResult, error) {
		user, err := s.UserDB.GetUserByUserID(ctx, userID)
		if err != nil {
			if errors.Is(err, userdb.ErrUserNotFound) {
				s.logger.Info("User not found",
					attr.String("user_id", string(userID)),
					attr.CorrelationIDFromMsg(msg),
				)
				s.metrics.RecordUserRetrieval(false, userID)

				return UserOperationResult{
					Failure: &userevents.GetUserFailedPayload{
						UserID: userID,
						Reason: "user not found",
					},
				}, errors.New("user not found") // Return an error message
			}

			s.logger.Error("Failed to get user",
				attr.Error(err),
				attr.String("user_id", string(userID)),
				attr.CorrelationIDFromMsg(msg),
			)
			s.metrics.RecordUserRetrieval(false, userID)

			return UserOperationResult{
				Failure: &userevents.GetUserFailedPayload{
					UserID: userID,
					Reason: "failed to retrieve user from database",
				},
				Error: err,
			}, err
		}

		s.metrics.RecordUserRetrieval(true, userID)

		return UserOperationResult{
			Success: &userevents.GetUserResponsePayload{
				User: &usertypes.UserData{
					ID:     user.ID,
					UserID: user.UserID,
					Role:   user.Role,
				},
			},
		}, nil
	})

	if err != nil {
		// Return failure payload if present
		if result.Failure != nil {
			return nil, result.Failure.(*userevents.GetUserFailedPayload), err
		}

		// Otherwise, return a generic failure response
		return nil, &userevents.GetUserFailedPayload{
			UserID: userID,
			Reason: err.Error(),
		}, err
	}

	// Return success payload
	return result.Success.(*userevents.GetUserResponsePayload), nil, nil
}

// GetUserRole retrieves a user's role and returns a response payload.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, msg *message.Message, userID usertypes.DiscordID) (*userevents.GetUserRoleResponsePayload, *userevents.GetUserRoleFailedPayload, error) {
	operationName := "GetUserRole"

	result, err := s.serviceWrapper(msg, operationName, userID, func() (UserOperationResult, error) {
		role, err := s.UserDB.GetUserRole(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get user role",
				attr.Error(err),
				attr.String("userID", string(userID)),
				attr.CorrelationIDFromMsg(msg),
			)
			s.metrics.RecordUserRoleRetrieval(false, userID)

			return UserOperationResult{
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: userID,
					Reason: "failed to retrieve user role",
				},
				Error: err,
			}, errors.New("failed to retrieve user role")
		}

		// Ensure role is valid before returning
		if !role.IsValid() {
			s.logger.Error("Retrieved invalid role for user",
				attr.String("userID", string(userID)),
				attr.String("role", string(role)),
			)

			return UserOperationResult{
				Failure: &userevents.GetUserRoleFailedPayload{
					UserID: userID,
					Reason: "retrieved invalid user role",
				},
				Error: errors.New("invalid role in database"),
			}, errors.New("invalid role in database")
		}

		s.metrics.RecordUserRoleRetrieval(true, userID)

		return UserOperationResult{
			Success: &userevents.GetUserRoleResponsePayload{
				UserID: userID,
				Role:   role,
			},
		}, nil
	})

	if err != nil {
		// Return failure payload if present
		if result.Failure != nil {
			return nil, result.Failure.(*userevents.GetUserRoleFailedPayload), err
		}

		// Otherwise, return a generic failure response
		return nil, &userevents.GetUserRoleFailedPayload{
			UserID: userID,
			Reason: err.Error(),
		}, err
	}

	// Return success payload
	return result.Success.(*userevents.GetUserRoleResponsePayload), nil, nil
}
