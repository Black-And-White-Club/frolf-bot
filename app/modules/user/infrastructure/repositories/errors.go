package userdb

import "errors"

// Sentinel errors for the user repository layer.
// These indicate infrastructure-level outcomes (presence/absence of rows), not
// domain validation failures. Service/business layers decide how to map these
// into domain errors or user-visible messages.
var (
    // ErrNotFound indicates the requested user/row does not exist.
    ErrNotFound = errors.New("user record not found")

    // ErrNoRowsAffected indicates an UPDATE/DELETE affected zero rows.
    ErrNoRowsAffected = errors.New("no rows affected")
)
