package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Impl implements Repository using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new user repository implementation.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// GetUserGlobal checks if a user exists globally (no guild context).
func (r *Impl) GetUserGlobal(ctx context.Context, userID sharedtypes.DiscordID) (*User, error) {
	user := &User{}
	err := r.db.NewSelect().
		Model(user).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetUserGlobal: %w", err)
	}
	return user, nil
}

// CreateGlobalUser creates a new global user record.
func (r *Impl) CreateGlobalUser(ctx context.Context, user *User) error {
	if _, err := r.db.NewInsert().Model(user).Exec(ctx); err != nil {
		return fmt.Errorf("userdb.CreateGlobalUser: %w", err)
	}
	return nil
}

// UpdateUDiscIdentityGlobal updates the global user record (applies to all guilds).
func (r *Impl) UpdateUDiscIdentityGlobal(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) error {
	normalizedUsername := normalizeNullablePointer(username)
	normalizedName := normalizeNullablePointer(name)

	q := r.db.NewUpdate().
		Model((*User)(nil)).
		Set("udisc_username = ?", normalizedUsername).
		Set("udisc_name = ?", normalizedName).
		Where("user_id = ?", userID)

	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("userdb.UpdateUDiscIdentityGlobal: %w", err)
	}
	return nil
}

// CreateGuildMembership creates a new guild membership.
func (r *Impl) CreateGuildMembership(ctx context.Context, membership *GuildMembership) error {
	if _, err := r.db.NewInsert().Model(membership).Exec(ctx); err != nil {
		return fmt.Errorf("userdb.CreateGuildMembership: %w", err)
	}
	return nil
}

// GetGuildMembership retrieves a guild membership.
func (r *Impl) GetGuildMembership(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error) {
	membership := &GuildMembership{}
	err := r.db.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetGuildMembership: %w", err)
	}
	return membership, nil
}

// UpdateMembershipRole updates the role in a guild membership.
func (r *Impl) UpdateMembershipRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	res, err := r.db.NewUpdate().
		Model((*GuildMembership)(nil)).
		Set("role = ?", role).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.UpdateMembershipRole: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// GetUserMemberships retrieves all guild memberships for a user.
func (r *Impl) GetUserMemberships(ctx context.Context, userID sharedtypes.DiscordID) ([]*GuildMembership, error) {
	var memberships []*GuildMembership
	err := r.db.NewSelect().
		Model(&memberships).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetUserMemberships: %w", err)
	}
	return memberships, nil
}

// GetUserByUserID retrieves a user with their guild membership by their Discord ID and Guild ID.
func (r *Impl) GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := r.db.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("u.user_id = ?", userID).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetUserByUserID: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = r.db.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetUserByUserID (membership): %w", err)
	}

	uwm.Role = membership.Role
	uwm.JoinedAt = membership.JoinedAt

	return uwm, nil
}

// GetUserRole retrieves the role of a user by their Discord ID and Guild ID.
func (r *Impl) GetUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
	membership := &GuildMembership{}
	err := r.db.NewSelect().
		Model(membership).
		Column("role").
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("userdb.GetUserRole: %w", err)
	}
	return membership.Role, nil
}

// UpdateUserRole updates the role of an existing user within a transaction.
func (r *Impl) UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	return r.UpdateMembershipRole(ctx, userID, guildID, role)
}

// FindByUDiscUsername looks up a user by UDisc username within a guild.
func (r *Impl) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := r.db.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_username) = LOWER(?)", username).
		Where("gm.guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.FindByUDiscUsername: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = r.db.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", uwm.User.UserID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get guild membership: %w", err)
	}

	uwm.Role = membership.Role
	uwm.JoinedAt = membership.JoinedAt

	return uwm, nil
}

// FindByUDiscName looks up a user by UDisc name within a guild.
func (r *Impl) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := r.db.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_name) = LOWER(?)", name).
		Where("gm.guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.FindByUDiscName: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = r.db.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", uwm.User.UserID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get guild membership: %w", err)
	}

	uwm.Role = membership.Role
	uwm.JoinedAt = membership.JoinedAt

	return uwm, nil
}

func normalizeNullablePointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}
