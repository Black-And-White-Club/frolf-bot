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

var ErrUserNotFound = errors.New("user not found")

// UserDBImpl is a repository for user data operations.
type UserDBImpl struct {
	DB *bun.DB
}

// GetUserGlobal checks if a user exists globally (no guild context).
func (db *UserDBImpl) GetUserGlobal(ctx context.Context, userID sharedtypes.DiscordID) (*User, error) {
	user := &User{}
	err := db.DB.NewSelect().
		Model(user).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get global user: %w", err)
	}
	return user, nil
}

// CreateGlobalUser creates a new global user record.
func (db *UserDBImpl) CreateGlobalUser(ctx context.Context, user *User) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create global user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateUDiscIdentityGlobal updates the global user record (applies to all guilds).
func (db *UserDBImpl) UpdateUDiscIdentityGlobal(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) error {
	normalizedUsername := normalizeNullablePointer(username)
	normalizedName := normalizeNullablePointer(name)

	_, err := db.DB.NewUpdate().
		Model((*User)(nil)).
		Set("udisc_username = ?", normalizedUsername).
		Set("udisc_name = ?", normalizedName).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update udisc identity: %w", err)
	}
	return nil
}

// CreateGuildMembership creates a new guild membership.
func (db *UserDBImpl) CreateGuildMembership(ctx context.Context, membership *GuildMembership) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.NewInsert().Model(membership).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create guild membership: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetGuildMembership retrieves a guild membership.
func (db *UserDBImpl) GetGuildMembership(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error) {
	membership := &GuildMembership{}
	err := db.DB.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get guild membership: %w", err)
	}
	return membership, nil
}

// UpdateMembershipRole updates the role in a guild membership.
func (db *UserDBImpl) UpdateMembershipRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if !role.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	result, err := tx.NewUpdate().
		Model((*GuildMembership)(nil)).
		Set("role = ?", role).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute update membership role query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected after update: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserMemberships retrieves all guild memberships for a user.
func (db *UserDBImpl) GetUserMemberships(ctx context.Context, userID sharedtypes.DiscordID) ([]*GuildMembership, error) {
	var memberships []*GuildMembership
	err := db.DB.NewSelect().
		Model(&memberships).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user memberships: %w", err)
	}
	return memberships, nil
}

// GetUserByUserID retrieves a user with their guild membership by their Discord ID and Guild ID.
func (db *UserDBImpl) GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := db.DB.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("u.user_id = ?", userID).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by user id: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = db.DB.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild membership: %w", err)
	}

	uwm.Role = membership.Role
	uwm.JoinedAt = membership.JoinedAt

	return uwm, nil
}

// GetUserRole retrieves the role of a user by their Discord ID and Guild ID.
func (db *UserDBImpl) GetUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
	membership := &GuildMembership{}
	err := db.DB.NewSelect().
		Model(membership).
		Column("role").
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}
	return membership.Role, nil
}

// UpdateUserRole updates the role of an existing user within a transaction.
func (db *UserDBImpl) UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	return db.UpdateMembershipRole(ctx, userID, guildID, role)
}

// FindByUDiscUsername looks up a user by UDisc username within a guild.
func (db *UserDBImpl) FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := db.DB.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_username) = LOWER(?)", username).
		Where("gm.guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by udisc username: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = db.DB.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", uwm.User.UserID, guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild membership: %w", err)
	}

	uwm.Role = membership.Role
	uwm.JoinedAt = membership.JoinedAt

	return uwm, nil
}

// FindByUDiscName looks up a user by UDisc name within a guild.
func (db *UserDBImpl) FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error) {
	uwm := &UserWithMembership{
		User: &User{},
	}
	err := db.DB.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_name) = LOWER(?)", name).
		Where("gm.guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by udisc name: %w", err)
	}

	// Now get the membership details
	membership := &GuildMembership{}
	err = db.DB.NewSelect().
		Model(membership).
		Where("user_id = ? AND guild_id = ?", uwm.User.UserID, guildID).
		Scan(ctx)
	if err != nil {
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
