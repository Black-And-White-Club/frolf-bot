package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// FakeRepository is a fake implementation of Repository for testing.
type FakeRepository struct {
	GetUUIDByDiscordIDFn          func(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error)
	GetClubUUIDByDiscordGuildIDFn func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error)
	GetDiscordGuildIDByClubUUIDFn func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error)

	GetUserGlobalFn         func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error)
	GetUserByUUIDFn         func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*User, error)
	GetByUserIDsFn          func(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*User, error)
	SaveGlobalUserFn        func(ctx context.Context, db bun.IDB, user *User) error
	UpdateGlobalUserFn      func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error
	CreateGuildMembershipFn func(ctx context.Context, db bun.IDB, membership *GuildMembership) error
	GetGuildMembershipFn    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error)
	UpdateMembershipRoleFn  func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMembershipsFn    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error)

	GetClubMembershipFn             func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*ClubMembership, error)
	UpsertClubMembershipFn          func(ctx context.Context, db bun.IDB, membership *ClubMembership) error
	GetClubMembershipByExternalIDFn func(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*ClubMembership, error)
	GetClubMembershipsByUserUUIDFn  func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*ClubMembership, error)
	GetClubMembershipsByUserUUIDsFn func(ctx context.Context, db bun.IDB, userUUIDs []uuid.UUID) ([]*ClubMembership, error)

	GetUserByUserIDFn           func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error)
	GetUserRoleFn               func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRoleFn            func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsernameFn       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error)
	FindGlobalByUDiscUsernameFn func(ctx context.Context, db bun.IDB, username string) (*User, error)
	FindByUDiscNameFn           func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error)
	GetUsersByUDiscNamesFn      func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, names []string) ([]UserWithMembership, error)
	GetUsersByUDiscUsernamesFn  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, usernames []string) ([]UserWithMembership, error)
	FindByUDiscNameFuzzyFn      func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error)
	// Profile operations
	UpdateProfileFn func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error

	// Refresh Token operations
	SaveRefreshTokenFn    func(ctx context.Context, db bun.IDB, token *RefreshToken) error
	GetRefreshTokenFn     func(ctx context.Context, db bun.IDB, hash string) (*RefreshToken, error)
	RevokeRefreshTokenFn  func(ctx context.Context, db bun.IDB, hash string) error
	RevokeAllUserTokensFn func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error

	// Magic Link operations
	SaveMagicLinkFn     func(ctx context.Context, db bun.IDB, link *MagicLink) error
	GetMagicLinkFn      func(ctx context.Context, db bun.IDB, token string) (*MagicLink, error)
	MarkMagicLinkUsedFn func(ctx context.Context, db bun.IDB, token string) error
}

func (f *FakeRepository) GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	if f.GetUUIDByDiscordIDFn != nil {
		return f.GetUUIDByDiscordIDFn(ctx, db, discordID)
	}
	return uuid.Nil, nil
}

func (f *FakeRepository) GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	if f.GetClubUUIDByDiscordGuildIDFn != nil {
		return f.GetClubUUIDByDiscordGuildIDFn(ctx, db, guildID)
	}
	return uuid.Nil, nil
}

func (f *FakeRepository) GetDiscordGuildIDByClubUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error) {
	if f.GetDiscordGuildIDByClubUUIDFn != nil {
		return f.GetDiscordGuildIDByClubUUIDFn(ctx, db, clubUUID)
	}
	return "", nil
}

func (f *FakeRepository) GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*ClubMembership, error) {
	if f.GetClubMembershipFn != nil {
		return f.GetClubMembershipFn(ctx, db, userUUID, clubUUID)
	}
	return nil, nil
}

func (f *FakeRepository) UpsertClubMembership(ctx context.Context, db bun.IDB, membership *ClubMembership) error {
	if f.UpsertClubMembershipFn != nil {
		return f.UpsertClubMembershipFn(ctx, db, membership)
	}
	return nil
}

func (f *FakeRepository) SaveMagicLink(ctx context.Context, db bun.IDB, link *MagicLink) error {
	if f.SaveMagicLinkFn != nil {
		return f.SaveMagicLinkFn(ctx, db, link)
	}
	return nil
}

func (f *FakeRepository) GetMagicLink(ctx context.Context, db bun.IDB, token string) (*MagicLink, error) {
	if f.GetMagicLinkFn != nil {
		return f.GetMagicLinkFn(ctx, db, token)
	}
	return nil, nil
}

func (f *FakeRepository) MarkMagicLinkUsed(ctx context.Context, db bun.IDB, token string) error {
	if f.MarkMagicLinkUsedFn != nil {
		return f.MarkMagicLinkUsedFn(ctx, db, token)
	}
	return nil
}

func (f *FakeRepository) GetClubMembershipByExternalID(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*ClubMembership, error) {
	if f.GetClubMembershipByExternalIDFn != nil {
		return f.GetClubMembershipByExternalIDFn(ctx, db, externalID, clubUUID)
	}
	return nil, nil
}

func (f *FakeRepository) GetClubMembershipsByUserUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*ClubMembership, error) {
	if f.GetClubMembershipsByUserUUIDFn != nil {
		return f.GetClubMembershipsByUserUUIDFn(ctx, db, userUUID)
	}
	return nil, nil
}

func (f *FakeRepository) GetClubMembershipsByUserUUIDs(ctx context.Context, db bun.IDB, userUUIDs []uuid.UUID) ([]*ClubMembership, error) {
	if f.GetClubMembershipsByUserUUIDsFn != nil {
		return f.GetClubMembershipsByUserUUIDsFn(ctx, db, userUUIDs)
	}
	return nil, nil
}

func (f *FakeRepository) GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error) {
	if f.GetUserGlobalFn != nil {
		return f.GetUserGlobalFn(ctx, db, userID)
	}
	return nil, nil
}

func (f *FakeRepository) GetByUserIDs(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*User, error) {
	if f.GetByUserIDsFn != nil {
		return f.GetByUserIDsFn(ctx, db, userIDs)
	}
	return []*User{}, nil
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

func (f *FakeRepository) UpdateProfile(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error {
	if f.UpdateProfileFn != nil {
		return f.UpdateProfileFn(ctx, db, userID, displayName, avatarHash)
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

func (f *FakeRepository) FindGlobalByUDiscUsername(ctx context.Context, db bun.IDB, username string) (*User, error) {
	if f.FindGlobalByUDiscUsernameFn != nil {
		return f.FindGlobalByUDiscUsernameFn(ctx, db, username)
	}
	return nil, nil
}

func (f *FakeRepository) FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error) {
	if f.FindByUDiscNameFn != nil {
		return f.FindByUDiscNameFn(ctx, db, guildID, name)
	}
	return nil, nil
}

func (f *FakeRepository) GetUsersByUDiscNames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, names []string) ([]UserWithMembership, error) {
	if f.GetUsersByUDiscNamesFn != nil {
		return f.GetUsersByUDiscNamesFn(ctx, db, guildID, names)
	}
	return nil, nil
}

func (f *FakeRepository) GetUsersByUDiscUsernames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, usernames []string) ([]UserWithMembership, error) {
	if f.GetUsersByUDiscUsernamesFn != nil {
		return f.GetUsersByUDiscUsernamesFn(ctx, db, guildID, usernames)
	}
	return nil, nil
}

func (f *FakeRepository) FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error) {
	if f.FindByUDiscNameFuzzyFn != nil {
		return f.FindByUDiscNameFuzzyFn(ctx, db, guildID, partialName)
	}
	return nil, nil
}

func (f *FakeRepository) SaveRefreshToken(ctx context.Context, db bun.IDB, token *RefreshToken) error {
	if f.SaveRefreshTokenFn != nil {
		return f.SaveRefreshTokenFn(ctx, db, token)
	}
	return nil
}

func (f *FakeRepository) GetRefreshToken(ctx context.Context, db bun.IDB, hash string) (*RefreshToken, error) {
	if f.GetRefreshTokenFn != nil {
		return f.GetRefreshTokenFn(ctx, db, hash)
	}
	return nil, nil
}

func (f *FakeRepository) RevokeRefreshToken(ctx context.Context, db bun.IDB, hash string) error {
	if f.RevokeRefreshTokenFn != nil {
		return f.RevokeRefreshTokenFn(ctx, db, hash)
	}
	return nil
}

func (f *FakeRepository) RevokeAllUserTokens(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error {
	if f.RevokeAllUserTokensFn != nil {
		return f.RevokeAllUserTokensFn(ctx, db, userUUID)
	}
	return nil
}
