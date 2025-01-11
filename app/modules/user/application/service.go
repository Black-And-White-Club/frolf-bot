package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB   userdb.UserDB
	eventBus shared.EventBus
	logger   *slog.Logger
}

// NewUserService creates a new UserService.
func NewUserService(db userdb.UserDB, eventBus shared.EventBus, logger *slog.Logger) *UserServiceImpl {
	return &UserServiceImpl{
		UserDB:   db,
		eventBus: eventBus,
		logger:   logger,
	}
}

func (s *UserServiceImpl) OnUserSignupRequest(ctx context.Context, req events.UserSignupRequestPayload) (*events.UserSignupResponsePayload, error) {
	s.logger.Info("OnUserSignupRequest started",
		slog.String("discord_id", string(req.DiscordID)),
		slog.Int("tag_number", req.TagNumber),
	)

	if req.TagNumber != 0 {
		// Step 1: Check tag availability
		isTagAvailable, err := s.checkTagAvailability(ctx, req.TagNumber)
		if err != nil {
			s.logger.Error("Failed to check tag availability", slog.Any("error", err))
			return nil, fmt.Errorf("failed to check tag availability: %w", err)
		}
		if !isTagAvailable {
			s.logger.Info("Tag number is not available", slog.Int("tag_number", req.TagNumber))
			return nil, fmt.Errorf("tag number %d is already taken", req.TagNumber)
		}

		// Step 2: Publish TagAssignedRequest
		if err := s.publishTagAssigned(ctx, req.DiscordID, req.TagNumber); err != nil {
			s.logger.Error("Failed to publish TagAssignedRequest event", slog.Any("error", err))
			return nil, fmt.Errorf("failed to publish TagAssignedRequest: %w", err)
		}
	}

	// Step 3: Create the user after tag handling is complete
	newUser := &userdb.User{
		DiscordID: req.DiscordID,
		Role:      usertypes.UserRoleRattler,
	}
	if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
		s.logger.Error("Failed to create user", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User successfully created", slog.String("discord_id", string(req.DiscordID)))

	return &events.UserSignupResponsePayload{
		Success: true,
	}, nil
}

var errTimeout = errors.New("timeout waiting for tag availability response")

func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Info("checkTagAvailability started", slog.Int("tag_number", tagNumber))

	const defaultTimeout = 3 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	correlationID := watermill.NewUUID()
	responseChan := make(chan events.CheckTagAvailabilityResponsePayload)

	requestPayload, err := json.Marshal(events.CheckTagAvailabilityRequestPayload{
		TagNumber: tagNumber,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequestPayload: %w", err)
	}

	requestMsg := message.NewMessage(watermill.NewUUID(), requestPayload)
	requestMsg.SetContext(ctxWithTimeout)
	requestMsg.Metadata.Set("correlation_id", correlationID)
	requestMsg.Metadata.Set("subject", events.CheckTagAvailabilityRequest)

	// Subscribe to the response in the leaderboard-stream
	responseSubject := events.CheckTagAvailabilityResponse
	responseStream := userstream.LeaderboardStreamName

	err = s.eventBus.Subscribe(ctxWithTimeout, responseStream, responseSubject, func(ctx context.Context, msg *message.Message) error {
		var responsePayload events.CheckTagAvailabilityResponsePayload
		if err := json.Unmarshal(msg.Payload, &responsePayload); err != nil {
			return fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
		}
		responseChan <- responsePayload
		return nil
	})
	if err != nil {
		s.logger.Error("failed to subscribe to response topic", slog.Any("error", err), slog.String("topic", responseStream))
		return false, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	// Publish CheckTagAvailabilityRequest to the leaderboard-stream
	streamName := userstream.LeaderboardStreamName
	if err := s.eventBus.Publish(ctxWithTimeout, streamName, requestMsg); err != nil {
		s.logger.Error("failed to publish CheckTagAvailabilityRequest", slog.Any("error", err))
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err) // Return the error immediately
	}

	select {
	case <-ctx.Done():
		return false, errTimeout
	case responsePayload := <-responseChan:
		return responsePayload.IsAvailable, nil
	}
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserServiceImpl) OnUserRoleUpdateRequest(ctx context.Context, req events.UserRoleUpdateRequestPayload) (*events.UserRoleUpdateResponsePayload, error) {
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

	evt := events.UserRoleUpdatedPayload{
		DiscordID: req.DiscordID,
		NewRole:   req.NewRole.String(),
	}

	if err := s.publishUserRoleUpdated(ctx, evt); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	return &events.UserRoleUpdateResponsePayload{
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
	s.logger.Info("publishTagAssigned called", slog.String("discord_id", string(discordID)), slog.Int("tag_number", tagNumber))

	msgPayload := events.TagAssignedRequestPayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}

	payloadBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssignedRequest payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", events.TagAssignedRequest)

	// Publish to the LeaderboardStreamName
	if err := s.eventBus.Publish(ctx, userstream.LeaderboardStreamName, msg); err != nil {
		return fmt.Errorf("failed to publish TagAssignedRequest event: %w", err)
	}

	return nil
}

func (s *UserServiceImpl) publishUserRoleUpdated(ctx context.Context, evt events.UserRoleUpdatedPayload) error {
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal UserRoleUpdatedPayload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), evtData)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", events.UserRoleUpdated)

	streamName := userstream.UserRoleUpdateResponseStreamName
	if err := s.eventBus.Publish(ctx, streamName, msg); err != nil {
		return fmt.Errorf("eventBus.Publish UserRoleUpdated: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", slog.String("discord_id", string(evt.DiscordID)), slog.String("new_role", evt.NewRole))

	return nil
}
