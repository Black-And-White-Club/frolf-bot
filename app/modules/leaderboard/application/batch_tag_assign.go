package leaderboardservice

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
)

// ProcessTagAssignments handles tag assignments from different sources with optimized validation
// utilizing utils methods for consistent validation and error handling
// Accepts both string, enum, and payload types for source determination
func (s *LeaderboardService) ProcessTagAssignments(
	ctx context.Context,
	source interface{}, // Accept string, enum, or payload for source determination
	requests []sharedtypes.TagAssignmentRequest,
	requestingUserID *sharedtypes.DiscordID,
	operationID uuid.UUID,
	batchID uuid.UUID,
) (LeaderboardOperationResult, error) {
	// Map source to enum type with intelligent determination
	var sourceType sharedtypes.ServiceUpdateSource
	switch src := source.(type) {
	case string:
		switch src {
		case "user_creation":
			sourceType = sharedtypes.ServiceUpdateSourceCreateUser
		default:
			sourceType = sharedtypes.ServiceUpdateSourceManual
		}
	case sharedtypes.ServiceUpdateSource:
		sourceType = src
	case *sharedevents.BatchTagAssignmentRequestedPayload:
		// Intelligent source determination based on payload context
		if len(src.Assignments) == 1 && src.RequestingUserID == "system" {
			// Single assignment from system is likely user creation
			sourceType = sharedtypes.ServiceUpdateSourceCreateUser
		} else {
			// Multiple assignments or non-system requests are admin batch operations
			sourceType = sharedtypes.ServiceUpdateSourceAdminBatch
		}
	default:
		return LeaderboardOperationResult{}, fmt.Errorf("invalid source type: %T", source)
	}

	s.logger.InfoContext(ctx, "Processing tag assignments",
		attr.ExtractCorrelationID(ctx),
		attr.String("source", string(sourceType)),
		attr.UUIDValue("batch_id", batchID),
		attr.UUIDValue("operation_id", operationID),
		attr.String("requesting_user", getRequestingUserDisplayName(requestingUserID)),
		attr.Int("request_count", len(requests)),
	)

	return s.serviceWrapper(ctx, "ProcessTagAssignments", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// Get current leaderboard for validation
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		if err != nil {
			return s.buildFailureResponse(sourceType, requestingUserID, operationID, batchID, "failed to get leaderboard"), err
		}

		// Early return for empty requests - return current leaderboard state
		if len(requests) == 0 {
			s.logger.InfoContext(ctx, "No requests to process, completing successfully")

			// Get current complete leaderboard for empty requests case
			allRequests := make([]sharedtypes.TagAssignmentRequest, len(currentLeaderboard.LeaderboardData))
			for i, entry := range currentLeaderboard.LeaderboardData {
				allRequests[i] = sharedtypes.TagAssignmentRequest{
					UserID:    entry.UserID,
					TagNumber: entry.TagNumber,
				}
			}

			return s.buildSuccessResponse(sourceType, requestingUserID, operationID, batchID, allRequests), nil
		}

		// For single assignments, use utils methods for proper validation and swap detection
		if len(requests) == 1 {
			return s.processSingleAssignment(ctx, currentLeaderboard, requests[0], sourceType, requestingUserID, operationID, batchID)
		}

		// Handle score processing differently - it's a complete leaderboard replacement
		if sourceType == sharedtypes.ServiceUpdateSourceProcessScores {
			return s.processScoreUpdate(ctx, currentLeaderboard, requests, sourceType, requestingUserID, operationID, batchID)
		}

		// For other batch operations, validate each request using utils methods
		validRequests, swapNeeded := s.validateBatchRequests(ctx, currentLeaderboard, requests)
		if swapNeeded != nil {
			return *swapNeeded, nil
		}

		if len(validRequests) == 0 {
			s.logger.InfoContext(ctx, "No valid requests after validation, completing successfully")

			// Get current complete leaderboard for no valid requests case
			allRequests := make([]sharedtypes.TagAssignmentRequest, len(currentLeaderboard.LeaderboardData))
			for i, entry := range currentLeaderboard.LeaderboardData {
				allRequests[i] = sharedtypes.TagAssignmentRequest{
					UserID:    entry.UserID,
					TagNumber: entry.TagNumber,
				}
			}

			return s.buildSuccessResponse(sourceType, requestingUserID, operationID, batchID, allRequests), nil
		}

		// Convert to tag:user format and use GenerateUpdatedLeaderboard
		tagUserPairs := make([]string, len(validRequests))
		for i, request := range validRequests {
			tagUserPairs[i] = fmt.Sprintf("%d:%s", request.TagNumber, request.UserID)
		}

		newLeaderboardData, err := s.GenerateUpdatedLeaderboard(currentLeaderboard.LeaderboardData, tagUserPairs)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to generate updated leaderboard", attr.Any("error", err))
			// Business logic error - return failure response with nil error
			return s.buildFailureResponse(sourceType, requestingUserID, operationID, batchID, err.Error()), nil
		}

		// Atomic leaderboard update
		startTime := time.Now()
		updatedLeaderboard, err := s.LeaderboardDB.UpdateLeaderboard(ctx, newLeaderboardData, sharedtypes.RoundID(operationID))
		s.metrics.RecordOperationDuration(ctx, "UpdateCompleteLeaderboard", "ProcessTagAssignments", time.Since(startTime))

		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update leaderboard",
				attr.Any("error", err),
				attr.String("source", string(sourceType)),
			)
			// Infrastructure error - return failure response with the error
			return s.buildFailureResponse(sourceType, requestingUserID, operationID, batchID, err.Error()), err
		}

		// Convert complete leaderboard to requests format for Discord client
		allRequests := make([]sharedtypes.TagAssignmentRequest, len(updatedLeaderboard.LeaderboardData))
		for i, entry := range updatedLeaderboard.LeaderboardData {
			allRequests[i] = sharedtypes.TagAssignmentRequest{
				UserID:    entry.UserID,
				TagNumber: entry.TagNumber,
			}
		}

		s.logger.InfoContext(ctx, "Tag assignments completed successfully",
			attr.Int("assignment_count", len(validRequests)),
			attr.Int("total_leaderboard_entries", len(allRequests)),
			attr.String("source", string(sourceType)),
		)

		return s.buildSuccessResponse(sourceType, requestingUserID, operationID, batchID, allRequests), nil
	})
}

// processScoreUpdate handles score processing updates which are complete leaderboard replacements
// and don't need individual validation since they represent the authoritative final state
func (s *LeaderboardService) processScoreUpdate(
	ctx context.Context,
	currentLeaderboard *leaderboarddb.Leaderboard,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	requestingUserID *sharedtypes.DiscordID,
	operationID uuid.UUID,
	batchID uuid.UUID,
) (LeaderboardOperationResult, error) {
	s.logger.InfoContext(ctx, "Processing score update - complete leaderboard replacement",
		attr.Int("request_count", len(requests)),
	)

	// Convert to tag:user format and use GenerateUpdatedLeaderboard
	// Score processing is authoritative - no individual validation needed
	tagUserPairs := make([]string, len(requests))
	for i, request := range requests {
		tagUserPairs[i] = fmt.Sprintf("%d:%s", request.TagNumber, request.UserID)
	}

	newLeaderboardData, err := s.GenerateUpdatedLeaderboard(currentLeaderboard.LeaderboardData, tagUserPairs)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to generate updated leaderboard for score processing", attr.Any("error", err))
		// Business logic error - return failure response with nil error
		return s.buildFailureResponse(source, requestingUserID, operationID, batchID, err.Error()), nil
	}

	// Atomic leaderboard update
	startTime := time.Now()
	updatedLeaderboard, err := s.LeaderboardDB.UpdateLeaderboard(ctx, newLeaderboardData, sharedtypes.RoundID(operationID))
	s.metrics.RecordOperationDuration(ctx, "UpdateCompleteLeaderboard", "ProcessScoreUpdate", time.Since(startTime))

	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update leaderboard for score processing",
			attr.Any("error", err),
		)
		// Infrastructure error - return failure response with the error
		return s.buildFailureResponse(source, requestingUserID, operationID, batchID, err.Error()), err
	}

	// Convert complete leaderboard to requests format
	allRequests := make([]sharedtypes.TagAssignmentRequest, len(updatedLeaderboard.LeaderboardData))
	for i, entry := range updatedLeaderboard.LeaderboardData {
		allRequests[i] = sharedtypes.TagAssignmentRequest{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		}
	}

	s.logger.InfoContext(ctx, "Score processing leaderboard update completed successfully",
		attr.Int("updated_users", len(requests)),
		attr.Int("total_leaderboard_entries", len(allRequests)),
	)

	return s.buildSuccessResponse(source, requestingUserID, operationID, batchID, allRequests), nil
}

// processSingleAssignment uses utils methods for proper validation and error handling
func (s *LeaderboardService) processSingleAssignment(
	ctx context.Context,
	currentLeaderboard *leaderboarddb.Leaderboard,
	request sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	requestingUserID *sharedtypes.DiscordID,
	operationID uuid.UUID,
	batchID uuid.UUID,
) (LeaderboardOperationResult, error) {
	// Early validation for invalid tag numbers
	if request.TagNumber <= 0 {
		s.logger.ErrorContext(ctx, "Invalid tag number",
			attr.String("user_id", string(request.UserID)),
			attr.Int("tag_number", int(request.TagNumber)),
		)
		// Return failure response for invalid tag numbers
		return s.buildFailureResponse(source, requestingUserID, operationID, batchID, fmt.Sprintf("invalid tag number: %d", request.TagNumber)), nil
	}

	// Check if user exists in leaderboard
	userExists := s.userExistsInLeaderboard(currentLeaderboard, request.UserID)

	var newLeaderboardData leaderboardtypes.LeaderboardData
	var err error

	// Use appropriate utils method based on user existence
	if userExists {
		newLeaderboardData, err = s.PrepareTagUpdateForExistingUser(currentLeaderboard, request.UserID, request.TagNumber)
	} else {
		newLeaderboardData, err = s.PrepareTagAssignment(currentLeaderboard, request.UserID, request.TagNumber)
	}

	// Handle specific error types
	if err != nil {
		if swapErr, ok := err.(*TagSwapNeededError); ok {
			s.logger.InfoContext(ctx, "Tag assignment requires swap",
				attr.String("requesting_user", string(swapErr.RequestorID)),
				attr.String("current_holder", string(swapErr.TargetID)),
				attr.Int("tag_number", int(swapErr.TagNumber)),
			)
			return LeaderboardOperationResult{
				Success: &leaderboardevents.TagSwapRequestedPayload{
					RequestorID: swapErr.RequestorID,
					TargetID:    swapErr.TargetID,
				},
			}, nil
		}

		s.logger.ErrorContext(ctx, "Failed to prepare tag assignment",
			attr.Any("error", err),
			attr.String("user_id", string(request.UserID)),
			attr.Int("tag_number", int(request.TagNumber)),
		)
		// Return failure response for other validation errors
		return s.buildFailureResponse(source, requestingUserID, operationID, batchID, err.Error()), nil
	}

	// No-op case (user already has the tag)
	if newLeaderboardData == nil {
		s.logger.InfoContext(ctx, "User already has requested tag, no update needed",
			attr.String("user_id", string(request.UserID)),
			attr.Int("tag_number", int(request.TagNumber)),
		)

		// Get current complete leaderboard for no-op case
		allRequests := make([]sharedtypes.TagAssignmentRequest, len(currentLeaderboard.LeaderboardData))
		for i, entry := range currentLeaderboard.LeaderboardData {
			allRequests[i] = sharedtypes.TagAssignmentRequest{
				UserID:    entry.UserID,
				TagNumber: entry.TagNumber,
			}
		}

		return s.buildSuccessResponse(source, requestingUserID, operationID, batchID, allRequests), nil
	}

	// Update leaderboard with the new data
	startTime := time.Now()
	updatedLeaderboard, err := s.LeaderboardDB.UpdateLeaderboard(ctx, newLeaderboardData, sharedtypes.RoundID(operationID))
	s.metrics.RecordOperationDuration(ctx, "UpdateCompleteLeaderboard", "ProcessTagAssignments", time.Since(startTime))

	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update leaderboard for single assignment",
			attr.Any("error", err),
			attr.String("user_id", string(request.UserID)),
		)
		// Infrastructure error - return failure response with the error
		return s.buildFailureResponse(source, requestingUserID, operationID, batchID, err.Error()), err
	}

	// Convert complete leaderboard to requests format
	allRequests := make([]sharedtypes.TagAssignmentRequest, len(updatedLeaderboard.LeaderboardData))
	for i, entry := range updatedLeaderboard.LeaderboardData {
		allRequests[i] = sharedtypes.TagAssignmentRequest{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		}
	}

	s.logger.InfoContext(ctx, "Single tag assignment completed successfully",
		attr.String("user_id", string(request.UserID)),
		attr.Int("tag_number", int(request.TagNumber)),
		attr.Int("total_leaderboard_entries", len(allRequests)),
	)

	return s.buildSuccessResponse(source, requestingUserID, operationID, batchID, allRequests), nil
}

// validateBatchRequests validates all requests using utils methods and skips any that would require swaps
func (s *LeaderboardService) validateBatchRequests(
	ctx context.Context,
	currentLeaderboard *leaderboarddb.Leaderboard,
	requests []sharedtypes.TagAssignmentRequest,
) ([]sharedtypes.TagAssignmentRequest, *LeaderboardOperationResult) {
	validRequests := make([]sharedtypes.TagAssignmentRequest, 0, len(requests))

	for i, request := range requests {
		// Use utils methods for validation
		userExists := s.userExistsInLeaderboard(currentLeaderboard, request.UserID)

		var err error
		if userExists {
			_, err = s.PrepareTagUpdateForExistingUser(currentLeaderboard, request.UserID, request.TagNumber)
		} else {
			_, err = s.PrepareTagAssignment(currentLeaderboard, request.UserID, request.TagNumber)
		}

		if err != nil {
			// For batch operations, skip ALL errors (including swaps)
			// Batch operations don't trigger swaps - they just skip problematic assignments
			s.logger.WarnContext(ctx, "Skipping invalid request in batch",
				attr.String("user_id", string(request.UserID)),
				attr.Int("tag_number", int(request.TagNumber)),
				attr.String("error", err.Error()),
				attr.Int("request_index", i),
			)
			continue
		}

		validRequests = append(validRequests, request)
	}

	return validRequests, nil
}

// userExistsInLeaderboard checks if a user exists in the current leaderboard
func (s *LeaderboardService) userExistsInLeaderboard(leaderboard *leaderboarddb.Leaderboard, userID sharedtypes.DiscordID) bool {
	for _, entry := range leaderboard.LeaderboardData {
		if entry.UserID == userID {
			return true
		}
	}
	return false
}

// resolveRequestingUser returns the requesting user or system default
func resolveRequestingUser(userID *sharedtypes.DiscordID) sharedtypes.DiscordID {
	if userID == nil {
		return sharedtypes.DiscordID("system")
	}
	return *userID
}

// getRequestingUserDisplayName returns the requesting user as a string for logging
func getRequestingUserDisplayName(userID *sharedtypes.DiscordID) string {
	if userID == nil {
		return "system"
	}
	return string(*userID)
}

// buildSuccessResponse creates the appropriate success response based on the source
func (s *LeaderboardService) buildSuccessResponse(
	source sharedtypes.ServiceUpdateSource,
	requestingUserID *sharedtypes.DiscordID,
	operationID uuid.UUID,
	batchID uuid.UUID,
	completedRequests []sharedtypes.TagAssignmentRequest,
) LeaderboardOperationResult {
	switch source {
	case sharedtypes.ServiceUpdateSourceProcessScores:
		// Score processing updates return leaderboard updated event
		return LeaderboardOperationResult{
			Success: &leaderboardevents.LeaderboardUpdatedPayload{
				RoundID: sharedtypes.RoundID(operationID),
			},
		}

	case sharedtypes.ServiceUpdateSourceCreateUser:
		// Single user creation returns individual tag assigned event
		if len(completedRequests) == 1 {
			request := completedRequests[0]
			return LeaderboardOperationResult{
				Success: &leaderboardevents.TagAssignedPayload{
					UserID:       request.UserID,
					TagNumber:    &request.TagNumber,
					AssignmentID: sharedtypes.RoundID(operationID),
					Source:       string(source),
				},
			}
		}
		// Multiple assignments use batch response
		return LeaderboardOperationResult{
			Success: createBatchAssignedPayload(completedRequests, resolveRequestingUser(requestingUserID), batchID),
		}

	case sharedtypes.ServiceUpdateSourceAdminBatch, sharedtypes.ServiceUpdateSourceManual:
		// Admin operations return batch assigned event
		return LeaderboardOperationResult{
			Success: createBatchAssignedPayload(completedRequests, resolveRequestingUser(requestingUserID), batchID),
		}

	default:
		// Fallback to batch response for unknown sources
		return LeaderboardOperationResult{
			Success: createBatchAssignedPayload(completedRequests, resolveRequestingUser(requestingUserID), batchID),
		}
	}
}

// buildFailureResponse creates the appropriate failure response based on the source
func (s *LeaderboardService) buildFailureResponse(
	source sharedtypes.ServiceUpdateSource,
	requestingUserID *sharedtypes.DiscordID,
	operationID uuid.UUID,
	batchID uuid.UUID,
	errorReason string,
) LeaderboardOperationResult {
	switch source {
	case sharedtypes.ServiceUpdateSourceProcessScores:
		// Score processing failures return leaderboard update failed event
		return LeaderboardOperationResult{
			Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: sharedtypes.RoundID(operationID),
				Reason:  errorReason,
			},
		}

	default:
		// All other failures return batch assignment failed event
		return LeaderboardOperationResult{
			Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
				RequestingUserID: resolveRequestingUser(requestingUserID),
				BatchID:          batchID.String(),
				Reason:           errorReason,
			},
		}
	}
}

// createBatchAssignedPayload creates a batch assigned payload from requests
func createBatchAssignedPayload(
	requests []sharedtypes.TagAssignmentRequest,
	requestingUser sharedtypes.DiscordID,
	batchID uuid.UUID,
) *leaderboardevents.BatchTagAssignedPayload {
	if len(requests) == 0 {
		return &leaderboardevents.BatchTagAssignedPayload{
			RequestingUserID: requestingUser,
			BatchID:          batchID.String(),
			AssignmentCount:  0,
			Assignments:      []leaderboardevents.TagAssignmentInfo{},
		}
	}

	assignments := make([]leaderboardevents.TagAssignmentInfo, len(requests))
	for i, request := range requests {
		assignments[i] = leaderboardevents.TagAssignmentInfo{
			UserID:    request.UserID,
			TagNumber: request.TagNumber,
		}
	}

	return &leaderboardevents.BatchTagAssignedPayload{
		RequestingUserID: requestingUser,
		BatchID:          batchID.String(),
		AssignmentCount:  len(requests),
		Assignments:      assignments,
	}
}
