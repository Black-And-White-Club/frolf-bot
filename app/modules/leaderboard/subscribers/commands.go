package leaderboardcommands

import (
	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
)

// GetLeaderboardRequest represents the command to get the leaderboard.
type GetLeaderboardRequest struct{}

func (GetLeaderboardRequest) CommandName() string {
	return "GetLeaderboardRequest"
}

// UpdateLeaderboardRequest represents the command to update the leaderboard.
type UpdateLeaderboardRequest struct {
	Input leaderboarddto.UpdateLeaderboardInput
}

func (UpdateLeaderboardRequest) CommandName() string {
	return "UpdateLeaderboardRequest"
}

// ReceiveScoresRequest represents the command to receive scores.
type ReceiveScoresRequest struct {
	Input leaderboarddto.ReceiveScoresInput
}

func (ReceiveScoresRequest) CommandName() string {
	return "ReceiveScoresRequest"
}

// AssignTagsRequest represents the command to assign tags.
type AssignTagsRequest struct {
	Input leaderboarddto.AssignTagsInput
}

func (AssignTagsRequest) CommandName() string {
	return "AssignTagsRequest"
}

// InitiateTagSwapRequest represents the command to initiate a tag swap.
type InitiateTagSwapRequest struct {
	Input leaderboarddto.InitiateTagSwapInput
}

func (InitiateTagSwapRequest) CommandName() string {
	return "InitiateTagSwapRequest"
}

// SwapGroupsRequest represents the command to swap groups.
type SwapGroupsRequest struct {
	Input leaderboarddto.SwapGroupsInput
}

func (SwapGroupsRequest) CommandName() string {
	return "SwapGroupsRequest"
}
