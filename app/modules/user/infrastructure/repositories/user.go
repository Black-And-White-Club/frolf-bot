package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
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

// --- IDENTITY RESOLUTION METHODS ---

func (r *Impl) GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error) {
	if db == nil {
		db = r.db
	}
	var u User
	err := db.NewSelect().Model(&u).Column("uuid").Where("user_id = ?", discordID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return u.UUID, nil
}

func (r *Impl) GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
	if db == nil {
		db = r.db
	}
	var gc struct {
		UUID uuid.UUID `bun:"uuid"`
	}
	err := db.NewSelect().Table("guild_configs").Column("uuid").Where("guild_id = ?", guildID).Scan(ctx, &gc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return gc.UUID, nil
}

func (r *Impl) GetDiscordGuildIDByClubUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error) {
	if db == nil {
		db = r.db
	}
	var gc struct {
		GuildID sharedtypes.GuildID `bun:"guild_id"`
	}
	err := db.NewSelect().Table("guild_configs").Column("guild_id").Where("uuid = ?", clubUUID).Scan(ctx, &gc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return gc.GuildID, nil
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

// GetByUserIDs fetches multiple users by their Discord IDs
func (r *Impl) GetByUserIDs(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*User, error) {
	if db == nil {
		db = r.db
	}
	if len(userIDs) == 0 {
		return []*User{}, nil
	}

	var users []*User
	err := db.NewSelect().
		Model(&users).
		Where("user_id IN (?)", bun.In(userIDs)).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("userdb.GetByUserIDs: %w", err)
	}

	return users, nil
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

// UpdateProfile updates user's display name and avatar.
func (r *Impl) UpdateProfile(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error {
	if db == nil {
		db = r.db
	}
	now := time.Now().UTC()

	user := &User{
		UserID:           &userID,
		DisplayName:      &displayName,
		AvatarHash:       &avatarHash,
		ProfileUpdatedAt: &now,
		UpdatedAt:        now,
		CreatedAt:        now,
	}

	_, err := db.NewInsert().
		Model(user).
		On("CONFLICT (user_id) DO UPDATE").
		Set("display_name = EXCLUDED.display_name").
		Set("avatar_hash = EXCLUDED.avatar_hash").
		Set("profile_updated_at = EXCLUDED.profile_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("userdb.UpdateProfile: %w", err)
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

// --- CLUB MEMBERSHIP METHODS ---

func (r *Impl) GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*ClubMembership, error) {
	if db == nil {
		db = r.db
	}
	cm := &ClubMembership{}
	err := db.NewSelect().Model(cm).Where("user_uuid = ? AND club_uuid = ?", userUUID, clubUUID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetClubMembership: %w", err)
	}
	return cm, nil
}

func (r *Impl) GetClubMembershipsByUserUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*ClubMembership, error) {
	if db == nil {
		db = r.db
	}
	var memberships []*ClubMembership
	err := db.NewSelect().
		Model(&memberships).
		Where("user_uuid = ?", userUUID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetClubMembershipsByUserUUID: %w", err)
	}
	return memberships, nil
}

func (r *Impl) GetClubMembershipsByUserUUIDs(ctx context.Context, db bun.IDB, userUUIDs []uuid.UUID) ([]*ClubMembership, error) {
	if db == nil {
		db = r.db
	}
	if len(userUUIDs) == 0 {
		return []*ClubMembership{}, nil
	}
	var memberships []*ClubMembership
	err := db.NewSelect().
		Model(&memberships).
		Where("user_uuid IN (?)", bun.In(userUUIDs)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetClubMembershipsByUserUUIDs: %w", err)
	}
	return memberships, nil
}

func (r *Impl) UpsertClubMembership(ctx context.Context, db bun.IDB, membership *ClubMembership) error {
	if db == nil {
		db = r.db
	}
	now := time.Now().UTC()
	membership.UpdatedAt = now
	_, err := db.NewInsert().
		Model(membership).
		On("CONFLICT (user_uuid, club_uuid) DO UPDATE").
		Set("display_name = EXCLUDED.display_name").
		Set("avatar_url = EXCLUDED.avatar_url").
		Set("role = EXCLUDED.role").
		Set("synced_at = EXCLUDED.synced_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.UpsertClubMembership: %w", err)
	}
	return nil
}

func (r *Impl) GetClubMembershipByExternalID(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*ClubMembership, error) {
	if db == nil {
		db = r.db
	}
	cm := &ClubMembership{}
	err := db.NewSelect().Model(cm).Where("external_id = ? AND club_uuid = ?", externalID, clubUUID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetClubMembershipByExternalID: %w", err)
	}
	return cm, nil
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
		ColumnExpr("u.*"). // Model is aliased to 'u'
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("u.user_id = ?", userID).
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
		Scan(ctx, uwm)
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
		Scan(ctx, uwm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.FindByUDiscName: %w", err)
	}
	return uwm, nil
}

func (r *Impl) GetUsersByUDiscNames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, names []string) ([]UserWithMembership, error) {
	if db == nil {
		db = r.db
	}
	if len(names) == 0 {
		return nil, nil
	}
	var results []UserWithMembership
	err := db.NewSelect().
		Model((*User)(nil)).
		ColumnExpr("u.*").
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_name) IN (?)", bun.In(names)).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx, &results)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetUsersByUDiscNames: %w", err)
	}
	return results, nil
}

func (r *Impl) GetUsersByUDiscUsernames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, usernames []string) ([]UserWithMembership, error) {
	if db == nil {
		db = r.db
	}
	if len(usernames) == 0 {
		return nil, nil
	}
	var results []UserWithMembership
	err := db.NewSelect().
		Model((*User)(nil)).
		ColumnExpr("u.*").
		ColumnExpr("gm.role").
		ColumnExpr("gm.joined_at").
		Join("JOIN guild_memberships AS gm ON u.user_id = gm.user_id").
		Where("LOWER(u.udisc_username) IN (?)", bun.In(usernames)).
		Where("gm.guild_id = ?", guildID).
		Scan(ctx, &results)
	if err != nil {
		return nil, fmt.Errorf("userdb.GetUsersByUDiscUsernames: %w", err)
	}
	return results, nil
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
		userIDs[i] = u.GetUserID()
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
		if m, ok := mMap[u.GetUserID()]; ok {
			results = append(results, &UserWithMembership{
				User:     u,
				Role:     m.Role,
				JoinedAt: m.JoinedAt,
			})
		}
	}

	return results, nil
}

// --- REFRESH TOKEN METHODS ---

func (r *Impl) SaveRefreshToken(ctx context.Context, db bun.IDB, token *RefreshToken) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewInsert().Model(token).Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.SaveRefreshToken: %w", err)
	}
	return nil
}

func (r *Impl) GetRefreshToken(ctx context.Context, db bun.IDB, hash string) (*RefreshToken, error) {
	if db == nil {
		db = r.db
	}
	token := &RefreshToken{}
	err := db.NewSelect().
		Model(token).
		Where("hash = ?", hash).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetRefreshToken: %w", err)
	}
	return token, nil
}

func (r *Impl) RevokeRefreshToken(ctx context.Context, db bun.IDB, hash string) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewUpdate().
		Model((*RefreshToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", time.Now()).
		Where("hash = ?", hash).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.RevokeRefreshToken: %w", err)
	}
	return nil
}

func (r *Impl) RevokeAllUserTokens(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewUpdate().
		Model((*RefreshToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", time.Now()).
		Where("user_uuid = ?", userUUID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.RevokeAllUserTokens: %w", err)
	}
	return nil
}

// --- HELPERS ---

func normalizeNullablePointer(val *string) *string {
	if val == nil || *val == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*val))
	return &normalized
}

// --- MAGIC LINK METHODS ---

func (r *Impl) SaveMagicLink(ctx context.Context, db bun.IDB, link *MagicLink) error {
	if db == nil {
		db = r.db
	}
	if _, err := db.NewInsert().Model(link).Exec(ctx); err != nil {
		return fmt.Errorf("userdb.SaveMagicLink: %w", err)
	}
	return nil
}

func (r *Impl) GetMagicLink(ctx context.Context, db bun.IDB, token string) (*MagicLink, error) {
	if db == nil {
		db = r.db
	}
	ml := &MagicLink{}
	err := db.NewSelect().Model(ml).Where("token = ?", token).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetMagicLink: %w", err)
	}
	return ml, nil
}

func (r *Impl) MarkMagicLinkUsed(ctx context.Context, db bun.IDB, token string) error {
	if db == nil {
		db = r.db
	}
	now := time.Now().UTC()
	res, err := db.NewUpdate().
		Model((*MagicLink)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Where("token = ?", token).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("userdb.MarkMagicLinkUsed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}
