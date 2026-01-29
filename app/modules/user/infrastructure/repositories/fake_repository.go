package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// FakeRepository is a fake implementation of Repository for testing.
type FakeRepository struct {
	GetUserGlobalFn         func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error)
	SaveGlobalUserFn        func(ctx context.Context, db bun.IDB, user *User) error
	UpdateGlobalUserFn      func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error
	CreateGuildMembershipFn func(ctx context.Context, db bun.IDB, membership *GuildMembership) error
	GetGuildMembershipFn    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error)
	UpdateMembershipRoleFn  func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMembershipsFn    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error)
	GetUserByUserIDFn       func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error)
	GetUserRoleFn           func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRoleFn        func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsernameFn   func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error)
	FindByUDiscNameFn       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error)
	FindByUDiscNameFuzzyFn  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error)
}

func (f *FakeRepository) GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error) {
	if f.GetUserGlobalFn != nil {
		return f.GetUserGlobalFn(ctx, db, userID)
	}
	return nil, nil
}

func (f *FakeRepository) SaveGlobalUser(ctx context.Context, db bun.IDB, user *User) error {
	if f.SaveGlobalUserFn != nil {
		return f.SaveGlobalUserFn(ctx, db, user)
	}
	return nil
}

func (f *FakeRepository) UpdateGlobalUser(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error {
	if f.UpdateGlobalUserFn != nil {
		return f.UpdateGlobalUserFn(ctx, db, userID, updates)
	}
	return nil
}

func (f *FakeRepository) CreateGuildMembership(ctx context.Context, db bun.IDB, membership *GuildMembership) error {
	if f.CreateGuildMembershipFn != nil {
		return f.CreateGuildMembershipFn(ctx, db, membership)
	}
	return nil
}

func (f *FakeRepository) GetGuildMembership(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error) {
	if f.GetGuildMembershipFn != nil {
		return f.GetGuildMembershipFn(ctx, db, userID, guildID)
	}
	return nil, nil
}

func (f *FakeRepository) UpdateMembershipRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	if f.UpdateMembershipRoleFn != nil {
		return f.UpdateMembershipRoleFn(ctx, db, userID, guildID, role)
	}
	return nil
}

func (f *FakeRepository) GetUserMemberships(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error) {
	if f.GetUserMembershipsFn != nil {
		return f.GetUserMembershipsFn(ctx, db, userID)
	}
	return nil, nil
}

func (f *FakeRepository) GetUserByUserID(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error) {
	if f.GetUserByUserIDFn != nil {
		return f.GetUserByUserIDFn(ctx, db, userID, guildID)
	}
	return nil, nil
}

func (f *FakeRepository) GetUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
	if f.GetUserRoleFn != nil {
		return f.GetUserRoleFn(ctx, db, userID, guildID)
	}
	return "", nil
}

func (f *FakeRepository) UpdateUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	if f.UpdateUserRoleFn != nil {
		return f.UpdateUserRoleFn(ctx, db, userID, guildID, role)
	}
	return nil
}

func (f *FakeRepository) FindByUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error) {
	if f.FindByUDiscUsernameFn != nil {
		return f.FindByUDiscUsernameFn(ctx, db, guildID, username)
	}
	return nil, nil
}

func (f *FakeRepository) FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error) {
	if f.FindByUDiscNameFn != nil {
		return f.FindByUDiscNameFn(ctx, db, guildID, name)
	}
	return nil, nil
}

func (f *FakeRepository) FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error) {
	if f.FindByUDiscNameFuzzyFn != nil {
		return f.FindByUDiscNameFuzzyFn(ctx, db, guildID, partialName)
	}
	return nil, nil
}
