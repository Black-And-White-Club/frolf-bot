package authhandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	RefreshTokenCookie = "refresh_token"
)

func (h *AuthHandlers) httpError(w http.ResponseWriter, r *http.Request, message string, code int, err error) {
	h.logger.WarnContext(r.Context(), message, attr.Error(err), attr.Int("code", code))
	http.Error(w, message, code)
}

func (h *AuthHandlers) HandleHTTPLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		h.httpError(w, r, "method not allowed", http.StatusMethodNotAllowed, nil)
		return
	}

	token, err := readMagicLinkToken(w, r)
	if token == "" {
		h.httpError(w, r, "missing token", http.StatusBadRequest, err)
		return
	}

	resp, err := h.service.LoginUser(ctx, token)
	if err != nil {
		h.httpError(w, r, "authentication failed", http.StatusUnauthorized, err)
		return
	}

	// Set HttpOnly cookie for session continuity after token exchange.
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(authservice.RefreshTokenExpiry),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user_uuid": resp.UserUUID})
}

type loginRequest struct {
	Token string `json:"token"`
}

func readMagicLinkToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if token := strings.TrimSpace(r.Header.Get("X-Magic-Link-Token")); token != "" {
		return token, nil
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		req := loginRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", fmt.Errorf("invalid json body: %w", err)
		}
		return strings.TrimSpace(req.Token), nil
	}

	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("invalid form body: %w", err)
	}
	return strings.TrimSpace(r.FormValue("token")), nil
}

func (h *AuthHandlers) HandleHTTPTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, err)
		return
	}

	activeClubUUID := r.URL.Query().Get("active_club")

	resp, err := h.service.GetTicket(ctx, cookie.Value, activeClubUUID)
	if err != nil {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, err)
		return
	}

	// Update cookie with rotated token
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(authservice.RefreshTokenExpiry),
	})

	// Dispatch profile/role sync requests (non-blocking, best-effort).
	for _, sr := range resp.SyncRequests {
		syncPayload := userevents.UserProfileSyncRequestPayloadV1{
			UserID:  sharedtypes.DiscordID(sr.UserID),
			GuildID: sharedtypes.GuildID(sr.GuildID),
		}
		payloadBytes, _ := json.Marshal(syncPayload)
		syncMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
		syncMsg.Metadata.Set("topic", userevents.UserProfileSyncRequestTopicV1)
		syncMsg.Metadata.Set("user_id", sr.UserID)
		syncMsg.Metadata.Set("guild_id", sr.GuildID)

		if err := h.eventBus.Publish(userevents.UserProfileSyncRequestTopicV1, syncMsg); err != nil {
			h.logger.WarnContext(ctx, "Failed to publish profile sync request",
				attr.Error(err),
				attr.String("user_id", sr.UserID),
			)
		} else {
			h.logger.InfoContext(ctx, "Published profile sync request",
				attr.String("user_id", sr.UserID),
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"ticket": resp.NATSToken,
	})
}

func (h *AuthHandlers) HandleHTTPLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, _ := r.Cookie(RefreshTokenCookie)
	if cookie != nil {
		_ = h.service.LogoutUser(ctx, cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}
