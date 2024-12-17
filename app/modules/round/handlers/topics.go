package roundhandlers

const (
	TopicCreateRound            = "create.round"
	TopicGetRound               = "get.round"
	TopicGetRoundResponse       = "get.round.response"
	TopicGetRounds              = "get.rounds"
	TopicGetRoundsResponse      = "get.rounds.response"
	TopicEditRound              = "edit.round"
	TopicDeleteRound            = "delete.round"
	TopicUpdateParticipant      = "update.participant"
	TopicJoinRound              = "join.round"
	TopicSubmitScore            = "submit.score"
	TopicStartRound             = "start.round"
	TopicRecordScores           = "record.scores"
	TopicProcessScoreSubmission = "process.score.submission"
	TopicFinalizeRound          = "finalize.round"
	TopicGetTagNumberRequest    = "get.tag.number.request"
	TopicGetTagNumberResponse   = "get.tag.number.response"
	TopicRoundScoresProcessed   = "round.scores.processed" // Added for consistency
	TopicRoundReminder          = "round.reminder"         // Added for consistency
	TopicParticipantJoinedRound = "participant.joined.round"
	TopicSendScores             = "send.scores"
	TopicRoundStart             = "round.start"
	TopicRoundStateUpdated      = "round.state.updated"
	TopicRoundDeleted           = "round.deleted"
	TopicRoundEdited            = "round.edited"
	TopicRoundFinalized         = "round.finalized" // Added for consistency
	TopicScheduledTask          = "scheduled.task"
)
