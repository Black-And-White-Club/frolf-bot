package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB       userdb.UserDB
	eventBus     shared.EventBus
	logger       shared.LoggerAdapter
	eventAdapter shared.EventAdapterInterface
}

// NewUserService creates a new UserService.
func NewUserService(db userdb.UserDB, eventBus shared.EventBus, logger shared.LoggerAdapter, eventAdapter shared.EventAdapterInterface) *UserServiceImpl {
	return &UserServiceImpl{
		UserDB:       db,
		eventBus:     eventBus,
		logger:       logger,
		eventAdapter: eventAdapter,
	}
}

func (s *UserServiceImpl) OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequestPayload) (*userevents.UserSignupResponsePayload, error) {
	s.logger.Info("OnUserSignupRequest started", shared.LogFields{
		"discord_id": req.DiscordID,
		"tag_number": req.TagNumber,
	})

	var isTagAvailable bool
	var err error

	if req.TagNumber != 0 {
		// Launch a goroutine to check tag availability asynchronously
		tagAvailabilityChan := make(chan bool)
		go func() {
			isTagAvailable, err = s.checkTagAvailability(ctx, req.TagNumber)
			tagAvailabilityChan <- isTagAvailable
		}()

		// Wait for the result or a timeout
		select {
		case <-time.After(3 * time.Second): // Adjust timeout as needed
			return nil, errTimeout
		case isTagAvailable = <-tagAvailabilityChan:
			if err != nil {
				return nil, fmt.Errorf("failed to check tag availability: %w", err)
			}

			if !isTagAvailable {
				return nil, fmt.Errorf("tag number %d is already taken", req.TagNumber)
			}
		}
	}

	newUser := &userdb.User{
		DiscordID: req.DiscordID,
		Role:      usertypes.UserRoleRattler,
	}
	if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if req.TagNumber != 0 {
		if err := s.publishTagAssigned(ctx, newUser.DiscordID, req.TagNumber); err != nil {
			s.logger.Error("failed to publish TagAssignedRequest event", err, shared.LogFields{"error": err})
			return nil, err
		}
	}

	return &userevents.UserSignupResponsePayload{
		Success: true,
	}, nil
}

var errTimeout = errors.New("timeout waiting for tag availability response")

func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Info("checkTagAvailability started", shared.LogFields{
		"tag_number": tagNumber,
	})

	const defaultTimeout = 3 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	correlationID := shared.NewUUID()
	responseChan := make(chan userevents.CheckTagAvailabilityResponsePayload)

	requestPayload := []byte(fmt.Sprintf(`{"tag_number":%d}`, tagNumber))
	requestMsg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), requestPayload)
	requestMsg.SetMetadata("correlation_id", correlationID)

	// Publish CheckTagAvailabilityRequest
	if err := s.eventBus.Publish(ctxWithTimeout, userevents.CheckTagAvailabilityRequest, requestMsg); err != nil {
		s.logger.Error("failed to publish CheckTagAvailabilityRequest", err, shared.LogFields{"error": err})
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}

	// Proceed with Subscribe only if Publish succeeds
	responseTopic := userevents.CheckTagAvailabilityResponse.String() + "." + correlationID
	go func() {
		err := s.eventBus.Subscribe(ctxWithTimeout, responseTopic, func(ctx context.Context, msg shared.Message) error {
			var responsePayload userevents.CheckTagAvailabilityResponsePayload
			if err := json.Unmarshal(msg.Payload(), &responsePayload); err != nil {
				return fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
			}
			if responsePayload.Error != "" {
				return fmt.Errorf("leaderboard error: %s", responsePayload.Error)
			}
			responseChan <- responsePayload
			return nil
		})
		if err != nil {
			s.logger.Error("failed to subscribe to response topic", err, shared.LogFields{"topic": responseTopic})
		}
	}()

	select {
	case <-ctx.Done():
		return false, errTimeout
	case responsePayload := <-responseChan:
		return responsePayload.IsAvailable, nil
	}
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserServiceImpl) OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequestPayload) (*userevents.UserRoleUpdateResponsePayload, error) {
	if !req.NewRole.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", req.NewRole)
	}

	if req.DiscordID == "" {
		return nil, fmt.Errorf("missing DiscordID in request")
	}

	if err := s.UserDB.UpdateUserRole(ctx, req.DiscordID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	user, err := s.UserDB.GetUserByDiscordID(ctx, req.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found: %s", req.DiscordID)
	}

	evt := userevents.UserRoleUpdatedPayload{
		DiscordID: req.DiscordID,
		NewRole:   req.NewRole.String(),
	}

	if err := s.publishUserRoleUpdated(ctx, evt); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	return &userevents.UserRoleUpdateResponsePayload{
		Success: true,
	}, nil
}

// GetUserRole retrieves the role of a user.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error) {
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		return usertypes.UserRoleUnknown, fmt.Errorf("failed to get user role: %w", err)
	}
	return role, nil
}

// GetUser retrieves user data by Discord ID.
func (s *UserServiceImpl) GetUser(ctx context.Context, discordID usertypes.DiscordID) (usertypes.User, error) {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *UserServiceImpl) publishTagAssigned(ctx context.Context, discordID usertypes.DiscordID, tagNumber int) error {
	s.logger.Info("publishTagAssigned called", shared.LogFields{
		"discord_id": discordID,
		"tag_number": tagNumber,
	})

	msgPayload := userevents.TagAssignedRequestPayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}

	payloadBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssignedRequest payload: %w", err)
	}

	msg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), payloadBytes)

	if err := s.eventBus.Publish(ctx, userevents.TagAssignedRequest, msg); err != nil {
		return fmt.Errorf("failed to publish TagAssignedRequest event: %w", err)
	}

	return nil
}

func (s *UserServiceImpl) publishUserRoleUpdated(ctx context.Context, evt userevents.UserRoleUpdatedPayload) error {
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal UserRoleUpdatedPayload: %w", err)
	}

	msg := adapters.NewWatermillMessageAdapter(shared.NewUUID(), evtData)

	if err := s.eventBus.Publish(ctx, userevents.UserRoleUpdated, msg); err != nil {
		return fmt.Errorf("eventBus.Publish UserRoleUpdated: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", shared.LogFields{
		"discord_id": evt.DiscordID,
		"new_role":   evt.NewRole,
	})

	return nil
}
