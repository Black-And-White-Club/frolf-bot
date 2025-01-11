package userstream

const (
	UserSignupRequestStreamName      = "user-signup-request-stream"
	UserSignupResponseStreamName     = "user-signup-response-stream"
	UserRoleUpdateRequestStreamName  = "user-role-update-request-stream"
	UserRoleUpdateResponseStreamName = "user-role-update-response-stream"
	LeaderboardStreamName            = "leaderboard-stream"
)

// StreamNameForEvent returns the appropriate stream name based on the event type.
func StreamNameForEvent(eventType string) string {
	// ...
	switch eventType {
	case "user.signup.request":
		return UserSignupRequestStreamName
	case "user.signup.response":
		return UserSignupResponseStreamName
	case "user.role.update.request":
		return UserRoleUpdateRequestStreamName
	case "user.role.update.response":
		return UserRoleUpdateResponseStreamName
	// ... other cases for leaderboard events
	default:
		return UserSignupRequestStreamName
	}
}
