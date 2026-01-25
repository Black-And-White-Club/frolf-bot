package jwt

import "github.com/golang-jwt/jwt/v5"

type PWAClaims struct {
	jwt.RegisteredClaims
	Guild string `json:"guild"`
	Role  string `json:"role"`
}

type Role string

const (
	RoleViewer Role = "viewer"
	RolePlayer Role = "player"
	RoleEditor Role = "editor"
)
