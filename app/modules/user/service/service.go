package userservice

import (
	"context"
	"encoding/json"
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
	// 1. Check tag availability (communicate with leaderboard module)
	tagAvailable, err := s.checkTagAvailability(ctx, req.TagNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check tag availability: %w", err)
	}

	if !tagAvailable {
		return &userevents.UserSignupResponse{
			Success: false,
			Error:   fmt.Sprintf("tag number %d is already taken", req.TagNumber),
		}, nil
	}

	// 2. If tag is available, create the user
	newUser := &userdb.User{
		DiscordID: req.DiscordID,
		Role:      userdb.UserRoleRattler, // Default role
	}
	if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 3. Publish TagAssigned event to the leaderboard module using the context from the message
	if err := s.publishTagAssigned(ctx, newUser.DiscordID, req.TagNumber); err != nil {
		return nil, fmt.Errorf("failed to publish TagAssigned event: %w", err)
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

// checkTagAvailability checks if a tag number is available by querying the leaderboard module.
func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// 1. Prepare the request
	req := userevents.CheckTagAvailabilityRequest{
		TagNumber: tagNumber,
	}

	// 2. Marshal the request to JSON
	reqData, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequest: %w", err)
	}

	// 3. Use a unique correlation ID for this request
	correlationID := watermill.NewUUID()

	// 4. Create a new message with headers for correlation
	msg := message.NewMessage(watermill.NewUUID(), reqData)
	msg.Metadata.Set("correlation_id", correlationID)

	// Add a temporary delay for debugging (REMOVE THIS IN PRODUCTION)
	fmt.Println("[DEBUG] checkTagAvailability: Waiting for subscribers to be ready...")
	time.Sleep(2 * time.Second) // Wait for 2 seconds

	// 5. Publish the request
	fmt.Println("[DEBUG] checkTagAvailability: Publishing CheckTagAvailabilityRequest")
	if err := s.Publisher.Publish(userevents.CheckTagAvailabilityRequestSubject, msg); err != nil {
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}

	// 6. Subscribe to the response topic (using the string literal to avoid dependency)
	responseTopic := "leaderboard.check_tag_availability_response"

	fmt.Printf("[DEBUG] checkTagAvailability: Subscribing to response topic: %s\n", responseTopic)
	sub, err := s.Subscriber.Subscribe(ctx, responseTopic)
	if err != nil {
		return false, fmt.Errorf("failed to subscribe to CheckTagAvailabilityResponse: %w", err)
	}
	fmt.Printf("[DEBUG] checkTagAvailability: Subscribed to response topic: %s\n", responseTopic)

	// 7. Define a local struct to unmarshal the response (avoids dependency)
	type CheckTagAvailabilityResponse struct {
		IsAvailable bool `json:"is_available"`
		TagNumber   int  `json:"tag_number"`
	}

	// 8. Wait for the response with the matching correlation ID
	for msg := range sub {
		fmt.Printf("[DEBUG] checkTagAvailability: Received message on response topic. Correlation ID: %s\n", msg.Metadata.Get("correlation_id"))
		if msg.Metadata.Get("correlation_id") == correlationID {
			// 9. Unmarshal the response using the local struct
			var resp CheckTagAvailabilityResponse
			if err := json.Unmarshal(msg.Payload, &resp); err != nil {
				return false, fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
			}

			// 10. Acknowledge the message
			msg.Ack()

			// 11. Return the result
			fmt.Printf("[DEBUG] checkTagAvailability: Returning IsAvailable: %t\n", resp.IsAvailable)
			return resp.IsAvailable, nil
		}
	}

	return false, fmt.Errorf("no response received for CheckTagAvailabilityRequest")
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
