package authhandlers

import (
	"net/http"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/go-chi/chi/v5"
)

const oauthStateCookie = "oauth_state"

// HandleHTTPOAuthLogin begins the OAuth2 authorization code flow.
// It generates a CSRF state token, stores it in a short-lived HttpOnly cookie,
// and redirects the client to the provider's authorization page.
//
// GET /api/auth/{provider}/login
func (h *AuthHandlers) HandleHTTPOAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	redirectURL, state, err := h.service.InitiateOAuthLogin(r.Context(), provider)
	if err != nil {
		h.httpError(w, r, "unsupported provider", http.StatusBadRequest, err)
		return
	}

	// Store CSRF state in a short-lived HttpOnly cookie (5 minutes).
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// HandleHTTPOAuthCallback completes the OAuth2 authorization code flow.
// It validates the CSRF state cookie, exchanges the code for user identity,
// creates or logs in the user, and sets a refresh token cookie.
//
// GET /api/auth/{provider}/callback
func (h *AuthHandlers) HandleHTTPOAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Validate CSRF: compare state cookie to query param.
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value == "" {
		h.httpError(w, r, "missing oauth state", http.StatusBadRequest, err)
		return
	}
	stateParam := r.URL.Query().Get("state")
	if stateParam == "" || stateParam != stateCookie.Value {
		h.logger.WarnContext(r.Context(), "OAuth state mismatch",
			attr.String("provider", provider),
		)
		h.httpError(w, r, "invalid oauth state", http.StatusBadRequest, nil)
		return
	}

	// Clear the state cookie immediately after validation.
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		h.httpError(w, r, "missing authorization code", http.StatusBadRequest, nil)
		return
	}

	resp, err := h.service.HandleOAuthCallback(r.Context(), provider, code, stateParam)
	if err != nil {
		h.logger.WarnContext(r.Context(), "OAuth callback failed",
			attr.String("provider", provider),
			attr.Error(err),
		)
		h.httpError(w, r, "authentication failed", http.StatusUnauthorized, err)
		return
	}

	// Set the long-lived session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    resp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(authservice.RefreshTokenExpiry),
	})

	// Redirect to PWA.
	http.Redirect(w, r, h.pwaBaseURL, http.StatusFound)
}

// HandleHTTPOAuthLink links an additional OAuth provider to an existing authenticated account.
// The caller must already hold a valid refresh_token cookie.
//
// POST /api/auth/{provider}/link
func (h *AuthHandlers) HandleHTTPOAuthLink(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil || cookie.Value == "" {
		h.httpError(w, r, "unauthorized", http.StatusUnauthorized, err)
		return
	}

	// Validate CSRF state.
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value == "" {
		h.httpError(w, r, "missing oauth state", http.StatusBadRequest, err)
		return
	}
	stateParam := r.URL.Query().Get("state")
	if stateParam == "" || stateParam != stateCookie.Value {
		h.httpError(w, r, "invalid oauth state", http.StatusBadRequest, nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		h.httpError(w, r, "missing authorization code", http.StatusBadRequest, nil)
		return
	}

	if err := h.service.LinkIdentityToUser(r.Context(), cookie.Value, provider, code, stateParam); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to link identity",
			attr.String("provider", provider),
			attr.Error(err),
		)
		h.httpError(w, r, "failed to link identity", http.StatusUnprocessableEntity, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
