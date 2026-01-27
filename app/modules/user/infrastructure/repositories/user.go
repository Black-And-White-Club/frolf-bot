package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

// --- UPDATE STRUCTS ---

// UserUpdateFields represents optional fields to update for global user.
type UserUpdateFields struct {
	UDiscUsername *string
	UDiscName     *string
}

// IsEmpty returns true if no fields are set for update.
func (u *UserUpdateFields) IsEmpty() bool {
	if u == nil {
		return true
	}
	return u.UDiscUsername == nil && u.UDiscName == nil
}

// --- GLOBAL USER METHODS ---

// GetUserGlobal retrieves a global user by Discord ID.
func (r *Impl) GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error) {
	if db == nil {
		db = r.db
	}
	user := &User{}
	err := db.NewSelect().
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

// SaveGlobalUser creates or updates a global user (upsert).
func (r *Impl) SaveGlobalUser(ctx context.Context, db bun.IDB, user *User) error {
	if db == nil {
		db = r.db
	}
	now := time.Now().UTC()
	q := db.NewInsert().
		Model(user).
		On("CONFLICT (user_id) DO UPDATE").
		Set("udisc_username = EXCLUDED.udisc_username").
		Set("udisc_name = EXCLUDED.udisc_name").
		Set("updated_at = ?", now)
	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("userdb.SaveGlobalUser: %w", err)
	}
	return nil
}

// UpdateGlobalUser applies partial updates to a global user.
func (r *Impl) UpdateGlobalUser(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error {
	if db == nil {
		db = r.db
	}
	if updates == nil || updates.IsEmpty() {
		return nil
	}
	q := db.NewUpdate().Model((*User)(nil)).Where("user_id = ?", userID)
	if updates.UDiscUsername != nil {
		q = q.Set("udisc_username = ?", normalizeNullablePointer(updates.UDiscUsername))
	}
	if updates.UDiscName != nil {
		q = q.Set("udisc_name = ?", normalizeNullablePointer(updates.UDiscName))
	}
	q = q.Set("updated_at = ?", time.Now().UTC())

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.UpdateGlobalUser: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// --- GUILD MEMBERSHIP METHODS ---

// CreateGuildMembership inserts a new membership.
func (r *Impl) CreateGuildMembership(ctx context.Context, db bun.IDB, membership *GuildMembership) error {
	if db == nil {
		db = r.db
	}
	if _, err := db.NewInsert().Model(membership).Exec(ctx); err != nil {
		return fmt.Errorf("userdb.CreateGuildMembership: %w", err)
	}
	return nil
}

// GetGuildMembership retrieves a membership for a user in a guild.
func (r *Impl) GetGuildMembership(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error) {
	if db == nil {
		db = r.db
	}
	m := &GuildMembership{}
	err := db.NewSelect().
		Model(m).
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetGuildMembership: %w", err)
	}
	return m, nil
}

// UpdateMembershipRole updates a user's role in a guild.
func (r *Impl) UpdateMembershipRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	if db == nil {
		db = r.db
	}
	if !role.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}
	res, err := db.NewUpdate().
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

// GetUserMemberships retrieves all memberships for a user.
func (r *Impl) GetUserMemberships(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error) {
	if db == nil {
		db = r.db
	}
	var memberships []*GuildMembership
	err := db.NewSelect().
		Model(&memberships).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetUserMemberships: %w", err)
	}
	return memberships, nil
}

// --- USER WITH MEMBERSHIP METHODS ---

// GetUserByUserID fetches user and membership in one query.
func (r *Impl) GetUserByUserID(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error) {
	if db == nil {
		db = r.db
	}

	uwm := &UserWithMembership{User: &User{}}

	// Fallback: Use the explicit table name (usually plural) instead of an alias
	// or use ColumnExpr to define the source.
	err := db.NewSelect().
		Model(uwm.User).
		ColumnExpr("users.*"). // Replace 'u' with the actual table name 'users'
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
		Join("JOIN guild_memberships AS gm ON users.user_id = gm.user_id").
		Where("users.user_id = ?", userID).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx, uwm)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetUserByUserID: %w", err)
	}

	return uwm, nil
}

// GetUserRole retrieves just the user's role in a guild.
func (r *Impl) GetUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
	if db == nil {
		db = r.db
	}
	m := &GuildMembership{}
	err := db.NewSelect().
		Model(m).
		Column("role").
		Where("user_id = ? AND guild_id = ?", userID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("userdb.GetUserRole: %w", err)
	}
	return m.Role, nil
}

// UpdateUserRole updates role (wrapper for UpdateMembershipRole).
func (r *Impl) UpdateUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	return r.UpdateMembershipRole(ctx, db, userID, guildID, role)
}

// --- SEARCH METHODS ---

func (r *Impl) FindByUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error) {
	if db == nil {
		db = r.db
	}
	uwm := &UserWithMembership{User: &User{}}
	err := db.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
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
	return uwm, nil
}

func (r *Impl) FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error) {
	if db == nil {
		db = r.db
	}
	uwm := &UserWithMembership{User: &User{}}
	err := db.NewSelect().
		Model(uwm.User).
		ColumnExpr("u.*").
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
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
	return uwm, nil
}

// Fuzzy search by partial username or name
func (r *Impl) FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partial string) ([]*UserWithMembership, error) {
	if db == nil {
		db = r.db
	}
	search := "%" + strings.ToLower(partial) + "%"

	var users []*User
	err := db.NewSelect().
		Model(&users).
		ColumnExpr("u.*").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("(LOWER(u.udisc_username) LIKE ? OR LOWER(u.udisc_name) LIKE ?)", search, search).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*UserWithMembership{}, nil
		}
		return nil, fmt.Errorf("userdb.FindByUDiscNameFuzzy: %w", err)
	}

	if len(users) == 0 {
		return []*UserWithMembership{}, nil
	}

	userIDs := make([]sharedtypes.DiscordID, len(users))
	for i, u := range users {
		userIDs[i] = u.UserID
	}

	var memberships []*GuildMembership
	err = db.NewSelect().
		Model(&memberships).
		Where("user_id IN (?)", bun.In(userIDs)).
		Where("guild_id = ?", guildID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("userdb.FindByUDiscNameFuzzy (memberships): %w", err)
	}

	mMap := make(map[sharedtypes.DiscordID]*GuildMembership)
	for _, m := range memberships {
		mMap[m.UserID] = m
	}

	results := make([]*UserWithMembership, 0, len(users))
	for _, u := range users {
		if m, ok := mMap[u.UserID]; ok {
			results = append(results, &UserWithMembership{
				User:     u,
				Role:     m.Role,
				JoinedAt: m.JoinedAt,
			})
		}
	}

	return results, nil
}

// --- HELPERS ---

func normalizeNullablePointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}
