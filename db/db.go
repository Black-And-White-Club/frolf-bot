package db

import (
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	sharedinterface "github.com/Black-And-White-Club/frolf-bot/app/shared/interfaces"
)

const (
	// DBTypePostgres represents an underlying POSTGRES database type.
	DatabaseTypePostgres string = "POSTGRES"
)

// DB provides methods for interacting with an underlying database or other storage mechanism.
type Database interface {
	leaderboarddb.LeaderboardDB
	rounddb.RoundDB
	scoredb.ScoreDB
	userdb.UserDB
	guilddb.GuildDB
	sharedinterface.GuildConfigReader
}
