package authservice

import "errors"

var (
	// ErrInvalidToken is returned when the token is invalid.
	ErrInvalidToken = errors.New("invalid authentication token")

	// ErrExpiredToken is returned when the token has expired.
	ErrExpiredToken = errors.New("authentication token has expired")

	// ErrMissingToken is returned when no token is provided.
	ErrMissingToken = errors.New("missing authentication token")

	// ErrInvalidRole is returned when an invalid role is specified.
	ErrInvalidRole = errors.New("invalid role specified")

	// ErrGenerateToken is returned when token generation fails.
	ErrGenerateToken = errors.New("failed to generate token")

	// ErrGenerateUserJWT is returned when NATS user JWT generation fails.
	ErrGenerateUserJWT = errors.New("failed to generate user credentials")
)
