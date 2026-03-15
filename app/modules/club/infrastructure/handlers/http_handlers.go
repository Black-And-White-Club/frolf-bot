package clubhandlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

const refreshTokenCookie = "refresh_token"

// HTTPHandlers implements the HTTP-specific club endpoints.
type HTTPHandlers struct {
	service  clubservice.Service
	userRepo userdb.Repository
	logger   *slog.Logger
	tracer   trace.Tracer
}

// NewHTTPHandlers creates new club HTTP handlers.
func NewHTTPHandlers(
	service clubservice.Service,
	userRepo userdb.Repository,
	logger *slog.Logger,
	tracer trace.Tracer,
) *HTTPHandlers {
	return &HTTPHandlers{
		service:  service,
		userRepo: userRepo,
		logger:   logger,
		tracer:   tracer,
	}
}

// resolveUserUUID reads the refresh_token cookie, looks up the token in the DB,
// and returns the associated user UUID.
func (h *HTTPHandlers) resolveUserUUID(r *http.Request) (uuid.UUID, error) {
	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || cookie.Value == "" {
		return uuid.Nil, fmt.Errorf("missing session")
	}
	hash := sha256hex(cookie.Value)
	token, err := h.userRepo.GetRefreshToken(r.Context(), nil, hash)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid session")
	}
	if token.Revoked {
		return uuid.Nil, fmt.Errorf("session revoked")
	}
	return token.UserUUID, nil
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// HandleGetClub returns club information by UUID.
// GET /api/clubs/{uuid}
func (h *HTTPHandlers) HandleGetClub(w http.ResponseWriter, r *http.Request) {
	clubUUID, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid club uuid")
		return
	}

	info, err := h.service.GetClub(r.Context(), clubUUID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "GetClub failed", slog.String("error", err.Error()))
		httpError(w, http.StatusNotFound, "club not found")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// HandleGetSuggestions returns club suggestions based on the user's Discord guilds.
// GET /api/clubs/suggestions
func (h *HTTPHandlers) HandleGetSuggestions(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	suggestions, err := h.service.GetClubSuggestions(r.Context(), userUUID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "GetClubSuggestions failed", slog.String("error", err.Error()))
		httpError(w, http.StatusInternalServerError, "failed to get suggestions")
		return
	}

	writeJSON(w, http.StatusOK, suggestions)
}

// HandleJoinClub adds the user to a club via Discord guild membership.
// POST /api/clubs/join
func (h *HTTPHandlers) HandleJoinClub(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		ClubUUID string `json:"club_uuid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ClubUUID == "" {
		httpError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	clubUUID, err := uuid.Parse(body.ClubUUID)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid club_uuid")
		return
	}

	if err := h.service.JoinClub(r.Context(), userUUID, clubUUID); err != nil {
		h.logger.WarnContext(r.Context(), "JoinClub failed", slog.String("error", err.Error()))
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleCreateInvite creates an invite code for a club.
// POST /api/clubs/{uuid}/invites
func (h *HTTPHandlers) HandleCreateInvite(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid club uuid")
		return
	}

	var req clubservice.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	info, err := h.service.CreateInvite(r.Context(), userUUID, clubUUID, req)
	if err != nil {
		h.logger.WarnContext(r.Context(), "CreateInvite failed", slog.String("error", err.Error()))
		status := http.StatusInternalServerError
		if err.Error()[:8] == "forbidden" {
			status = http.StatusForbidden
		}
		httpError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, info)
}

// HandleListInvites lists active invite codes for a club.
// GET /api/clubs/{uuid}/invites
func (h *HTTPHandlers) HandleListInvites(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid club uuid")
		return
	}

	invites, err := h.service.ListInvites(r.Context(), userUUID, clubUUID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "ListInvites failed", slog.String("error", err.Error()))
		status := http.StatusInternalServerError
		if err.Error()[:8] == "forbidden" {
			status = http.StatusForbidden
		}
		httpError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, invites)
}

// HandleRevokeInvite revokes an invite code.
// DELETE /api/clubs/{uuid}/invites/{code}
func (h *HTTPHandlers) HandleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid club uuid")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		httpError(w, http.StatusBadRequest, "missing invite code")
		return
	}

	if err := h.service.RevokeInvite(r.Context(), userUUID, clubUUID, code); err != nil {
		h.logger.WarnContext(r.Context(), "RevokeInvite failed", slog.String("error", err.Error()))
		status := http.StatusInternalServerError
		if err.Error()[:8] == "forbidden" {
			status = http.StatusForbidden
		} else if err.Error() == "invite not found" {
			status = http.StatusNotFound
		}
		httpError(w, status, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetInvitePreview returns a public preview of a club for a given invite code.
// GET /api/clubs/preview?code=ABC123
func (h *HTTPHandlers) HandleGetInvitePreview(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		httpError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	preview, err := h.service.GetInvitePreview(r.Context(), code)
	if err != nil {
		h.logger.WarnContext(r.Context(), "GetInvitePreview failed", slog.String("error", err.Error()))
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

// HandleJoinByCode uses an invite code to join a club.
// POST /api/clubs/join-by-code
func (h *HTTPHandlers) HandleJoinByCode(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		httpError(w, http.StatusBadRequest, "missing code")
		return
	}

	if err := h.service.JoinByCode(r.Context(), userUUID, body.Code); err != nil {
		h.logger.WarnContext(r.Context(), "JoinByCode failed", slog.String("error", err.Error()))
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
