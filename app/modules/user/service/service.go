package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB     userdb.UserDB
	Publisher  message.Publisher
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
}

// NewUserService creates a new UserService.
func NewUserService(publisher message.Publisher, subscriber message.Subscriber, db userdb.UserDB, logger watermill.LoggerAdapter) Service {
	return &UserServiceImpl{
		UserDB:     db,
		Publisher:  publisher,
		Subscriber: subscriber,
		logger:     logger,
	}
}

// OnUserSignupRequest processes a user signup request.
func (s *UserServiceImpl) OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequest) (*userevents.UserSignupResponse, error) {
	// 1. Check tag availability only if a tag number is provided
	var tagAvailable bool
	if req.TagNumber != 0 {
		var err error
		tagAvailable, err = s.checkTagAvailability(ctx, req.TagNumber)

		if err != nil {
			s.logger.Error("failed to check tag availability", err, watermill.LogFields{"error": err})
		}

	}

	// 2. Create the user
	newUser := &userdb.User{
		DiscordID: req.DiscordID,
		Role:      userdb.UserRoleRattler,
	}
	if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 3. Publish TagAssigned event only if a tag number is provided and available
	if req.TagNumber != 0 && tagAvailable {
		if err := s.publishTagAssigned(ctx, newUser.DiscordID, req.TagNumber); err != nil {
			return nil, fmt.Errorf("failed to publish TagAssigned event: %w", err)
		}
	}

	return &userevents.UserSignupResponse{
		Success: true,
	}, nil
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserServiceImpl) OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequest) (*userevents.UserRoleUpdateResponse, error) {
	if !req.NewRole.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", req.NewRole)
	}

	err := s.UserDB.UpdateUserRole(ctx, req.DiscordID, req.NewRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	// Publish UserRoleUpdated event (for other modules to consume)
	if err := s.publishUserRoleUpdated(req.DiscordID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	return &userevents.UserRoleUpdateResponse{
		Success: true,
	}, nil
}

// GetUserRole retrieves the role of a user.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, discordID string) (*userdb.UserRole, error) {
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}
	return &role, nil
}

// GetUser retrieves user data.
func (s *UserServiceImpl) GetUser(ctx context.Context, discordID string) (*userdb.User, error) {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// Define error variables at the package level
var (
	errTimeout          = errors.New("timeout waiting for tag availability response")
	errContextCancelled = errors.New("context cancelled while waiting for tag availability response")
	errSubscribe        = errors.New("failed to subscribe to reply: subscribe error")
)

func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	const waitForResponseTimeout = time.Second * 3

	msg := message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"tag_number":%d}`, tagNumber)))

	replySubject := fmt.Sprintf("%s.reply.%s", userevents.LeaderboardStream, msg.UUID)
	msg.Metadata.Set("Reply-To", replySubject)

	if err := s.Publisher.Publish(userevents.CheckTagAvailabilityRequestSubject, msg); err != nil {
		return false, fmt.Errorf("failed to publish request: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, waitForResponseTimeout)
	defer cancel()

	messages, err := s.Subscriber.Subscribe(ctxWithTimeout, replySubject)
	if err != nil {
		return false, fmt.Errorf("failed to subscribe to reply: %w", err)
	}

	select {
	case replyMsg, ok := <-messages:
		if !ok {
			return false, fmt.Errorf("reply channel closed unexpectedly")
		}
		defer replyMsg.Ack()

		var payload map[string]interface{}
		if err := json.Unmarshal(replyMsg.Payload, &payload); err != nil {
			return false, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		isAvailable, ok := payload["is_available"].(bool)
		if !ok {
			return false, fmt.Errorf("invalid response format: is_available field is not a boolean")
		}

		return isAvailable, nil

	case <-ctxWithTimeout.Done():
		if errors.Is(ctxWithTimeout.Err(), context.DeadlineExceeded) {
			return false, errTimeout
		}
		return false, errContextCancelled
	}
}

// publishTagAssigned publishes a TagAssigned event to the leaderboard module.
func (s *UserServiceImpl) publishTagAssigned(ctx context.Context, discordID string, tagNumber int) error {
	evt := userevents.TagAssigned{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}

	// Marshal the event data to JSON
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssigned: %w", err)
	}

	// Create a new message
	msg := message.NewMessage(watermill.NewUUID(), evtData)

	// Set the context of the message
	msg.SetContext(ctx)

	// Publish the message
	if err := s.Publisher.Publish(userevents.TagAssignedSubject, msg); err != nil {
		return fmt.Errorf("failed to publish TagAssigned: %w", err)
	}

	fmt.Printf("[DEBUG] publishTagAssigned: Published TagAssigned event for Discord ID: %s, Tag Number: %d\n", discordID, tagNumber)

	return nil
}

// publishUserRoleUpdated publishes a UserRoleUpdated event.
func (s *UserServiceImpl) publishUserRoleUpdated(discordID string, newRole userdb.UserRole) error {
	evt := userevents.UserRoleUpdated{
		DiscordID: discordID,
		NewRole:   newRole,
	}

	// Marshal the event data to JSON
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal UserRoleUpdated: %w", err)
	}

	// Create a new message
	msg := message.NewMessage(watermill.NewUUID(), evtData)

	// Publish the message
	if err := s.Publisher.Publish(userevents.UserRoleUpdatedSubject, msg); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdated: %w", err)
	}

	return nil
}
