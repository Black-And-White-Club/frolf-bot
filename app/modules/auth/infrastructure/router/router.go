package authrouter

import (
	authhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/handlers"
	"github.com/nats-io/nats.go"
)

const (
	// MagicLinkRequestSubject is the NATS subject for magic link requests.
	MagicLinkRequestSubject = "auth.magic-link.requested.v1"

	// AuthCalloutSubject is the default NATS subject for auth callout requests.
	AuthCalloutSubject = "$SYS.REQ.USER.AUTH"

	// QueueGroup is the queue group name for load balancing.
	QueueGroup = "backend"
)

// Router manages NATS subscriptions for the auth module.
type Router struct {
	handlers              authhandlers.Handlers
	nc                    *nats.Conn
	magicLinkSubscription *nats.Subscription
	authCalloutSub        *nats.Subscription
}

// NewRouter creates a new auth router.
func NewRouter(handlers authhandlers.Handlers, nc *nats.Conn) *Router {
	return &Router{
		handlers: handlers,
		nc:       nc,
	}
}

// Start subscribes to all auth-related NATS subjects.
func (r *Router) Start(authCalloutSubject string) error {
	var err error

	// Subscribe to magic link requests with queue group for load balancing
	r.magicLinkSubscription, err = r.nc.QueueSubscribe(
		MagicLinkRequestSubject,
		QueueGroup,
		r.handlers.HandleMagicLinkRequest,
	)
	if err != nil {
		return err
	}

	// Subscribe to auth callout subject (no queue group - each instance handles all)
	subject := authCalloutSubject
	if subject == "" {
		subject = AuthCalloutSubject
	}

	r.authCalloutSub, err = r.nc.Subscribe(subject, r.handlers.HandleNATSAuthCallout)
	if err != nil {
		// Clean up the magic link subscription if auth callout fails
		if r.magicLinkSubscription != nil {
			r.magicLinkSubscription.Unsubscribe()
		}
		return err
	}

	return nil
}

// Stop unsubscribes from all NATS subjects.
func (r *Router) Stop() error {
	var firstErr error

	if r.magicLinkSubscription != nil {
		if err := r.magicLinkSubscription.Unsubscribe(); err != nil {
			firstErr = err
		}
	}

	if r.authCalloutSub != nil {
		if err := r.authCalloutSub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
