package authservice

import (
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	"github.com/google/uuid"
)

// ------------------------
// Fake JWT Provider
// ------------------------

type FakeJWTProvider struct {
	trace []string

	GenerateTokenFunc func(claims *authdomain.Claims, ttl time.Duration) (string, error)
	ValidateTokenFunc func(tokenString string) (*authdomain.Claims, error)
}

func (f *FakeJWTProvider) Trace() []string {
	return f.trace
}

func (f *FakeJWTProvider) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeJWTProvider) GenerateToken(claims *authdomain.Claims, ttl time.Duration) (string, error) {
	f.record("GenerateToken")
	if f.GenerateTokenFunc != nil {
		return f.GenerateTokenFunc(claims, ttl)
	}
	return "fake-token", nil
}

func (f *FakeJWTProvider) ValidateToken(tokenString string) (*authdomain.Claims, error) {
	f.record("ValidateToken")
	if f.ValidateTokenFunc != nil {
		return f.ValidateTokenFunc(tokenString)
	}
	return &authdomain.Claims{
		UserID:   "test-user",
		UserUUID: uuid.New(),
		GuildID:  "test-guild",
		Role:     authdomain.RolePlayer,
	}, nil
}

// ------------------------
// Fake User JWT Builder
// ------------------------

type FakeUserJWTBuilder struct {
	trace []string

	BuildUserJWTFunc func(claims *authdomain.Claims, perms *permissions.Permissions) (string, error)
}

func (f *FakeUserJWTBuilder) Trace() []string {
	return f.trace
}

func (f *FakeUserJWTBuilder) record(step string) {
	f.trace = append(f.trace, step)
}

func (f *FakeUserJWTBuilder) BuildUserJWT(claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
	f.record("BuildUserJWT")
	if f.BuildUserJWTFunc != nil {
		return f.BuildUserJWTFunc(claims, perms)
	}
	return "fake-nats-jwt", nil
}
