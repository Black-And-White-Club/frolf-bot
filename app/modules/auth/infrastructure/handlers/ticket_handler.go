package authhandlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

const (
	RefreshTokenCookie = "refresh_token"
	RefreshTokenExpiry = 30 * 24 * time.Hour
)

func (h *AuthHandlers) httpError(w http.ResponseWriter, r *http.Request, message string, code int, err error) {
	h.logger.WarnContext(r.Context(), message, attr.Error(err), attr.Int("code", code))
	http.Error(w, message, code)
}

func (h *AuthHandlers) HandleHTTPLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := r.URL.Query().Get("t")
	if token == "" {
		h.httpError(w, r, "missing token", http.StatusBadRequest, nil)
		return
	}

	resp, err := h.service.LoginUser(ctx, token)
	if err != nil {
		h.httpError(w, r, "authentication failed", http.StatusUnauthorized, err)
		return
	}

	// Set HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(RefreshTokenExpiry),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user_uuid": resp.UserUUID})
}

func (h *AuthHandlers) HandleHTTPTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, err)
		return
	}

	resp, err := h.service.GetTicket(ctx, cookie.Value)
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
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(RefreshTokenExpiry),
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
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}
