package authhandlers

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ------------------------
// Fake Service
// ------------------------

type FakeService struct {
	GenerateMagicLinkFunc     func(ctx context.Context, userID, guildID string, role authdomain.Role) (*authservice.MagicLinkResponse, error)
	ValidateTokenFunc         func(ctx context.Context, tokenString string) (*authdomain.Claims, error)
	HandleNATSAuthRequestFunc func(ctx context.Context, req *authservice.NATSAuthRequest) (*authservice.NATSAuthResponse, error)
	LoginUserFunc             func(ctx context.Context, oneTimeToken string) (*authservice.LoginResponse, error)
	GetTicketFunc             func(ctx context.Context, refreshToken string, activeClubUUID string) (*authservice.TicketResponse, error)
	LogoutUserFunc            func(ctx context.Context, refreshToken string) error
	InitiateOAuthLoginFunc    func(ctx context.Context, provider string) (redirectURL, state string, err error)
	HandleOAuthCallbackFunc   func(ctx context.Context, provider, code, state string) (*authservice.LoginResponse, error)
	LinkIdentityToUserFunc    func(ctx context.Context, rawRefreshToken, provider, code, state string) error
	UnlinkProviderFunc        func(ctx context.Context, rawRefreshToken, provider string) error
}

func (f *FakeService) GenerateMagicLink(ctx context.Context, userID, guildID string, role authdomain.Role) (*authservice.MagicLinkResponse, error) {
	if f.GenerateMagicLinkFunc != nil {
		return f.GenerateMagicLinkFunc(ctx, userID, guildID, role)
	}
	return &authservice.MagicLinkResponse{Success: true, URL: "http://test.com"}, nil
}

func (f *FakeService) ValidateToken(ctx context.Context, tokenString string) (*authdomain.Claims, error) {
	if f.ValidateTokenFunc != nil {
		return f.ValidateTokenFunc(ctx, tokenString)
	}
	return &authdomain.Claims{}, nil
}

func (f *FakeService) HandleNATSAuthRequest(ctx context.Context, req *authservice.NATSAuthRequest) (*authservice.NATSAuthResponse, error) {
	if f.HandleNATSAuthRequestFunc != nil {
		return f.HandleNATSAuthRequestFunc(ctx, req)
	}
	return &authservice.NATSAuthResponse{Jwt: "test-jwt"}, nil
}

func (f *FakeService) LoginUser(ctx context.Context, oneTimeToken string) (*authservice.LoginResponse, error) {
	if f.LoginUserFunc != nil {
		return f.LoginUserFunc(ctx, oneTimeToken)
	}
	return &authservice.LoginResponse{RefreshToken: "fake-refresh-token", UserUUID: "test-uuid"}, nil
}

func (f *FakeService) GetTicket(ctx context.Context, refreshToken string, activeClubUUID string) (*authservice.TicketResponse, error) {
	if f.GetTicketFunc != nil {
		return f.GetTicketFunc(ctx, refreshToken, activeClubUUID)
	}
	return &authservice.TicketResponse{NATSToken: "fake-nats-token", RefreshToken: "new-fake-refresh-token"}, nil
}

func (f *FakeService) LogoutUser(ctx context.Context, refreshToken string) error {
	if f.LogoutUserFunc != nil {
		return f.LogoutUserFunc(ctx, refreshToken)
	}
	return nil
}

func (f *FakeService) InitiateOAuthLogin(ctx context.Context, provider string) (string, string, error) {
	if f.InitiateOAuthLoginFunc != nil {
		return f.InitiateOAuthLoginFunc(ctx, provider)
	}
	return "https://discord.com/oauth2/authorize?state=fake", "fake-state", nil
}

func (f *FakeService) HandleOAuthCallback(ctx context.Context, provider, code, state string) (*authservice.LoginResponse, error) {
	if f.HandleOAuthCallbackFunc != nil {
		return f.HandleOAuthCallbackFunc(ctx, provider, code, state)
	}
	return &authservice.LoginResponse{RefreshToken: "fake-refresh-token", UserUUID: "test-uuid"}, nil
}

func (f *FakeService) LinkIdentityToUser(ctx context.Context, rawRefreshToken, provider, code, state string) error {
	if f.LinkIdentityToUserFunc != nil {
		return f.LinkIdentityToUserFunc(ctx, rawRefreshToken, provider, code, state)
	}
	return nil
}

func (f *FakeService) UnlinkProvider(ctx context.Context, rawRefreshToken, provider string) error {
	if f.UnlinkProviderFunc != nil {
		return f.UnlinkProviderFunc(ctx, rawRefreshToken, provider)
	}
	return nil
}

// ------------------------
// Fake EventBus
// ------------------------

type FakeEventBus struct {
	PublishFunc func(topic string, messages ...*message.Message) error
}

func (f *FakeEventBus) Publish(topic string, messages ...*message.Message) error {
	if f.PublishFunc != nil {
		return f.PublishFunc(topic, messages...)
	}
	return nil
}

func (f *FakeEventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return nil, nil
}

func (f *FakeEventBus) Close() error {
	return nil
}

func (f *FakeEventBus) GetNATSConnection() *nats.Conn {
	return nil
}

func (f *FakeEventBus) GetJetStream() jetstream.JetStream {
	return nil
}

func (f *FakeEventBus) GetHealthCheckers() []eventbus.HealthChecker {
	return nil
}

func (f *FakeEventBus) CreateStream(ctx context.Context, streamName string) error {
	return nil
}

func (f *FakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return nil, nil
}

// ------------------------
// Fake Helpers
// ------------------------

type FakeHelpers struct {
	CreateNewMessageFunc func(payload any, topic string) (*message.Message, error)
}

func (f *FakeHelpers) CreateResultMessage(originalMsg *message.Message, payload any, topic string) (*message.Message, error) {
	return message.NewMessage("test-id", nil), nil
}

func (f *FakeHelpers) CreateNewMessage(payload any, topic string) (*message.Message, error) {
	if f.CreateNewMessageFunc != nil {
		return f.CreateNewMessageFunc(payload, topic)
	}
	return message.NewMessage("test-id", nil), nil
}

func (f *FakeHelpers) UnmarshalPayload(msg *message.Message, payload any) error {
	return nil
}
