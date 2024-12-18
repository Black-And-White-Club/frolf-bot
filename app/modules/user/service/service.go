package userservice

import (
	"context"
	"encoding/json"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/nats-io/nats.go"
)

// UserService handles user-related logic.
type UserService struct {
	UserDB    userdb.UserDB
	JS        nats.JetStreamContext
	Publisher nats.JetStreamContext
}

// OnUserSignupRequest processes a user signup request.
func (s *UserService) OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequest) (*userevents.UserSignupResponse, error) {
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

	// 3. Publish TagAssigned event to the leaderboard module
	if err := s.publishTagAssigned(newUser.DiscordID, req.TagNumber); err != nil {
		return nil, fmt.Errorf("failed to publish TagAssigned event: %w", err)
	}

	return &userevents.UserSignupResponse{
		Success: true,
	}, nil
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserService) OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequest) (*userevents.UserRoleUpdateResponse, error) {
	if !req.NewRole.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", req.NewRole)
	}

	err := s.UserDB.UpdateUserRole(ctx, req.DiscordID, req.NewRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	// Publish UserRoleUpdated event (for other modules to consume)
	if err := s.publishUserRoleUpdated(req.DiscordID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err) // Log the error, but don't fail the request
	}

	return &userevents.UserRoleUpdateResponse{
		Success: true,
	}, nil
}

// GetUserRole retrieves the role of a user.
func (s *UserService) GetUserRole(ctx context.Context, discordID string) (*userdb.UserRole, error) {
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}
	return &role, nil
}

// GetUser retrieves user data.
func (s *UserService) GetUser(ctx context.Context, discordID string) (*userdb.User, error) {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil // Return user directly, not &user
}

// checkTagAvailability checks if a tag number is available by querying the leaderboard module.
func (s *UserService) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// 1. Publish CheckTagAvailabilityRequest event
	req := userevents.CheckTagAvailabilityRequest{
		TagNumber: tagNumber,
	}

	// Marshal the request manually using encoding/json
	reqData, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequest: %w", err)
	}

	// Use a unique correlation ID for this request
	correlationID := watermill.NewUUID()

	// 2. Publish the event with the correlation ID
	// Note: Define CheckTagAvailabilityRequestSubject in your user module
	_, err = s.Publisher.Publish(userevents.CheckTagAvailabilityRequestSubject, reqData, nats.MsgId(correlationID))
	if err != nil {
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}

	// 3. Subscribe to CheckTagAvailabilityResponseSubject with the correlation ID
	// Note: Define CheckTagAvailabilityResponseSubject and LeaderboardStream in your user module
	sub, err := s.JS.QueueSubscribeSync(userevents.CheckTagAvailabilityResponseSubject, correlationID, nats.BindStream(userevents.LeaderboardStream))
	if err != nil {
		return false, fmt.Errorf("failed to subscribe to CheckTagAvailabilityResponse: %w", err)
	}
	defer sub.Unsubscribe()

	// 4. Receive the response
	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to receive CheckTagAvailabilityResponse: %w", err)
	}

	// 5. Unmarshal the response
	var resp userevents.CheckTagAvailabilityResponse
	// Unmarshal the response manually using encoding/json
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return false, fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
	}

	return resp.IsAvailable, nil
}

// publishTagAssigned publishes a TagAssigned event to the leaderboard module.
func (s *UserService) publishTagAssigned(discordID string, tagNumber int) error {
	evt := userevents.TagAssigned{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}

	// Marshal the event data manually
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssigned: %w", err)
	}

	// Note: Define TagAssignedSubject in your user module
	_, err = s.Publisher.Publish(userevents.TagAssignedSubject, evtData)
	if err != nil {
		return fmt.Errorf("failed to publish TagAssigned: %w", err)
	}

	return nil
}

// publishUserRoleUpdated publishes a UserRoleUpdated event.
func (s *UserService) publishUserRoleUpdated(discordID string, newRole userdb.UserRole) error {
	evt := userevents.UserRoleUpdated{
		DiscordID: discordID,
		NewRole:   newRole,
	}

	// Marshal the event data manually
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal UserRoleUpdated: %w", err)
	}

	_, err = s.Publisher.Publish(userevents.UserRoleUpdatedSubject, evtData)
	if err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdated: %w", err)
	}

	return nil
}
