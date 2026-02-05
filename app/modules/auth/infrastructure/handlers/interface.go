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

	// HTTP Handlers
	HandleHTTPLogin(w http.ResponseWriter, r *http.Request)
	HandleHTTPTicket(w http.ResponseWriter, r *http.Request)
	HandleHTTPLogout(w http.ResponseWriter, r *http.Request)
}
