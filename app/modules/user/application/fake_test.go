package userservice

import (
	"context"
	"database/sql"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake User Repo
// ------------------------

// FakeDB is a minimal fake that satisfies the requirement for RunInTx.
type FakeDB struct {
	bun.IDB // Embedded to satisfy the full interface if needed
}

// RunInTx simply executes the provided function, bypassing real DB logic.
func (f *FakeDB) RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	// We pass an empty bun.Tx{}.
	// As long as your repo methods use the 'db' parameter passed to them (which is bun.IDB),
	// this works perfectly.
	return fn(ctx, bun.Tx{})
}

// FakeUserRepository provides a programmable stub for the userdb.Repository interface.
type FakeUserRepository struct {
	trace []string

	// Identity resolution
	GetUUIDByDiscordIDFn          func(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error)
	GetClubUUIDByDiscordGuildIDFn func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error)

	// Global user operations
	GetUserGlobalFunc    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*userdb.User, error)
	SaveGlobalUserFunc   func(ctx context.Context, db bun.IDB, user *userdb.User) error
	UpdateGlobalUserFunc func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *userdb.UserUpdateFields) error
	GetByUserIDsFunc     func(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*userdb.User, error)

	// Guild membership operations
	CreateGuildMembershipFunc func(ctx context.Context, db bun.IDB, membership *userdb.GuildMembership) error
	GetGuildMembershipFunc    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*userdb.GuildMembership, error)
	UpdateMembershipRoleFunc  func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMembershipsFunc    func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*userdb.GuildMembership, error)

	// Club membership operations
	GetClubMembershipFn             func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error)
	GetClubMembershipsByUserUUIDFn  func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*userdb.ClubMembership, error)
	UpsertClubMembershipFn          func(ctx context.Context, db bun.IDB, membership *userdb.ClubMembership) error
	GetClubMembershipByExternalIDFn func(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*userdb.ClubMembership, error)

	// Guild-scoped operations
	GetUserByUserIDFunc      func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*userdb.UserWithMembership, error)
	GetUserRoleFunc          func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRoleFunc       func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsernameFunc  func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*userdb.UserWithMembership, error)
	FindByUDiscNameFunc      func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*userdb.UserWithMembership, error)
	FindByUDiscNameFuzzyFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*userdb.UserWithMembership, error)

	// Profile operations
	UpdateProfileFunc func(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error

	// Refresh Token operations
	SaveRefreshTokenFn    func(ctx context.Context, db bun.IDB, token *userdb.RefreshToken) error
	GetRefreshTokenFn     func(ctx context.Context, db bun.IDB, hash string) (*userdb.RefreshToken, error)
	RevokeRefreshTokenFn  func(ctx context.Context, db bun.IDB, hash string) error
	RevokeAllUserTokensFn func(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error
}

// Trace returns the sequence of method calls made to the fake.
func (f *FakeUserRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// NewFakeUserRepository initializes a new FakeUserRepository.
func NewFakeUserRepository() *FakeUserRepository {
	return &FakeUserRepository{
		trace: []string{},
	}
}

func (f *FakeUserRepository) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Repository Interface Implementation ---

func (f *FakeUserRepository) GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	f.record("GetUUIDByDiscordID")
	if f.GetUUIDByDiscordIDFn != nil {
		return f.GetUUIDByDiscordIDFn(ctx, db, discordID)
	}
	return uuid.Nil, nil
}

func (f *FakeUserRepository) GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	f.record("GetClubUUIDByDiscordGuildID")
	if f.GetClubUUIDByDiscordGuildIDFn != nil {
		return f.GetClubUUIDByDiscordGuildIDFn(ctx, db, guildID)
	}
	return uuid.Nil, nil
}

func (f *FakeUserRepository) GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*userdb.User, error) {
	f.record("GetUserGlobal")
	if f.GetUserGlobalFunc != nil {
		return f.GetUserGlobalFunc(ctx, db, userID)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) SaveGlobalUser(ctx context.Context, db bun.IDB, user *userdb.User) error {
	f.record("SaveGlobalUser")
	if f.SaveGlobalUserFunc != nil {
		return f.SaveGlobalUserFunc(ctx, db, user)
	}
	return nil
}

func (f *FakeUserRepository) UpdateGlobalUser(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *userdb.UserUpdateFields) error {
	f.record("UpdateGlobalUser")
	if f.UpdateGlobalUserFunc != nil {
		return f.UpdateGlobalUserFunc(ctx, db, userID, updates)
	}
	return nil
}

func (f *FakeUserRepository) CreateGuildMembership(ctx context.Context, db bun.IDB, membership *userdb.GuildMembership) error {
	f.record("CreateGuildMembership")
	if f.CreateGuildMembershipFunc != nil {
		return f.CreateGuildMembershipFunc(ctx, db, membership)
	}
	return nil
}

func (f *FakeUserRepository) GetGuildMembership(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*userdb.GuildMembership, error) {
	f.record("GetGuildMembership")
	if f.GetGuildMembershipFunc != nil {
		return f.GetGuildMembershipFunc(ctx, db, userID, guildID)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) UpdateMembershipRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	f.record("UpdateMembershipRole")
	if f.UpdateMembershipRoleFunc != nil {
		return f.UpdateMembershipRoleFunc(ctx, db, userID, guildID, role)
	}
	return nil
}

func (f *FakeUserRepository) GetUserMemberships(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*userdb.GuildMembership, error) {
	f.record("GetUserMemberships")
	if f.GetUserMembershipsFunc != nil {
		return f.GetUserMembershipsFunc(ctx, db, userID)
	}
	return []*userdb.GuildMembership{}, nil
}

func (f *FakeUserRepository) GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
	f.record("GetClubMembership")
	if f.GetClubMembershipFn != nil {
		return f.GetClubMembershipFn(ctx, db, userUUID, clubUUID)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) UpsertClubMembership(ctx context.Context, db bun.IDB, membership *userdb.ClubMembership) error {
	f.record("UpsertClubMembership")
	if f.UpsertClubMembershipFn != nil {
		return f.UpsertClubMembershipFn(ctx, db, membership)
	}
	return nil
}

func (f *FakeUserRepository) GetClubMembershipByExternalID(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
	f.record("GetClubMembershipByExternalID")
	if f.GetClubMembershipByExternalIDFn != nil {
		return f.GetClubMembershipByExternalIDFn(ctx, db, externalID, clubUUID)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) GetClubMembershipsByUserUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*userdb.ClubMembership, error) {
	f.record("GetClubMembershipsByUserUUID")
	if f.GetClubMembershipsByUserUUIDFn != nil {
		return f.GetClubMembershipsByUserUUIDFn(ctx, db, userUUID)
	}
	return nil, nil
}

func (f *FakeUserRepository) GetUserByUserID(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*userdb.UserWithMembership, error) {
	f.record("GetUserByUserID")
	if f.GetUserByUserIDFunc != nil {
		return f.GetUserByUserIDFunc(ctx, db, userID, guildID)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) GetUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
	f.record("GetUserRole")
	if f.GetUserRoleFunc != nil {
		return f.GetUserRoleFunc(ctx, db, userID, guildID)
	}
	return "", userdb.ErrNotFound
}

func (f *FakeUserRepository) UpdateUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	f.record("UpdateUserRole")
	if f.UpdateUserRoleFunc != nil {
		return f.UpdateUserRoleFunc(ctx, db, userID, guildID, role)
	}
	return nil
}

func (f *FakeUserRepository) FindByUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*userdb.UserWithMembership, error) {
	f.record("FindByUDiscUsername")
	if f.FindByUDiscUsernameFunc != nil {
		return f.FindByUDiscUsernameFunc(ctx, db, guildID, username)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*userdb.UserWithMembership, error) {
	f.record("FindByUDiscName")
	if f.FindByUDiscNameFunc != nil {
		return f.FindByUDiscNameFunc(ctx, db, guildID, name)
	}
	return nil, userdb.ErrNotFound
}

func (f *FakeUserRepository) FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*userdb.UserWithMembership, error) {
	f.record("FindByUDiscNameFuzzy")
	if f.FindByUDiscNameFuzzyFunc != nil {
		return f.FindByUDiscNameFuzzyFunc(ctx, db, guildID, partialName)
	}
	return []*userdb.UserWithMembership{}, nil
}

func (f *FakeUserRepository) GetByUserIDs(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*userdb.User, error) {
	f.record("GetByUserIDs")
	if f.GetByUserIDsFunc != nil {
		return f.GetByUserIDsFunc(ctx, db, userIDs)
	}
	return []*userdb.User{}, nil
}

func (f *FakeUserRepository) UpdateProfile(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error {
	f.record("UpdateProfile")
	if f.UpdateProfileFunc != nil {
		return f.UpdateProfileFunc(ctx, db, userID, displayName, avatarHash)
	}
	return nil
}

func (f *FakeUserRepository) SaveRefreshToken(ctx context.Context, db bun.IDB, token *userdb.RefreshToken) error {
	f.record("SaveRefreshToken")
	if f.SaveRefreshTokenFn != nil {
		return f.SaveRefreshTokenFn(ctx, db, token)
	}
	return nil
}

func (f *FakeUserRepository) GetRefreshToken(ctx context.Context, db bun.IDB, hash string) (*userdb.RefreshToken, error) {
	f.record("GetRefreshToken")
	if f.GetRefreshTokenFn != nil {
		return f.GetRefreshTokenFn(ctx, db, hash)
	}
	return nil, nil
}

func (f *FakeUserRepository) RevokeRefreshToken(ctx context.Context, db bun.IDB, hash string) error {
	f.record("RevokeRefreshToken")
	if f.RevokeRefreshTokenFn != nil {
		return f.RevokeRefreshTokenFn(ctx, db, hash)
	}
	return nil
}

func (f *FakeUserRepository) RevokeAllUserTokens(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error {
	f.record("RevokeAllUserTokens")
	if f.RevokeAllUserTokensFn != nil {
		return f.RevokeAllUserTokensFn(ctx, db, userUUID)
	}
	return nil
}

// Ensure the fake actually satisfies the interface
var _ userdb.Repository = (*FakeUserRepository)(nil)
