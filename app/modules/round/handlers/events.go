package roundhandlers

import rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"

// DeleteRoundRequest represents the request to delete a round.
type DeleteRoundRequest struct {
	RoundID int64 `json:"round_id"`
}

type RoundDeletedEvent struct {
	RoundID int64 `json:"round_id"`
}

// EditRoundRequest represents the request to edit a round.
type EditRoundRequest struct {
	RoundID int64                  `json:"round_id"`
	Input   rounddb.EditRoundInput `json:"input"`
}

// RoundEditedEvent represents the event triggered when a round is edited.
type RoundEditedEvent struct {
	RoundID int64 `json:"round_id"`
}
