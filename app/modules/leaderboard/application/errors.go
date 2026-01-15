package leaderboardservice

import (
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// TagSwapNeededError is returned when a requested tag is currently held by someone else.
type TagSwapNeededError struct {
	RequestorID  sharedtypes.DiscordID
	TargetUserID sharedtypes.DiscordID
	TargetTag    sharedtypes.TagNumber
	CurrentTag   sharedtypes.TagNumber
}

func (e *TagSwapNeededError) Error() string {
	return fmt.Sprintf("tag %d is held by %s; swap intent recorded for %s",
		e.TargetTag, e.TargetUserID, e.RequestorID)
}
