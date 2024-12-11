package db

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
)

const (
	// DBTypePostgres represents an underlying POSTGRES database type.
	DatabaseTypePostgres string = "POSTGRES"
)

// DB provides methods for interacting with an underlying database or other storage mechanism.
type Database interface {
	LeaderboardDB
	rounddb.RoundDB
	ScoreDB
	userdb.UserDB
}
