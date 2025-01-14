package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
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

func (s *UserServiceImpl) OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequestPayload) (*userevents.UserSignupResponsePayload, error) {
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

	return &userevents.UserSignupResponsePayload{
		Success: true,
	}, nil
}

func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Info("checkTagAvailability started", slog.Int("tag_number", tagNumber))

	const defaultTimeout = 5 * time.Second // Increased timeout for safety
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	correlationID := watermill.NewUUID()
	responseChan := make(chan userevents.CheckTagAvailabilityResponsePayload)

	// Prepare the request payload
	requestPayload, err := json.Marshal(userevents.CheckTagAvailabilityRequestPayload{
		TagNumber: tagNumber,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequestPayload: %w", err)
	}

	// Create the request message
	requestMsg := message.NewMessage(watermill.NewUUID(), requestPayload)
	requestMsg.SetContext(ctxWithTimeout)
	requestMsg.Metadata.Set("correlation_id", correlationID)
	requestMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityRequest)
	requestMsg.Metadata.Set("Reply-To", userevents.CheckTagAvailabilityResponse) // Set Reply-To metadata

	// Log the message metadata before publishing
	s.logger.Debug("Request message metadata before publishing",
		slog.String("correlation_id", correlationID),
		slog.String("Reply-To", requestMsg.Metadata.Get("Reply-To")),
		slog.String("subject", requestMsg.Metadata.Get("subject")),
	)

	// Subscribe to the response on the LeaderboardStreamName
	s.logger.Debug("Subscribing to CheckTagAvailabilityResponse",
		slog.String("stream_name", userevents.LeaderboardStreamName),
		slog.String("subject", userevents.CheckTagAvailabilityResponse),
	)

	// --- Add logging for subscription ---
	s.logger.Info("Attempting to subscribe to CheckTagAvailabilityResponse")
	err = s.eventBus.Subscribe(ctxWithTimeout, userevents.LeaderboardStreamName, userevents.CheckTagAvailabilityResponse, func(ctx context.Context, msg *message.Message) error {
		s.logger.Info("Received CheckTagAvailabilityResponse in subscriber")
		var responsePayload userevents.CheckTagAvailabilityResponsePayload
		if err := json.Unmarshal(msg.Payload, &responsePayload); err != nil {
			return fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
		}
		responseChan <- responsePayload
		return nil
	})
	if err != nil {
		s.logger.Error("Failed to subscribe to response topic",
			slog.Any("error", err),
			slog.String("topic", userevents.LeaderboardStreamName),
		)
		return false, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}
	s.logger.Info("Successfully subscribed to CheckTagAvailabilityResponse")

	// Publish the request to the LeaderboardStreamName
	s.logger.Debug("Publishing CheckTagAvailabilityRequest",
		slog.String("stream_name", userevents.LeaderboardStreamName),
		slog.String("subject", userevents.CheckTagAvailabilityRequest),
		slog.String("Reply-To", userevents.CheckTagAvailabilityResponse),
	)

	s.logger.Info("Publishing CheckTagAvailabilityRequest event")
	if err := s.eventBus.Publish(ctxWithTimeout, userevents.LeaderboardStreamName, requestMsg); err != nil {
		s.logger.Error("Failed to publish CheckTagAvailabilityRequest", slog.Any("error", err))
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}
	s.logger.Info("Published CheckTagAvailabilityRequest event")

	// Wait for the response or timeout
	select {
	case <-ctxWithTimeout.Done():
		return false, fmt.Errorf("timeout waiting for tag availability response")
	case responsePayload := <-responseChan:
		s.logger.Debug("Received CheckTagAvailabilityResponse", slog.Any("response", responsePayload))
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
	// Log the start of the function
	s.logger.Info("publishTagAssigned called",
		slog.String("discord_id", string(discordID)),
		slog.Int("tag_number", tagNumber),
	)

	// Prepare the payload
	msgPayload := userevents.TagAssignedRequestPayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}
	payloadBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssignedRequest payload: %w", err)
	}

	// Create a new message
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", userevents.TagAssignedRequest)
	msg.Metadata.Set("correlation_id", watermill.NewUUID()) // Add correlation ID for tracking

	// Log the prepared message details
	s.logger.Debug("Prepared message for publishing",
		slog.String("stream_name", userevents.LeaderboardStreamName),
		slog.String("subject", userevents.TagAssignedRequest),
		slog.String("message_id", msg.UUID),
	)

	// Validate context
	if ctx.Err() != nil {
		s.logger.Error("Context error before publishing", slog.String("ctx_err", fmt.Sprintf("%v", ctx.Err())))
		return fmt.Errorf("context error before publishing: %w", ctx.Err())
	}

	// Publish to the event bus
	err = s.eventBus.Publish(ctx, userevents.LeaderboardStreamName, msg)
	if err != nil {
		s.logger.Error("Failed to publish TagAssigned event", slog.Any("error", err))
		return fmt.Errorf("failed to publish TagAssigned event: %w", err)
	}

	// Log success
	s.logger.Info("Published TagAssigned event successfully",
		slog.String("stream_name", userevents.LeaderboardStreamName),
		slog.String("subject", userevents.TagAssignedRequest),
		slog.String("message_id", msg.UUID),
	)

	return nil
}

func (s *UserServiceImpl) publishUserRoleUpdated(ctx context.Context, evt userevents.UserRoleUpdatedPayload) error {
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal UserRoleUpdatedPayload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), evtData)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", userevents.UserRoleUpdated)

	// Publish to the UserStreamName
	if err := s.eventBus.Publish(ctx, userevents.UserStreamName, msg); err != nil {
		return fmt.Errorf("eventBus.Publish UserRoleUpdated: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", slog.String("discord_id", string(evt.DiscordID)), slog.String("new_role", evt.NewRole))

	return nil
}
