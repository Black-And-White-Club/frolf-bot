package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service interface {
	GenerateToken(userID, guildID string, role Role, ttl time.Duration) (string, error)
	ValidateToken(tokenString string) (*PWAClaims, error)
	GenerateMagicLink(userID, guildID string, role Role) (string, error)
}

type service struct {
	secret     []byte
	defaultTTL time.Duration
	pwaBaseURL string
}

func NewService(secret string, defaultTTL time.Duration, pwaBaseURL string) Service {
	return &service{
		secret:     []byte(secret),
		defaultTTL: defaultTTL,
		pwaBaseURL: pwaBaseURL,
	}
}

func (s *service) GenerateToken(userID, guildID string, role Role, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &PWAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Guild: guildID,
		Role:  string(role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (s *service) ValidateToken(tokenString string) (*PWAClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PWAClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*PWAClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *service) GenerateMagicLink(userID, guildID string, role Role) (string, error) {
	token, err := s.GenerateToken(userID, guildID, role, s.defaultTTL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s?t=%s", s.pwaBaseURL, token), nil
}
