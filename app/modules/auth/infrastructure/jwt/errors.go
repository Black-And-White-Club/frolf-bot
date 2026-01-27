package authjwt

import "errors"

var (
	// ErrInvalidToken is returned when the token is malformed or invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when the token has expired.
	ErrExpiredToken = errors.New("token has expired")

	// ErrInvalidSignature is returned when the token signature is invalid.
	ErrInvalidSignature = errors.New("invalid token signature")
)
