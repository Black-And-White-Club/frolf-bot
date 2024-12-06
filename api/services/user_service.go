package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
)

// UserService handles user-related logic and database interactions.
type UserService struct {
	db                 db.UserDB
	natsConnectionPool *nats.NatsConnectionPool
}

// NewUserService creates a new UserService.
func NewUserService(db db.UserDB, natsConnectionPool *nats.NatsConnectionPool) *UserService {
	return &UserService{
		db:                 db,
		natsConnectionPool: natsConnectionPool,
	}
}

// GetUser retrieves a user by Discord ID.
func (s *UserService) GetUser(ctx context.Context, discordID string) (*models.User, error) {
	return s.db.GetUser(ctx, discordID)
}

// CreateUser handles user creation logic, including tag availability checks.
func (s *UserService) CreateUser(ctx context.Context, user *models.User, tagNumber int) error {
	// Validate input
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.DiscordID == "" || user.Name == "" || user.Role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Handle tag number logic
	switch {
	case tagNumber != 0: // Tag number provided
		// 1. Prepare the request data for the leaderboard module
		checkTagEvent := &nats.CheckTagAvailabilityEvent{
			TagNumber: tagNumber,
			// No ReplyTo field needed here
		}
		log.Printf("Sending request to leaderboard service: discordID=%s, tagNumber=%d", user.DiscordID, tagNumber)

		// 2. Get a connection from the pool
		conn, err := s.natsConnectionPool.GetConnection()
		if err != nil {
			return fmt.Errorf("failed to get NATS connection from pool: %w", err)
		}
		defer s.natsConnectionPool.ReleaseConnection(conn)

		// 3. Marshal the request data
		payload, err := json.Marshal(checkTagEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal request data for subject %s: %w", "check-tag-availability", err)
		}

		// 4. Publish the request
		err = conn.Publish("check-tag-availability", payload)
		if err != nil {
			return fmt.Errorf("failed to publish request: %w", err)
		}

		//  5. If the tag is available, create the user
		if err := s.db.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// 6. Publish the "user.created" event
		userCreatedEvent := &nats.UserCreatedEvent{
			DiscordID: user.DiscordID,
			TagNumber: tagNumber,
		}
		if err := s.natsConnectionPool.Publish("user.created", userCreatedEvent); err != nil {
			return fmt.Errorf("failed to publish user.created event: %w", err)
		}

		return nil

	default: // No tag number provided
		// Create the user without a tag
		if err := s.db.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		return nil
	}
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, discordID string, updates *models.User) error {
	// Get the existing user from the database
	existingUser, err := s.db.GetUser(ctx, discordID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Update the fields of the existing user with the provided updates
	if updates.Name != "" {
		existingUser.Name = updates.Name
	}
	if updates.Role != "" {
		existingUser.Role = updates.Role
	}

	// Save the updated user to the database
	err = s.db.UpdateUser(ctx, discordID, existingUser) // Pass discordID to UpdateUser
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Publish UserUpdatedEvent to NATS
	err = s.natsConnectionPool.Publish("user.updated", &nats.UserUpdatedEvent{
		DiscordID: existingUser.DiscordID,
		Name:      existingUser.Name,
		Role:      existingUser.Role,
	})
	if err != nil {
		log.Printf("Failed to publish user.updated event: %v", err)
	}

	return nil
}

// GetUserRole retrieves the role of a user.
func (s *UserService) GetUserRole(ctx context.Context, discordID string) (models.UserRole, error) {
	user, err := s.db.GetUser(ctx, discordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}
	return user.Role, nil
}

// StartNATSSubscribers starts the NATS subscribers for the user service.
func (s *UserService) StartNATSSubscribers(ctx context.Context) error {
	conn, err := s.natsConnectionPool.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer s.natsConnectionPool.ReleaseConnection(conn)

	// Subscribe to "user.get_role" subject
	_, err = conn.Subscribe("user.get_role", func(msg *nats.Msg) {
		var event nats.UserGetRoleEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshaling UserGetRoleEvent: %v", err)
			return
		}

		// Get the user's role
		role, err := s.GetUserRole(ctx, event.DiscordID)
		if err != nil {
			log.Printf("Error getting user role: %v", err)
			return
		}

		// Publish the response
		response := &nats.UserGetRoleResponse{
			Role: role,
		}
		responseData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshaling UserGetRoleResponse: %v", err)
			return
		}

		err = conn.Publish(event.ReplyTo, responseData)
		if err != nil {
			log.Printf("Error publishing UserGetRoleResponse: %v", err)
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user.get_role: %w", err)
	}

	// Subscribe to "user.check-tag-availability-response" subject
	_, err = conn.Subscribe("tag-availability-result", func(msg *nats.Msg) {
		var tagAvailabilityResponse nats.TagAvailabilityResponse
		err := json.Unmarshal(msg.Data, &tagAvailabilityResponse)
		if err != nil {
			log.Printf("Failed to unmarshal tag availability response: %v", err)
			return
		}

		// Log the tag availability status
		if tagAvailabilityResponse.IsAvailable {
			log.Printf("Tag is available")
		} else {
			log.Printf("Tag is not available")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to user.check-tag-availability-response: %w", err)
	}

	return nil
}
