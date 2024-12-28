package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	"github.com/Black-And-White-Club/tcr-bot/app/events"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB   userdb.UserDB
	eventBus events.EventBus
	logger   types.LoggerAdapter
}

// NewUserService creates a new UserService.
func NewUserService(db userdb.UserDB, eventBus events.EventBus, logger types.LoggerAdapter) *UserServiceImpl {
	return &UserServiceImpl{
		UserDB:   db,
		eventBus: eventBus,
		logger:   logger,
	}
}

// OnUserSignupRequest processes a user signup request.
func (s *UserServiceImpl) OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequestPayload) (*userevents.UserSignupResponsePayload, error) {
	s.logger.Info("OnUserSignupRequest started", types.LogFields{
		"discord_id": req.DiscordID,
		"tag_number": req.TagNumber,
		"contextErr": ctx.Err(),
	})

	var tagAvailable bool
	var err error

	if req.TagNumber != 0 {
		s.logger.Debug("Before checkTagAvailability", types.LogFields{"contextErr": ctx.Err()})

		tagAvailable, err = s.checkTagAvailability(ctx, req.TagNumber)
		if err != nil {
			s.logger.Error("failed to check tag availability", err, types.LogFields{"error": err})
			return nil, fmt.Errorf("failed to check tag availability: %w", err)
		}
	}

	// Only create user if no error from checkTagAvailability
	if err == nil {
		newUser := &userdb.User{
			DiscordID: req.DiscordID,
			Role:      usertypes.UserRoleRattler,
		}
		if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		if req.TagNumber != 0 && tagAvailable {
			if err := s.publishTagAssigned(ctx, newUser.DiscordID, req.TagNumber); err != nil {
				s.logger.Error("failed to publish TagAssignedRequest event", err, types.LogFields{"error": err})
				return nil, fmt.Errorf("failed to publish TagAssignedRequest event: %w", err)
			}
		}
	}

	return &userevents.UserSignupResponsePayload{
		Success: true,
	}, nil
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserServiceImpl) OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequestPayload) (*userevents.UserRoleUpdateResponsePayload, error) { // Updated struct names
	if !req.NewRole.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", req.NewRole)
	}

	err := s.UserDB.UpdateUserRole(ctx, req.DiscordID, req.NewRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	if err := s.publishUserRoleUpdated(ctx, req.DiscordID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	return &userevents.UserRoleUpdateResponsePayload{ // Updated struct name
		Success: true,
	}, nil
}

// GetUser Role retrieves the role of a user.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error) {
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		return usertypes.UserRoleUnknown, fmt.Errorf("failed to get user role: %w", err) // Return the zero value
	}
	return role, nil
}

// GetUser  retrieves user data by Discord ID.
func (s *UserServiceImpl) GetUser(ctx context.Context, discordID usertypes.DiscordID) (usertypes.User, error) {
	// Retrieve the user from the database
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// Define error variables at the package level
var (
	errTimeout          = errors.New("timeout waiting for response")
	errContextCancelled = errors.New("context cancelled")
)

func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Info("checkTagAvailability started", types.LogFields{
		"tag_number": tagNumber,
		"contextErr": ctx.Err(),
	})

	const defaultWaitForResponseTimeout = time.Second * 3 // Default timeout for production

	// Create a new context for timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultWaitForResponseTimeout)
	defer cancel()

	// Generate a reply subject
	replySubject := fmt.Sprintf("%s.reply.%s", userevents.CheckTagAvailabilityRequest, types.NewUUID())
	s.logger.Debug("Generated replySubject", types.LogFields{
		"replySubject": replySubject,
	})

	// Publish CheckTagAvailabilityRequest before subscribing
	requestPayload := []byte(fmt.Sprintf(`{"tag_number":%d}`, tagNumber)) // Correctly format as integer
	requestMsg := adapters.NewWatermillMessageAdapter(types.NewUUID(), requestPayload)
	requestMsg.SetMetadata("replyTo", replySubject)

	// Add debugging print statements here:
	fmt.Printf("Generated requestPayload: %s\n", requestPayload)

	if err := s.eventBus.Publish(ctxWithTimeout, userevents.CheckTagAvailabilityRequest, requestMsg); err != nil {
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}

	// Use a buffered channel to receive responses
	responseChan := make(chan bool, 1)

	// Subscribe to the reply subject
	err := s.eventBus.Subscribe(ctxWithTimeout, replySubject, func(ctx context.Context, msg types.Message) error {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		isAvailable, ok := payload["is_available"].(bool)
		if !ok {
			return fmt.Errorf("invalid response format: is_available field is not a boolean")
		}

		select {
		case responseChan <- isAvailable:
			// Response sent successfully
		default:
			// Log if response channel is full
			s.logger.Error("Response channel full", nil, types.LogFields{"replySubject": replySubject})
		}
		return nil
	})

	if err != nil {
		// Check if the error is errTimeout and return it directly
		if errors.Is(err, errTimeout) {
			return false, errTimeout
		}
		return false, fmt.Errorf("failed to subscribe to reply: %w", err)
	}

	// Wait for the response or timeout
	select {
	case isAvailable := <-responseChan:
		return isAvailable, nil
	case <-ctxWithTimeout.Done():
		if errors.Is(ctxWithTimeout.Err(), context.DeadlineExceeded) {
			return false, errTimeout
		}
		return false, errContextCancelled
	}
}

func (s *UserServiceImpl) publishTagAssigned(ctx context.Context, discordID usertypes.DiscordID, tagNumber int) error {
	s.logger.Info("publishTagAssigned called", types.LogFields{
		"discord_id": discordID,
		"tag_number": tagNumber,
		"contextErr": ctx.Err(),
	})

	msgPayload := userevents.TagAssignedRequestPayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}

	payloadBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssignedRequest payload: %w", err)
	}

	msg := adapters.NewWatermillMessageAdapter(types.NewUUID(), payloadBytes)

	if err := s.eventBus.Publish(ctx, userevents.TagAssignedRequest, msg); err != nil {
		return fmt.Errorf("failed to publish TagAssignedRequest event: %w", err)
	}

	return nil
}

// publishUserRoleUpdated publishes a UserRoleUpdated event.
func (s *UserServiceImpl) publishUserRoleUpdated(ctx context.Context, discordID usertypes.DiscordID, newRole usertypes.UserRoleEnum) error {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return fmt.Errorf("failed to get user for publishing UserRoleUpdated: %w", err)
	}

	evt := userevents.UserRoleUpdatedPayload{
		DiscordID: user.GetDiscordID(),
		NewRole:   newRole.String(),
	}

	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal UserRoleUpdatedPayload: %w", err) // Updated struct name
	}

	msg := adapters.NewWatermillMessageAdapter(types.NewUUID(), evtData)

	// Use the UserRoleUpdated EventType
	if err := s.eventBus.Publish(ctx, userevents.UserRoleUpdated, msg); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdated: %w", err)
	}

	return nil
}
