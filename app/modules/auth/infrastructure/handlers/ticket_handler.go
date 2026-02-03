package authhandlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

const (
	RefreshTokenCookie = "refresh_token"
)

func (h *AuthHandlers) HandleHTTPLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := r.URL.Query().Get("t")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	resp, err := h.service.LoginUser(ctx, token)
	if err != nil {
		h.logger.ErrorContext(ctx, "HTTP Login failed", attr.Error(err))
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	// Set HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Should be true in production
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user_uuid": resp.UserUUID})
}

func (h *AuthHandlers) HandleHTTPTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.service.GetTicket(ctx, cookie.Value)
	if err != nil {
		h.logger.WarnContext(ctx, "Ticket request failed", attr.Error(err))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Update cookie with rotated token
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

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
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}
