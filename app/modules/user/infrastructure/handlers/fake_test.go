package userhandlers

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
)

// ------------------------
// Fake User Service
// ------------------------

type FakeUserService struct {
	trace []string

	CreateUserFunc               func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error)
	UpdateUserRoleInDatabaseFunc func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error)
	GetUserRoleFunc              func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error)
	GetUserFunc                  func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error)
	UpdateUDiscIdentityFunc      func(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error)
	FindByUDiscUsernameFunc      func(ctx context.Context, guildID sharedtypes.GuildID, username string) (userservice.UserWithMembershipResult, error)
	FindByUDiscNameFunc          func(ctx context.Context, guildID sharedtypes.GuildID, name string) (userservice.UserWithMembershipResult, error)
	MatchParsedScorecardFunc     func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (results.OperationResult[*userservice.MatchResult, error], error)
}

func NewFakeUserService() *FakeUserService {
	return &FakeUserService{trace: []string{}}
}

func (f *FakeUserService) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeUserService) Trace() []string {
	return f.trace
}

// --- Service Interface Implementation ---

func (f *FakeUserService) CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
	f.record("CreateUser")
	if f.CreateUserFunc != nil {
		return f.CreateUserFunc(ctx, guildID, userID, tag, udiscUsername, udiscName)
	}
	return userservice.UserResult{}, nil
}

func (f *FakeUserService) UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error) {
	f.record("UpdateUserRoleInDatabase")
	if f.UpdateUserRoleInDatabaseFunc != nil {
		return f.UpdateUserRoleInDatabaseFunc(ctx, guildID, userID, newRole)
	}
	return results.OperationResult[bool, error]{}, nil
}

func (f *FakeUserService) GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
	f.record("GetUserRole")
	if f.GetUserRoleFunc != nil {
		return f.GetUserRoleFunc(ctx, guildID, userID)
	}
	return results.OperationResult[sharedtypes.UserRoleEnum, error]{}, nil
}

func (f *FakeUserService) GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error) {
	f.record("GetUser")
	if f.GetUserFunc != nil {
		return f.GetUserFunc(ctx, guildID, userID)
	}
	return userservice.UserWithMembershipResult{}, nil
}

func (f *FakeUserService) UpdateUDiscIdentity(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error) {
	f.record("UpdateUDiscIdentity")
	if f.UpdateUDiscIdentityFunc != nil {
		return f.UpdateUDiscIdentityFunc(ctx, userID, username, name)
	}
	return results.OperationResult[bool, error]{}, nil
}

func (f *FakeUserService) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (userservice.UserWithMembershipResult, error) {
	f.record("FindByUDiscUsername")
	if f.FindByUDiscUsernameFunc != nil {
		return f.FindByUDiscUsernameFunc(ctx, guildID, username)
	}
	return userservice.UserWithMembershipResult{}, nil
}

func (f *FakeUserService) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (userservice.UserWithMembershipResult, error) {
	f.record("FindByUDiscName")
	if f.FindByUDiscNameFunc != nil {
		return f.FindByUDiscNameFunc(ctx, guildID, name)
	}
	return userservice.UserWithMembershipResult{}, nil
}

func (f *FakeUserService) MatchParsedScorecard(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (results.OperationResult[*userservice.MatchResult, error], error) {
	f.record("MatchParsedScorecard")
	if f.MatchParsedScorecardFunc != nil {
		return f.MatchParsedScorecardFunc(ctx, guildID, userID, playerNames)
	}
	return results.OperationResult[*userservice.MatchResult, error]{}, nil
}

var _ userservice.Service = (*FakeUserService)(nil)
