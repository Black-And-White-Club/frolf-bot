package guilddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new guild repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// UpdateFields represents the updateable fields of a guild config.
// Pointer fields distinguish "not provided" (nil) from "set to zero value".
// This enables clean partial updates without full object replacement.
type UpdateFields struct {
	SignupChannelID      *string
	SignupMessageID      *string
	EventChannelID       *string
	LeaderboardChannelID *string
	UserRoleID           *string
	EditorRoleID         *string
	AdminRoleID          *string
	SignupEmoji          *string
	AutoSetupCompleted   *bool
	SetupCompletedAt     *int64 // Unix nano timestamp
}

// IsEmpty reports whether any fields are set for update.
func (u *UpdateFields) IsEmpty() bool {
	if u == nil {
		return true
	}
	return u.SignupChannelID == nil &&
		u.SignupMessageID == nil &&
		u.EventChannelID == nil &&
		u.LeaderboardChannelID == nil &&
		u.UserRoleID == nil &&
		u.EditorRoleID == nil &&
		u.AdminRoleID == nil &&
		u.SignupEmoji == nil &&
		u.AutoSetupCompleted == nil &&
		u.SetupCompletedAt == nil
}

// upsertSetColumns defines fields to overwrite on a conflict (SaveConfig).
// Includes deletion_status/resource_state to handle re-activations.
var upsertSetColumns = []string{
	"signup_channel_id", "signup_message_id", "event_channel_id",
	"leaderboard_channel_id", "user_role_id", "editor_role_id",
	"admin_role_id", "signup_emoji", "auto_setup_completed",
	"setup_completed_at", "is_active", "updated_at",
	"deletion_status", "resource_state",
}

// --- READ METHODS ---

// GetConfig retrieves an active guild configuration by ID.
func (r *Impl) GetConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	if db == nil {
		db = r.db
	}

	model, err := r.selectConfigModel(ctx, db, guildID, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("guilddb.GetConfig: %w", err)
	}

	config := toSharedModel(model)
	entitlements, err := r.ResolveEntitlements(ctx, db, guildID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	config.Entitlements = entitlements

	return config, nil
}

// GetConfigIncludeDeleted retrieves a guild configuration by ID, including inactive ones.
func (r *Impl) GetConfigIncludeDeleted(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	if db == nil {
		db = r.db
	}

	model, err := r.selectConfigModel(ctx, db, guildID, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("guilddb.GetConfigIncludeDeleted: %w", err)
	}

	config := toSharedModel(model)
	entitlements, err := r.ResolveEntitlements(ctx, db, guildID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	config.Entitlements = entitlements

	return config, nil
}

// ResolveEntitlements resolves club-level entitlements from subscription/trial state plus manual overrides.
func (r *Impl) ResolveEntitlements(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
	if db == nil {
		db = r.db
	}

	resolvedGuildID, clubUUID, err := r.resolveLookupIdentifiers(ctx, db, guildID)
	if err != nil {
		return guildtypes.ResolvedClubEntitlements{}, fmt.Errorf("guilddb.ResolveEntitlements: %w", err)
	}

	model, err := r.selectConfigModel(ctx, db, resolvedGuildID, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return guildtypes.ResolvedClubEntitlements{}, ErrNotFound
		}
		return guildtypes.ResolvedClubEntitlements{}, fmt.Errorf("guilddb.ResolveEntitlements: %w", err)
	}

	now := time.Now().UTC()
	entitlements := guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: r.resolveBettingAccess(ctx, db, model, clubUUID, now),
		},
		ResolvedAt: &now,
	}

	return entitlements, nil
}

// --- WRITE METHODS ---

// SaveConfig creates or re-activates a guild configuration.
func (r *Impl) SaveConfig(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error {
	if db == nil {
		db = r.db
	}

	dbModel := toDBModel(config)
	dbModel.IsActive = true
	dbModel.UpdatedAt = time.Now().UTC()
	dbModel.DeletionStatus = "none"

	q := db.NewInsert().
		Model(dbModel).
		On("CONFLICT (guild_id) DO UPDATE")

	// Reuse existing upsert columns logic
	for _, col := range upsertSetColumns {
		q = q.Set("? = EXCLUDED.?", bun.Ident(col), bun.Ident(col))
	}

	if _, err := q.Exec(ctx); err != nil {
		return fmt.Errorf("guilddb.SaveConfig: %w", err)
	}
	return nil
}

// UpdateConfig applies partial updates to an active guild configuration.
func (r *Impl) UpdateConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *UpdateFields) error {
	if updates == nil || updates.IsEmpty() {
		return nil
	}
	if db == nil {
		db = r.db
	}

	q := db.NewUpdate().
		Table("guild_configs").
		Where("guild_id = ? AND is_active = true", guildID)

	// Apply partial updates
	if updates.SignupChannelID != nil {
		q = q.Set("signup_channel_id = ?", *updates.SignupChannelID)
	}
	if updates.SignupMessageID != nil {
		q = q.Set("signup_message_id = ?", *updates.SignupMessageID)
	}
	if updates.EventChannelID != nil {
		q = q.Set("event_channel_id = ?", *updates.EventChannelID)
	}
	if updates.LeaderboardChannelID != nil {
		q = q.Set("leaderboard_channel_id = ?", *updates.LeaderboardChannelID)
	}
	if updates.UserRoleID != nil {
		q = q.Set("user_role_id = ?", *updates.UserRoleID)
	}
	if updates.EditorRoleID != nil {
		q = q.Set("editor_role_id = ?", *updates.EditorRoleID)
	}
	if updates.AdminRoleID != nil {
		q = q.Set("admin_role_id = ?", *updates.AdminRoleID)
	}
	if updates.SignupEmoji != nil {
		q = q.Set("signup_emoji = ?", *updates.SignupEmoji)
	}
	if updates.AutoSetupCompleted != nil {
		q = q.Set("auto_setup_completed = ?", *updates.AutoSetupCompleted)
	}
	if updates.SetupCompletedAt != nil {
		q = q.Set("setup_completed_at = ?", unixNanoToTime(*updates.SetupCompletedAt))
	}

	q = q.Set("updated_at = ?", time.Now().UTC())

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("guilddb.UpdateConfig: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// DeleteConfig performs a soft delete.
// Note: Transaction handling is removed here as it should be managed by the Service.
func (r *Impl) DeleteConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error {
	if db == nil {
		db = r.db
	}

	model := new(GuildConfig)
	if err := db.NewSelect().Model(model).Where("guild_id = ? AND is_active = true", guildID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("guilddb.DeleteConfig select: %w", err)
	}

	// Capture resource state snapshot for cleanup
	rs := &ResourceState{
		SignupChannelID:      model.SignupChannelID,
		SignupMessageID:      model.SignupMessageID,
		EventChannelID:       model.EventChannelID,
		LeaderboardChannelID: model.LeaderboardChannelID,
		UserRoleID:           model.UserRoleID,
		EditorRoleID:         model.EditorRoleID,
		AdminRoleID:          model.AdminRoleID,
		Results:              make(map[string]DeletionResult),
	}

	_, err := db.NewUpdate().
		Table("guild_configs").
		Where("guild_id = ?", guildID).
		Set("resource_state = ?", rs).
		Set("deletion_status = 'pending'").
		Set("is_active = false").
		Set("updated_at = ?", time.Now().UTC()).
		Set("signup_channel_id = NULL, signup_message_id = NULL, event_channel_id = NULL, leaderboard_channel_id = NULL").
		Set("user_role_id = NULL, editor_role_id = NULL, admin_role_id = NULL").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("guilddb.DeleteConfig update: %w", err)
	}
	return nil
}

func (r *Impl) selectConfigModel(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, includeDeleted bool) (*GuildConfig, error) {
	resolvedGuildID, _, err := r.resolveLookupIdentifiers(ctx, db, guildID)
	if err != nil {
		return nil, err
	}

	model := new(GuildConfig)
	query := db.NewSelect().Model(model).Where("guild_id = ?", resolvedGuildID)

	if !includeDeleted {
		query = query.Where("is_active = true")
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return model, nil
}

func (r *Impl) resolveLookupIdentifiers(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (sharedtypes.GuildID, uuid.UUID, error) {
	resolvedGuildID := guildID
	var clubUUID uuid.UUID

	if parsedUUID, err := uuid.Parse(string(guildID)); err == nil {
		clubUUID = parsedUUID

		var discordGuildID string
		if err := db.NewSelect().
			TableExpr("clubs").
			ColumnExpr("discord_guild_id").
			Where("uuid = ?", parsedUUID).
			Scan(ctx, &discordGuildID); err == nil && discordGuildID != "" {
			resolvedGuildID = sharedtypes.GuildID(discordGuildID)
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", uuid.Nil, err
		}
	}

	if clubUUID == uuid.Nil && resolvedGuildID != "" {
		if err := db.NewSelect().
			TableExpr("clubs").
			ColumnExpr("uuid").
			Where("discord_guild_id = ?", resolvedGuildID).
			Scan(ctx, &clubUUID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", uuid.Nil, err
		}
	}

	return resolvedGuildID, clubUUID, nil
}

func (r *Impl) resolveBettingAccess(
	ctx context.Context,
	db bun.IDB,
	model *GuildConfig,
	clubUUID uuid.UUID,
	now time.Time,
) guildtypes.ClubFeatureAccess {
	access := guildtypes.ClubFeatureAccess{
		Key:    guildtypes.ClubFeatureBetting,
		State:  guildtypes.FeatureAccessStateDisabled,
		Source: guildtypes.FeatureAccessSourceNone,
		Reason: "premium subscription required",
	}

	if model == nil {
		return access
	}

	// Manual overrides have highest precedence: manual_deny > manual_allow > subscription/trial.
	// Filter expired overrides at the query level to avoid stale in-memory comparisons.
	if clubUUID != uuid.Nil {
		override := new(ClubFeatureOverride)
		if err := db.NewSelect().
			Model(override).
			Where("club_uuid = ? AND feature_key = ?", clubUUID, string(guildtypes.ClubFeatureBetting)).
			Where("expires_at IS NULL OR expires_at > ?", now).
			Scan(ctx); err == nil {
			access.Reason = override.Reason
			access.ExpiresAt = override.ExpiresAt
			switch {
			case strings.EqualFold(override.State, string(guildtypes.FeatureAccessStateEnabled)):
				access.State = guildtypes.FeatureAccessStateEnabled
				access.Source = guildtypes.FeatureAccessSourceManualAllow
			case strings.EqualFold(override.State, string(guildtypes.FeatureAccessStateFrozen)):
				access.State = guildtypes.FeatureAccessStateFrozen
				access.Source = guildtypes.FeatureAccessSourceManualDeny
			default:
				access.State = guildtypes.FeatureAccessStateDisabled
				access.Source = guildtypes.FeatureAccessSourceManualDeny
			}
			// Active manual override — no need to evaluate subscription or trial.
			return access
		}
	}

	// Evaluate trial and subscription access independently, then pick the best.
	// Precedence (already handled above): manual_deny > manual_allow > subscription/trial > disabled.
	// Within subscription/trial: enabled > frozen > disabled; ties broken by latest ExpiresAt.
	var trialAccess, subAccess *guildtypes.ClubFeatureAccess

	if model.IsTrial {
		a := guildtypes.ClubFeatureAccess{Key: guildtypes.ClubFeatureBetting}
		if model.TrialExpiresAt == nil || model.TrialExpiresAt.After(now) {
			a.State = guildtypes.FeatureAccessStateEnabled
			a.Source = guildtypes.FeatureAccessSourceTrial
			a.Reason = "trial active"
		} else {
			a.State = guildtypes.FeatureAccessStateFrozen
			a.Source = guildtypes.FeatureAccessSourceTrial
			a.Reason = "trial expired; feature is read-only until access is restored"
		}
		a.ExpiresAt = model.TrialExpiresAt
		trialAccess = &a
	}

	if isPremiumSubscriptionTier(model.SubscriptionTier) {
		a := guildtypes.ClubFeatureAccess{Key: guildtypes.ClubFeatureBetting}
		if model.SubscriptionExpiresAt == nil || model.SubscriptionExpiresAt.After(now) {
			a.State = guildtypes.FeatureAccessStateEnabled
			a.Source = guildtypes.FeatureAccessSourceSubscription
			a.Reason = fmt.Sprintf("subscription tier %s active", model.SubscriptionTier)
		} else {
			a.State = guildtypes.FeatureAccessStateFrozen
			a.Source = guildtypes.FeatureAccessSourceSubscription
			a.Reason = fmt.Sprintf("subscription tier %s expired; feature is read-only until access is restored", model.SubscriptionTier)
		}
		a.ExpiresAt = model.SubscriptionExpiresAt
		subAccess = &a
	}

	if best := pickBestAccess(trialAccess, subAccess); best != nil {
		access = *best
	}

	return access
}

// pickBestAccess returns the most permissive of two optional feature accesses.
// Rules: enabled > frozen > disabled; ties broken by latest ExpiresAt (nil = indefinite = best).
func pickBestAccess(a, b *guildtypes.ClubFeatureAccess) *guildtypes.ClubFeatureAccess {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	rankState := func(s guildtypes.FeatureAccessState) int {
		switch s {
		case guildtypes.FeatureAccessStateEnabled:
			return 2
		case guildtypes.FeatureAccessStateFrozen:
			return 1
		default:
			return 0
		}
	}
	ra, rb := rankState(a.State), rankState(b.State)
	if ra != rb {
		if ra > rb {
			return a
		}
		return b
	}
	// Same state: prefer the one with a later (or nil) expiry.
	if a.ExpiresAt == nil {
		return a
	}
	if b.ExpiresAt == nil {
		return b
	}
	if a.ExpiresAt.After(*b.ExpiresAt) {
		return a
	}
	return b
}

func isPremiumSubscriptionTier(tier string) bool {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "premium", "pro", "enterprise", "paid":
		return true
	default:
		return false
	}
}

// =============================================================================
// Model Conversion Helpers
// =============================================================================

// toSharedModel converts the DB model to the shared domain type.
func toSharedModel(cfg *GuildConfig) *guildtypes.GuildConfig {
	if cfg == nil {
		return nil
	}
	return &guildtypes.GuildConfig{
		GuildID:              cfg.GuildID,
		SignupChannelID:      cfg.SignupChannelID,
		SignupMessageID:      cfg.SignupMessageID,
		EventChannelID:       cfg.EventChannelID,
		LeaderboardChannelID: cfg.LeaderboardChannelID,
		UserRoleID:           cfg.UserRoleID,
		EditorRoleID:         cfg.EditorRoleID,
		AdminRoleID:          cfg.AdminRoleID,
		SignupEmoji:          cfg.SignupEmoji,
		AutoSetupCompleted:   cfg.AutoSetupCompleted,
		SetupCompletedAt:     cfg.SetupCompletedAt,
		ResourceState:        toSharedResourceState(cfg.ResourceState),
	}
}

// toDBModel converts the shared domain type to the DB model.
func toDBModel(cfg *guildtypes.GuildConfig) *GuildConfig {
	if cfg == nil {
		return nil
	}
	return &GuildConfig{
		GuildID:              cfg.GuildID,
		SignupChannelID:      cfg.SignupChannelID,
		SignupMessageID:      cfg.SignupMessageID,
		EventChannelID:       cfg.EventChannelID,
		LeaderboardChannelID: cfg.LeaderboardChannelID,
		UserRoleID:           cfg.UserRoleID,
		EditorRoleID:         cfg.EditorRoleID,
		AdminRoleID:          cfg.AdminRoleID,
		SignupEmoji:          cfg.SignupEmoji,
		AutoSetupCompleted:   cfg.AutoSetupCompleted,
		SetupCompletedAt:     cfg.SetupCompletedAt,
		ResourceState:        toDBResourceState(&cfg.ResourceState),
	}
}

// toSharedResourceState converts DB ResourceState to shared type.
func toSharedResourceState(rs *ResourceState) guildtypes.ResourceState {
	if rs == nil {
		return guildtypes.ResourceState{}
	}
	results := make(map[string]guildtypes.DeletionResult, len(rs.Results))
	for k, v := range rs.Results {
		results[k] = guildtypes.DeletionResult{
			Status:    v.Status,
			Error:     v.Error,
			DeletedAt: v.DeletedAt,
		}
	}
	return guildtypes.ResourceState{
		SignupChannelID:      rs.SignupChannelID,
		SignupMessageID:      rs.SignupMessageID,
		EventChannelID:       rs.EventChannelID,
		LeaderboardChannelID: rs.LeaderboardChannelID,
		UserRoleID:           rs.UserRoleID,
		EditorRoleID:         rs.EditorRoleID,
		AdminRoleID:          rs.AdminRoleID,
		Results:              results,
	}
}

// toDBResourceState converts shared ResourceState to DB type.
func toDBResourceState(rs *guildtypes.ResourceState) *ResourceState {
	if rs == nil || rs.IsEmpty() {
		return nil
	}
	results := make(map[string]DeletionResult, len(rs.Results))
	for k, v := range rs.Results {
		results[k] = DeletionResult{
			Status:    v.Status,
			Error:     v.Error,
			DeletedAt: v.DeletedAt,
		}
	}
	return &ResourceState{
		SignupChannelID:      rs.SignupChannelID,
		SignupMessageID:      rs.SignupMessageID,
		EventChannelID:       rs.EventChannelID,
		LeaderboardChannelID: rs.LeaderboardChannelID,
		UserRoleID:           rs.UserRoleID,
		EditorRoleID:         rs.EditorRoleID,
		AdminRoleID:          rs.AdminRoleID,
		Results:              results,
	}
}

// unixNanoToTime converts a unix nano timestamp to *time.Time.
func unixNanoToTime(nano int64) *time.Time {
	if nano == 0 {
		return nil
	}
	t := time.Unix(0, nano).UTC()
	return &t
}

// UpsertFeatureOverride inserts or updates a feature override and creates an audit record.
func (r *Impl) UpsertFeatureOverride(ctx context.Context, db bun.IDB, override *ClubFeatureOverride, audit *ClubFeatureAccessAudit) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(override).
			On("CONFLICT (club_uuid, feature_key) DO UPDATE").
			Set("state = EXCLUDED.state").
			Set("reason = EXCLUDED.reason").
			Set("expires_at = EXCLUDED.expires_at").
			Set("updated_by = EXCLUDED.updated_by").
			Set("updated_at = current_timestamp").
			Exec(ctx); err != nil {
			return fmt.Errorf("upsert feature override: %w", err)
		}

		if _, err := tx.NewInsert().Model(audit).Exec(ctx); err != nil {
			return fmt.Errorf("insert feature access audit: %w", err)
		}

		return nil
	})
}

// DeleteFeatureOverride deletes a feature override and creates an audit record.
func (r *Impl) DeleteFeatureOverride(ctx context.Context, db bun.IDB, clubUUID string, featureKey string, audit *ClubFeatureAccessAudit) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*ClubFeatureOverride)(nil)).
			Where("club_uuid = ? AND feature_key = ?", clubUUID, featureKey).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete feature override: %w", err)
		}

		if _, err := tx.NewInsert().Model(audit).Exec(ctx); err != nil {
			return fmt.Errorf("insert feature access audit: %w", err)
		}

		return nil
	})
}

// ListFeatureAccessAudit retrieves the audit history for a club's feature.
func (r *Impl) ListFeatureAccessAudit(ctx context.Context, db bun.IDB, clubUUID string, featureKey string) ([]ClubFeatureAccessAudit, error) {
	var audits []ClubFeatureAccessAudit
	if err := db.NewSelect().
		Model(&audits).
		Where("club_uuid = ? AND feature_key = ?", clubUUID, featureKey).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list feature access audit: %w", err)
	}
	return audits, nil
}

// GuildOutboxEvent is a transactional outbox row used to reliably deliver
// domain events after the business mutation commits.
type GuildOutboxEvent struct {
	bun.BaseModel `bun:"table:guild_outbox,alias:o"`

	ID          string     `bun:"id,pk,default:gen_random_uuid()"`
	Topic       string     `bun:"topic,notnull"`
	Payload     []byte     `bun:"payload,notnull,type:jsonb"`
	PublishedAt *time.Time `bun:"published_at"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
}

// InsertOutboxEvent inserts a pending outbox event.
// Must be called with a bun.Tx to be atomic with the associated business mutation.
func (r *Impl) InsertOutboxEvent(ctx context.Context, db bun.IDB, topic string, payload []byte) error {
	if db == nil {
		db = r.db
	}
	event := &GuildOutboxEvent{
		Topic:   topic,
		Payload: payload,
	}
	if _, err := db.NewInsert().Model(event).Exec(ctx); err != nil {
		return fmt.Errorf("guilddb.InsertOutboxEvent: %w", err)
	}
	return nil
}

// PollAndLockOutboxEvents returns up to limit unpublished outbox rows,
// acquiring a row-level lock via SELECT … FOR UPDATE SKIP LOCKED so that
// multiple forwarder instances do not double-publish the same event.
func (r *Impl) PollAndLockOutboxEvents(ctx context.Context, db bun.IDB, limit int) ([]GuildOutboxEvent, error) {
	if db == nil {
		db = r.db
	}
	var events []GuildOutboxEvent
	if err := db.NewRaw(
		`SELECT id, topic, payload, published_at, created_at
 FROM guild_outbox
 WHERE published_at IS NULL
 ORDER BY created_at
 LIMIT ?
 FOR UPDATE SKIP LOCKED`,
		limit,
	).Scan(ctx, &events); err != nil {
		return nil, fmt.Errorf("guilddb.PollAndLockOutboxEvents: %w", err)
	}
	return events, nil
}

// MarkOutboxEventPublished sets published_at = NOW() for the given row,
// confirming the event was delivered to the message bus.
func (r *Impl) MarkOutboxEventPublished(ctx context.Context, db bun.IDB, id string) error {
	if db == nil {
		db = r.db
	}
	now := time.Now()
	if _, err := db.NewUpdate().
		TableExpr("guild_outbox").
		Set("published_at = ?", now).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return fmt.Errorf("guilddb.MarkOutboxEventPublished: %w", err)
	}
	return nil
}
