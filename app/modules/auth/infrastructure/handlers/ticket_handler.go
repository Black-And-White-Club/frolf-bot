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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user_uuid": resp.UserUUID})
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

	rawToken, fromHeader := extractRefreshToken(r)
	if rawToken == "" {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, nil)
		return
	}

	activeClubUUID := r.URL.Query().Get("active_club")

	resp, err := h.service.GetTicket(ctx, rawToken, activeClubUUID)
	if err != nil {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, err)
		return
	}

	// For cookie-based callers (PWA), rotate the cookie as before.
	if !fromHeader {
		http.SetCookie(w, &http.Cookie{
			Name:     RefreshTokenCookie,
			Value:    resp.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.secureCookies,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(authservice.RefreshTokenExpiry),
		})
	}

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

	// For bearer callers (Activity), include the rotated refresh_token in the
	// response body so the client can update its in-memory token.
	response := map[string]string{"ticket": resp.NATSToken}
	if fromHeader {
		response["refresh_token"] = resp.RefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// extractRefreshToken returns the raw refresh token and whether it came from
// an Authorization header (true) or a cookie (false).
func extractRefreshToken(r *http.Request) (token string, fromHeader bool) {
	if cookie, err := r.Cookie(RefreshTokenCookie); err == nil && cookie.Value != "" {
		if isValidTokenFormat(cookie.Value) {
			return cookie.Value, false
		}
	}
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		t := auth[7:]
		if isValidTokenFormat(t) {
			return t, true
		}
	}
	return "", false
}

func (h *AuthHandlers) HandleHTTPLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawToken, _ := extractRefreshToken(r)
	if rawToken != "" {
		_ = h.service.LogoutUser(ctx, rawToken)
	}

	// Clear cookie (no-op for bearer-only callers, harmless to send).
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
