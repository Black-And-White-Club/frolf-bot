package authhandlers

import (
	"net/http"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	"github.com/go-chi/chi/v5"
)

const oauthStateCookie = "oauth_state"
const linkModeCookie = "link_mode"

// HandleHTTPOAuthLogin begins the OAuth2 authorization code flow.
// It generates a CSRF state token, stores it in a short-lived HttpOnly cookie,
// and redirects the client to the provider's authorization page.
//
// GET /api/auth/{provider}/login
func (h *AuthHandlers) HandleHTTPOAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Clear any stale link_mode cookie to ensure we perform a fresh login, not a link.
	http.SetCookie(w, &http.Cookie{
		Name:     linkModeCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

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

// HandleHTTPOAuthLinkInitiate begins the OAuth2 flow for linking an additional provider
// to an existing authenticated account. Requires a valid refresh_token cookie.
//
// GET /api/auth/{provider}/link
func (h *AuthHandlers) HandleHTTPOAuthLinkInitiate(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Require existing session.
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, h.pwaBaseURL+"/auth/signin", http.StatusFound)
		return
	}

	redirectURL, state, err := h.service.InitiateOAuthLogin(r.Context(), provider)
	if err != nil {
		h.httpError(w, r, "unsupported provider", http.StatusBadRequest, err)
		return
	}

	// Store CSRF state cookie (5 minutes).
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	// Mark this flow as a link (not a new login) so the callback branches correctly.
	http.SetCookie(w, &http.Cookie{
		Name:     linkModeCookie,
		Value:    "1",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   360,
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

	// Link-mode branch: link a new provider to the existing account.
	linkCookie, _ := r.Cookie(linkModeCookie)
	if linkCookie != nil && linkCookie.Value == "1" {
		// Clear the link_mode cookie.
		http.SetCookie(w, &http.Cookie{
			Name:     linkModeCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   h.secureCookies,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		refreshCookie, err := r.Cookie(RefreshTokenCookie)
		if err != nil || refreshCookie.Value == "" {
			http.Redirect(w, r, h.pwaBaseURL+"/account?error=link_failed", http.StatusFound)
			return
		}

		if err := h.service.LinkIdentityToUser(r.Context(), refreshCookie.Value, provider, code, stateParam); err != nil {
			h.logger.WarnContext(r.Context(), "Failed to link identity",
				attr.String("provider", provider),
				attr.Error(err),
			)
			http.Redirect(w, r, h.pwaBaseURL+"/account?error=link_failed", http.StatusFound)
			return
		}

		http.Redirect(w, r, h.pwaBaseURL+"/account", http.StatusFound)
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
