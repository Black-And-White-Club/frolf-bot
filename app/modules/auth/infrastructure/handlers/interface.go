package authhandlers

import (
	"net/http"

	"github.com/nats-io/nats.go"
)

// Handlers defines the interface for auth event handlers.
type Handlers interface {
	// HandleMagicLinkRequest handles incoming magic link requests via NATS.
	HandleMagicLinkRequest(msg *nats.Msg)

	// HandleNATSAuthCallout handles NATS auth callout messages.
	HandleNATSAuthCallout(msg *nats.Msg)

	// HTTP Handlers (magic link + session)
	HandleHTTPLogin(w http.ResponseWriter, r *http.Request)
	HandleHTTPTicket(w http.ResponseWriter, r *http.Request)
	HandleHTTPLogout(w http.ResponseWriter, r *http.Request)

	// HTTP Handlers (OAuth2)
	HandleHTTPOAuthLogin(w http.ResponseWriter, r *http.Request)
	HandleHTTPOAuthCallback(w http.ResponseWriter, r *http.Request)
	HandleHTTPOAuthLinkInitiate(w http.ResponseWriter, r *http.Request)
	HandleHTTPOAuthUnlink(w http.ResponseWriter, r *http.Request)

	// HandleActivityTokenExchange handles the Discord Activity OAuth code exchange.
	// It is a public route — no AuthMiddleware, no cookie required.
	HandleActivityTokenExchange(w http.ResponseWriter, r *http.Request)
}
