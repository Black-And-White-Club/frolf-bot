package guildservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ResolveClubEntitlements resolves current feature access for a club UUID.
func (s *GuildService) ResolveClubEntitlements(ctx context.Context, clubUUID uuid.UUID) (guildtypes.ResolvedClubEntitlements, error) {
	guildConfig := new(guilddb.GuildConfig)
	err := s.db.NewSelect().Model(guildConfig).Where("uuid = ?", clubUUID).Scan(ctx)
	if err != nil {
		return guildtypes.ResolvedClubEntitlements{}, fmt.Errorf("find guild by club uuid: %w", err)
	}

	return s.repo.ResolveEntitlements(ctx, s.db, guildConfig.GuildID)
}

// ResolveClubFeature resolves a specific feature access for a club UUID.
func (s *GuildService) ResolveClubFeature(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error) {
	entitlements, err := s.ResolveClubEntitlements(ctx, clubUUID)
	if err != nil {
		return guildtypes.ClubFeatureAccess{}, err
	}

	feature, ok := entitlements.Feature(featureKey)
	if !ok {
		return guildtypes.ClubFeatureAccess{
			Key:    featureKey,
			State:  guildtypes.FeatureAccessStateDisabled,
			Source: guildtypes.FeatureAccessSourceNone,
			Reason: "not configured",
		}, nil
	}

	return feature, nil
}

// ResolveClubFeatureByGuildID resolves a specific feature access for a guild ID.
func (s *GuildService) ResolveClubFeatureByGuildID(ctx context.Context, guildID sharedtypes.GuildID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error) {
	entitlements, err := s.repo.ResolveEntitlements(ctx, s.db, guildID)
	if err != nil {
		return guildtypes.ClubFeatureAccess{}, fmt.Errorf("resolve entitlements: %w", err)
	}

	feature, ok := entitlements.Feature(featureKey)
	if !ok {
		return guildtypes.ClubFeatureAccess{
			Key:    featureKey,
			State:  guildtypes.FeatureAccessStateDisabled,
			Source: guildtypes.FeatureAccessSourceNone,
			Reason: "not configured",
		}, nil
	}

	return feature, nil
}

// GrantFeatureAccess grants access to a feature for a club.
//
// The feature override, audit record, and the guild.feature_access.updated.v1
// outbox event are committed in a single transaction so no event is lost even if
// the process crashes between the DB write and message publication.
func (s *GuildService) GrantFeatureAccess(ctx context.Context, req GrantAccessRequest) error {
	guildConfig := new(guilddb.GuildConfig)
	if err := s.db.NewSelect().Model(guildConfig).Where("uuid = ?", req.ClubUUID).Scan(ctx); err != nil {
		return fmt.Errorf("find guild by club uuid: %w", err)
	}

	override := &guilddb.ClubFeatureOverride{
		ClubUUID:   req.ClubUUID,
		FeatureKey: string(req.FeatureKey),
		State:      string(guildtypes.FeatureAccessStateEnabled),
		Reason:     req.Reason,
		ExpiresAt:  req.ExpiresAt,
		UpdatedBy:  req.ActorUUID,
	}

	audit := &guilddb.ClubFeatureAccessAudit{
		ClubUUID:   req.ClubUUID,
		GuildID:    string(guildConfig.GuildID),
		FeatureKey: string(req.FeatureKey),
		State:      string(guildtypes.FeatureAccessStateEnabled),
		Source:     string(guildtypes.FeatureAccessSourceManualAllow),
		Reason:     req.Reason,
		UpdatedBy:  req.ActorUUID,
		ExpiresAt:  req.ExpiresAt,
	}

	return s.upsertOverrideWithOutbox(ctx, guildConfig.GuildID, req.ClubUUID, override, audit)
}

// RevokeFeatureAccess revokes access to a feature for a club.
//
// Same atomic outbox guarantee as GrantFeatureAccess.
func (s *GuildService) RevokeFeatureAccess(ctx context.Context, req RevokeAccessRequest) error {
	guildConfig := new(guilddb.GuildConfig)
	if err := s.db.NewSelect().Model(guildConfig).Where("uuid = ?", req.ClubUUID).Scan(ctx); err != nil {
		return fmt.Errorf("find guild by club uuid: %w", err)
	}

	override := &guilddb.ClubFeatureOverride{
		ClubUUID:   req.ClubUUID,
		FeatureKey: string(req.FeatureKey),
		State:      string(guildtypes.FeatureAccessStateDisabled),
		Reason:     req.Reason,
		UpdatedBy:  req.ActorUUID,
	}

	audit := &guilddb.ClubFeatureAccessAudit{
		ClubUUID:   req.ClubUUID,
		GuildID:    string(guildConfig.GuildID),
		FeatureKey: string(req.FeatureKey),
		State:      string(guildtypes.FeatureAccessStateDisabled),
		Source:     string(guildtypes.FeatureAccessSourceManualDeny),
		Reason:     req.Reason,
		UpdatedBy:  req.ActorUUID,
	}

	return s.upsertOverrideWithOutbox(ctx, guildConfig.GuildID, req.ClubUUID, override, audit)
}

// upsertOverrideWithOutbox executes the feature-override upsert, audit insert,
// and outbox-event insert as a single atomic database transaction.
//
// Publishing is NOT done here; the OutboxForwarder worker picks up the row and
// publishes guild.feature_access.updated.v1 after the transaction commits.
func (s *GuildService) upsertOverrideWithOutbox(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	clubUUID uuid.UUID,
	override *guilddb.ClubFeatureOverride,
	audit *guilddb.ClubFeatureAccessAudit,
) error {
	if s.db == nil {
		// Test path (no real DB): delegate to repo directly without outbox.
		if err := s.repo.UpsertFeatureOverride(ctx, nil, override, audit); err != nil {
			return fmt.Errorf("upsert feature override: %w", err)
		}
		return nil
	}

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// UpsertFeatureOverride creates a savepoint internally when called with a
		// bun.Tx; this is safe in PostgreSQL.
		if err := s.repo.UpsertFeatureOverride(ctx, tx, override, audit); err != nil {
			return fmt.Errorf("upsert feature override: %w", err)
		}

		// Resolve current entitlements within the same snapshot so the event
		// payload reflects the committed state.
		entitlements, err := s.repo.ResolveEntitlements(ctx, tx, guildID)
		if err != nil {
			return fmt.Errorf("resolve entitlements for outbox: %w", err)
		}

		payload := &guildevents.GuildFeatureAccessUpdatedPayloadV1{
			GuildID:      guildID,
			Entitlements: entitlements,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}

		if err := s.repo.InsertOutboxEvent(ctx, tx, guildevents.GuildFeatureAccessUpdatedV1, payloadBytes); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}

		return nil
	})
}

// GetFeatureAccessAudit retrieves the audit history for a club's feature.
func (s *GuildService) GetFeatureAccessAudit(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) ([]FeatureAccessAuditRecord, error) {
	audits, err := s.repo.ListFeatureAccessAudit(ctx, s.db, clubUUID.String(), string(featureKey))
	if err != nil {
		return nil, fmt.Errorf("list feature access audit: %w", err)
	}

	records := make([]FeatureAccessAuditRecord, len(audits))
	for i, a := range audits {
		records[i] = FeatureAccessAuditRecord{
			ID:         a.ID,
			ClubUUID:   a.ClubUUID,
			GuildID:    a.GuildID,
			FeatureKey: a.FeatureKey,
			State:      a.State,
			Source:     a.Source,
			Reason:     a.Reason,
			UpdatedBy:  a.UpdatedBy,
			ExpiresAt:  a.ExpiresAt,
			CreatedAt:  a.CreatedAt,
		}
	}

	return records, nil
}
