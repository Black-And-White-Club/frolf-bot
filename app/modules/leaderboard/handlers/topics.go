package leaderboardhandlers

const (
	TopicLeaderboardEntriesReceived = "leaderboard.entries.received"
	TopicAssignLeaderboardTags      = "leaderboard.tags.assign"
	TopicLeaderboardTagsAssigned    = "leaderboard.tags.assigned"
	TopicUpdateLeaderboard          = "leaderboard.update"
	TopicGetLeaderboard             = "leaderboard.get"
	TopicGetUserLeaderboardEntry    = "leaderboard.user.get"
	TopicGetLeaderboardTag          = "leaderboard.tag.get"
	TopicCheckLeaderboardTag        = "leaderboard.tag.check"
	TopicUserTagResponse            = "user.tag.response"
	TopicTagAvailabilityResponse    = "tag.availability.response"
	TopicParticipantTagResponse     = "participant.tag.response"
	TagSwapRequest                  = "tag.swap.request"
	TopicAssignTag                  = "assign.tag"
	TopicTagSwapCommand             = "tag.swap.command"
	TopicUserCreated                = "user.created"
	TopicReceiveScores              = "leaderboard.scores.receive"   // Added
	TopicInitiateTagSwap            = "leaderboard.tagSwap.initiate" // Added
	TopicSwapGroups                 = "leaderboard.groups.swap"      // Added
	TopicAssignTags                 = "assign.tags"
)
