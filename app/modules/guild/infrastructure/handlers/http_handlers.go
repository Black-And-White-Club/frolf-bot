package guildhandlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type HTTPHandlers struct {
	service guildservice.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

func NewHTTPHandlers(
	service guildservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) *HTTPHandlers {
	return &HTTPHandlers{
		service: service,
		logger:  logger,
		tracer:  tracer,
	}
}

// HandleGetEntitlements GET /api/guilds/{club_uuid}/entitlements
func (h *HTTPHandlers) HandleGetEntitlements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clubUUIDStr := chi.URLParam(r, "club_uuid")
	clubUUID, err := uuid.Parse(clubUUIDStr)
	if err != nil {
		http.Error(w, "invalid club_uuid", http.StatusBadRequest)
		return
	}

	entitlements, err := h.service.ResolveClubEntitlements(ctx, clubUUID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to resolve entitlements", "error", err)
		http.Error(w, "failed to resolve entitlements", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entitlements)
}

// HandleGrantFeatureAccess POST /api/admin/guilds/{club_uuid}/features/{feature_key}/grant
func (h *HTTPHandlers) HandleGrantFeatureAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clubUUIDStr := chi.URLParam(r, "club_uuid")
	clubUUID, err := uuid.Parse(clubUUIDStr)
	if err != nil {
		http.Error(w, "invalid club_uuid", http.StatusBadRequest)
		return
	}

	featureKey := guildtypes.ClubFeatureKey(chi.URLParam(r, "feature_key"))

	var payload struct {
		Reason    string     `json:"reason"`
		ExpiresAt *time.Time `json:"expires_at"`
		ActorUUID string     `json:"actor_uuid"` // usually from auth context
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req := guildservice.GrantAccessRequest{
		ClubUUID:   clubUUID,
		FeatureKey: featureKey,
		Reason:     payload.Reason,
		ExpiresAt:  payload.ExpiresAt,
		ActorUUID:  payload.ActorUUID, // fall back to payload if auth middleware isn't setting it yet
	}

	// Optional: extract actor UUID from auth token in context if it exists
	// actorID := auth.GetUserID(ctx)
	// if actorID != "" { req.ActorUUID = actorID }

	if err := h.service.GrantFeatureAccess(ctx, req); err != nil {
		h.logger.ErrorContext(ctx, "failed to grant feature access", "error", err)
		http.Error(w, "failed to grant feature access", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleRevokeFeatureAccess POST /api/admin/guilds/{club_uuid}/features/{feature_key}/revoke
func (h *HTTPHandlers) HandleRevokeFeatureAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clubUUIDStr := chi.URLParam(r, "club_uuid")
	clubUUID, err := uuid.Parse(clubUUIDStr)
	if err != nil {
		http.Error(w, "invalid club_uuid", http.StatusBadRequest)
		return
	}

	featureKey := guildtypes.ClubFeatureKey(chi.URLParam(r, "feature_key"))

	var payload struct {
		Reason    string `json:"reason"`
		ActorUUID string `json:"actor_uuid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req := guildservice.RevokeAccessRequest{
		ClubUUID:   clubUUID,
		FeatureKey: featureKey,
		Reason:     payload.Reason,
		ActorUUID:  payload.ActorUUID,
	}

	if err := h.service.RevokeFeatureAccess(ctx, req); err != nil {
		h.logger.ErrorContext(ctx, "failed to revoke feature access", "error", err)
		http.Error(w, "failed to revoke feature access", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetFeatureAccessAudit GET /api/admin/guilds/{club_uuid}/features/{feature_key}/audit
func (h *HTTPHandlers) HandleGetFeatureAccessAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clubUUIDStr := chi.URLParam(r, "club_uuid")
	clubUUID, err := uuid.Parse(clubUUIDStr)
	if err != nil {
		http.Error(w, "invalid club_uuid", http.StatusBadRequest)
		return
	}

	featureKey := guildtypes.ClubFeatureKey(chi.URLParam(r, "feature_key"))

	records, err := h.service.GetFeatureAccessAudit(ctx, clubUUID, featureKey)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get feature access audit", "error", err)
		http.Error(w, "failed to retrieve audit records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"audit_records": records})
}
