package authhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAuthHandlers_HandleMagicLinkRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name         string
		reqPayload   MagicLinkRequest
		setupService func(*FakeService)
		verify       func(t *testing.T, called bool)
	}{
		{
			name: "success",
			reqPayload: MagicLinkRequest{
				UserID:        "u1",
				GuildID:       "g1",
				Role:          "player",
				CorrelationID: "corr1",
			},
			setupService: func(s *FakeService) {
				s.GenerateMagicLinkFunc = func(ctx context.Context, userID, guildID string, role authdomain.Role) (*authservice.MagicLinkResponse, error) {
					return &authservice.MagicLinkResponse{Success: true, URL: "http://magic.link"}, nil
				}
			},
			verify: func(t *testing.T, called bool) {
				if !called {
					t.Error("expected EventBus.Publish to be called")
				}
			},
		},
		{
			name: "service error",
			reqPayload: MagicLinkRequest{
				UserID:  "u1",
				GuildID: "g1",
			},
			setupService: func(s *FakeService) {
				s.GenerateMagicLinkFunc = func(ctx context.Context, userID, guildID string, role authdomain.Role) (*authservice.MagicLinkResponse, error) {
					return nil, errors.New("service error")
				}
			},
			verify: func(t *testing.T, called bool) {
				// Even on error, it might publish a failure (depending on implementation)
				// Currently, the fake has called = true if Publish is called.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := &FakeService{}
			fakeEventBus := &FakeEventBus{}
			fakeHelpers := &FakeHelpers{}

			if tt.setupService != nil {
				tt.setupService(fakeService)
			}

			called := false
			fakeEventBus.PublishFunc = func(topic string, messages ...*message.Message) error {
				called = true
				return nil
			}

			h := NewAuthHandlers(fakeService, fakeEventBus, fakeHelpers, logger, tracer)

			data, _ := json.Marshal(tt.reqPayload)
			msg := &nats.Msg{Data: data}

			h.HandleMagicLinkRequest(msg)

			if tt.verify != nil {
				tt.verify(t, called)
			}
		})
	}
}

func TestAuthHandlers_HandleNATSAuthCallout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name         string
		reqData      []byte
		setupService func(*FakeService)
		verify       func(t *testing.T)
	}{
		{
			name:    "success",
			reqData: []byte(`{"connect_opts":{"pass":"token"}}`),
			setupService: func(s *FakeService) {
				s.HandleNATSAuthRequestFunc = func(ctx context.Context, req *authservice.NATSAuthRequest) (*authservice.NATSAuthResponse, error) {
					return &authservice.NATSAuthResponse{Jwt: "jwt"}, nil
				}
			},
		},
		{
			name:    "invalid json",
			reqData: []byte(`invalid`),
			setupService: func(s *FakeService) {
				s.HandleNATSAuthRequestFunc = func(ctx context.Context, req *authservice.NATSAuthRequest) (*authservice.NATSAuthResponse, error) {
					t.Error("service should not be called on invalid json")
					return nil, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := &FakeService{}
			if tt.setupService != nil {
				tt.setupService(fakeService)
			}

			h := NewAuthHandlers(fakeService, &FakeEventBus{}, &FakeHelpers{}, logger, tracer)
			msg := &nats.Msg{Data: tt.reqData}
			h.HandleNATSAuthCallout(msg)

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}
