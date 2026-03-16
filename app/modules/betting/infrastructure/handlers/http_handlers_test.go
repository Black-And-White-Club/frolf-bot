package bettinghandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newHTTPHandlers(svc bettingservice.Service, userRepo userdb.Repository) *HTTPHandlers {
	return NewHTTPHandlers(svc, userRepo, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test"), bettingmetrics.NewNoop())
}

// validSession wires a fake userRepo to return a valid non-revoked token for cookie value "tok".
func validSession(userUUID uuid.UUID) *userdb.FakeRepository {
	repo := &userdb.FakeRepository{}
	repo.GetRefreshTokenFn = func(_ context.Context, _ bun.IDB, _ string) (*userdb.RefreshToken, error) {
		return &userdb.RefreshToken{UserUUID: userUUID, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	return repo
}

func withRefreshCookie(r *http.Request, value string) *http.Request {
	r.AddCookie(&http.Cookie{Name: refreshTokenCookie, Value: value})
	return r
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestHandleGetOverview
// ---------------------------------------------------------------------------

func TestHandleGetOverview(t *testing.T) {
	t.Parallel()
	userUUID := uuid.New()
	clubUUID := uuid.New()

	tests := []struct {
		name   string
		setup  func() (*HTTPHandlers, *http.Request)
		verify func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing cookie → 401",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, &userdb.FakeRepository{})
				r := httptest.NewRequest(http.MethodGet, "/betting/overview?club_uuid="+clubUUID.String(), nil)
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("want 401, got %d", rr.Code)
				}
			},
		},
		{
			name: "missing club_uuid → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodGet, "/betting/overview", nil), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "service error ErrFeatureDisabled → 403",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.GetOverviewFunc = func(_ context.Context, _, _ uuid.UUID) (*bettingservice.Overview, error) {
					return nil, bettingservice.ErrFeatureDisabled
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodGet, "/betting/overview?club_uuid="+clubUUID.String(), nil), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusForbidden {
					t.Errorf("want 403, got %d", rr.Code)
				}
			},
		},
		{
			name: "success → 200 with JSON overview",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.GetOverviewFunc = func(_ context.Context, _, _ uuid.UUID) (*bettingservice.Overview, error) {
					return &bettingservice.Overview{ClubUUID: clubUUID.String(), GuildID: "guild-1"}, nil
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodGet, "/betting/overview?club_uuid="+clubUUID.String(), nil), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusOK {
					t.Errorf("want 200, got %d", rr.Code)
				}
				var body bettingservice.Overview
				decodeJSON(t, rr.Body, &body)
				if body.ClubUUID != clubUUID.String() {
					t.Errorf("ClubUUID: want %s, got %s", clubUUID, body.ClubUUID)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, r := tt.setup()
			rr := httptest.NewRecorder()
			h.HandleGetOverview(rr, r)
			tt.verify(t, rr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandlePlaceBet
// ---------------------------------------------------------------------------

func TestHandlePlaceBet(t *testing.T) {
	t.Parallel()
	userUUID := uuid.New()

	validBody := func(req bettingservice.PlaceBetRequest) *bytes.Buffer {
		b, _ := json.Marshal(req)
		return bytes.NewBuffer(b)
	}

	tests := []struct {
		name   string
		setup  func() (*HTTPHandlers, *http.Request)
		verify func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing cookie → 401",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, &userdb.FakeRepository{})
				r := httptest.NewRequest(http.MethodPost, "/betting/bet", validBody(bettingservice.PlaceBetRequest{}))
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("want 401, got %d", rr.Code)
				}
			},
		},
		{
			name: "malformed JSON body → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/bet", bytes.NewBufferString("not-json")), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrMarketLocked → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.PlaceBetFunc = func(_ context.Context, _ bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
					return nil, bettingservice.ErrMarketLocked
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/bet", validBody(bettingservice.PlaceBetRequest{Stake: 100})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrInsufficientBalance → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.PlaceBetFunc = func(_ context.Context, _ bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
					return nil, bettingservice.ErrInsufficientBalance
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/bet", validBody(bettingservice.PlaceBetRequest{Stake: 100})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "success → 201 with ticket",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.PlaceBetFunc = func(_ context.Context, req bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
					return &bettingservice.BetTicket{SelectionKey: "player-a", Stake: req.Stake, Status: "accepted"}, nil
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/bet", validBody(bettingservice.PlaceBetRequest{Stake: 50})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusCreated {
					t.Errorf("want 201, got %d", rr.Code)
				}
				var ticket bettingservice.BetTicket
				decodeJSON(t, rr.Body, &ticket)
				if ticket.Status != "accepted" {
					t.Errorf("Status: want accepted, got %s", ticket.Status)
				}
			},
		},
		{
			name: "user UUID is injected from session (not request body)",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.PlaceBetFunc = func(_ context.Context, req bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
					if req.UserUUID != userUUID {
						return nil, bettingservice.ErrSelectionInvalid // sentinel to signal wrong UUID
					}
					return &bettingservice.BetTicket{Status: "accepted"}, nil
				}
				h := newHTTPHandlers(svc, validSession(userUUID))
				// Body has a different UserUUID — handler must overwrite it
				body := bettingservice.PlaceBetRequest{UserUUID: uuid.New(), Stake: 50}
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/bet", func() *bytes.Buffer {
					b, _ := json.Marshal(body)
					return bytes.NewBuffer(b)
				}()), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusCreated {
					t.Errorf("want 201 (session UUID used), got %d", rr.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, r := tt.setup()
			rr := httptest.NewRecorder()
			h.HandlePlaceBet(rr, r)
			tt.verify(t, rr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandleAdjustWallet
// ---------------------------------------------------------------------------

func TestHandleAdjustWallet(t *testing.T) {
	t.Parallel()
	adminUUID := uuid.New()

	jsonBody := func(v any) *bytes.Buffer {
		b, _ := json.Marshal(v)
		return bytes.NewBuffer(b)
	}

	tests := []struct {
		name   string
		setup  func() (*HTTPHandlers, *http.Request)
		verify func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing cookie → 401",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, &userdb.FakeRepository{})
				r := httptest.NewRequest(http.MethodPost, "/betting/admin/wallet", jsonBody(bettingservice.AdjustWalletRequest{}))
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("want 401, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrAdminRequired → 403",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdjustWalletFunc = func(_ context.Context, _ bettingservice.AdjustWalletRequest) (*bettingservice.WalletJournal, error) {
					return nil, bettingservice.ErrAdminRequired
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/wallet", jsonBody(bettingservice.AdjustWalletRequest{Amount: 100, Reason: "test"})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusForbidden {
					t.Errorf("want 403, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrAdjustmentAmountInvalid → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdjustWalletFunc = func(_ context.Context, _ bettingservice.AdjustWalletRequest) (*bettingservice.WalletJournal, error) {
					return nil, bettingservice.ErrAdjustmentAmountInvalid
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/wallet", jsonBody(bettingservice.AdjustWalletRequest{Amount: 0})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "success → 201 with entry",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdjustWalletFunc = func(_ context.Context, _ bettingservice.AdjustWalletRequest) (*bettingservice.WalletJournal, error) {
					return &bettingservice.WalletJournal{EntryType: "manual_adjustment", Amount: 50}, nil
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/wallet", jsonBody(bettingservice.AdjustWalletRequest{Amount: 50, Reason: "bonus"})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusCreated {
					t.Errorf("want 201, got %d", rr.Code)
				}
				var entry bettingservice.WalletJournal
				decodeJSON(t, rr.Body, &entry)
				if entry.Amount != 50 {
					t.Errorf("Amount: want 50, got %d", entry.Amount)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, r := tt.setup()
			rr := httptest.NewRecorder()
			h.HandleAdjustWallet(rr, r)
			tt.verify(t, rr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandleAdminMarketAction
// ---------------------------------------------------------------------------

func TestHandleAdminMarketAction(t *testing.T) {
	t.Parallel()
	adminUUID := uuid.New()

	jsonBody := func(v any) *bytes.Buffer {
		b, _ := json.Marshal(v)
		return bytes.NewBuffer(b)
	}

	tests := []struct {
		name   string
		setup  func() (*HTTPHandlers, *http.Request)
		verify func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing cookie → 401",
			setup: func() (*HTTPHandlers, *http.Request) {
				h := newHTTPHandlers(&FakeBettingService{}, &userdb.FakeRepository{})
				r := httptest.NewRequest(http.MethodPost, "/betting/admin/market", jsonBody(bettingservice.AdminMarketActionRequest{}))
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("want 401, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrInvalidMarketAction → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdminMarketActionFunc = func(_ context.Context, _ bettingservice.AdminMarketActionRequest) (*bettingservice.AdminMarketActionResult, error) {
					return nil, bettingservice.ErrInvalidMarketAction
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/market", jsonBody(bettingservice.AdminMarketActionRequest{Action: "bad"})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "service ErrRoundNotFinalized → 400",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdminMarketActionFunc = func(_ context.Context, _ bettingservice.AdminMarketActionRequest) (*bettingservice.AdminMarketActionResult, error) {
					return nil, bettingservice.ErrRoundNotFinalized
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/market", jsonBody(bettingservice.AdminMarketActionRequest{Action: "resettle"})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("want 400, got %d", rr.Code)
				}
			},
		},
		{
			name: "success → 200 with result",
			setup: func() (*HTTPHandlers, *http.Request) {
				svc := &FakeBettingService{}
				svc.AdminMarketActionFunc = func(_ context.Context, req bettingservice.AdminMarketActionRequest) (*bettingservice.AdminMarketActionResult, error) {
					return &bettingservice.AdminMarketActionResult{Action: req.Action, MarketID: 42}, nil
				}
				h := newHTTPHandlers(svc, validSession(adminUUID))
				r := withRefreshCookie(httptest.NewRequest(http.MethodPost, "/betting/admin/market", jsonBody(bettingservice.AdminMarketActionRequest{Action: "void", MarketID: 42})), "tok")
				return h, r
			},
			verify: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Code != http.StatusOK {
					t.Errorf("want 200, got %d", rr.Code)
				}
				var result bettingservice.AdminMarketActionResult
				decodeJSON(t, rr.Body, &result)
				if result.MarketID != 42 {
					t.Errorf("MarketID: want 42, got %d", result.MarketID)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, r := tt.setup()
			rr := httptest.NewRecorder()
			h.HandleAdminMarketAction(rr, r)
			tt.verify(t, rr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestWriteServiceError — full sentinel → HTTP status mapping
// ---------------------------------------------------------------------------

func TestWriteServiceError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{bettingservice.ErrMembershipRequired, http.StatusForbidden, "membership_required"},
		{bettingservice.ErrFeatureDisabled, http.StatusForbidden, "feature_disabled"},
		{bettingservice.ErrFeatureFrozen, http.StatusForbidden, "feature_frozen"},
		{bettingservice.ErrAdminRequired, http.StatusForbidden, "admin_required"},
		{bettingservice.ErrTargetMemberNotFound, http.StatusNotFound, "target_member_not_found"},
		{bettingservice.ErrAdjustmentAmountInvalid, http.StatusBadRequest, "invalid_adjustment_amount"},
		{bettingservice.ErrAdjustmentReasonRequired, http.StatusBadRequest, "reason_required"},
		{bettingservice.ErrNoEligibleRound, http.StatusNotFound, "no_eligible_round"},
		{bettingservice.ErrBetStakeInvalid, http.StatusBadRequest, "invalid_stake"},
		{bettingservice.ErrSelectionInvalid, http.StatusBadRequest, "invalid_selection"},
		{bettingservice.ErrInsufficientBalance, http.StatusBadRequest, "insufficient_balance"},
		{bettingservice.ErrMarketLocked, http.StatusBadRequest, "market_locked"},
		{bettingservice.ErrMarketNotFound, http.StatusNotFound, "market_not_found"},
		{bettingservice.ErrInvalidMarketAction, http.StatusBadRequest, "invalid_market_action"},
		{bettingservice.ErrRoundNotFinalized, http.StatusBadRequest, "round_not_finalized"},
		{bettingservice.ErrSelfBetProhibited, http.StatusUnprocessableEntity, "self_bet_prohibited"},
		{bettingservice.ErrInvalidMarketType, http.StatusBadRequest, "invalid_market_type"},
	}

	h := newHTTPHandlers(&FakeBettingService{}, &userdb.FakeRepository{})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.wantCode, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			h.writeServiceError(rr, r, tc.err)

			if rr.Code != tc.wantStatus {
				t.Errorf("status: want %d, got %d", tc.wantStatus, rr.Code)
			}
			var body map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body["code"] != tc.wantCode {
				t.Errorf("code: want %q, got %q", tc.wantCode, body["code"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandlePlaceBet_RateLimitExceeded (F7)
// ---------------------------------------------------------------------------

func TestHandlePlaceBet_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	userUUID := uuid.New()
	clubUUID := uuid.New()

	svc := &FakeBettingService{}
	svc.PlaceBetFunc = func(_ context.Context, req bettingservice.PlaceBetRequest) (*bettingservice.BetTicket, error) {
		return &bettingservice.BetTicket{}, nil
	}

	h := newHTTPHandlers(svc, validSession(userUUID))
	// Pre-drain the burst so the very next request is rate-limited.
	for range betRateLimitBurst {
		h.rateLimiter.allow(clubUUID, userUUID)
	}

	reqBody, _ := json.Marshal(bettingservice.PlaceBetRequest{ClubUUID: clubUUID, Stake: 10})

	// The burst is exhausted; the next request should be rate-limited.
	rr := httptest.NewRecorder()
	r := withRefreshCookie(
		httptest.NewRequest(http.MethodPost, "/betting/bet", bytes.NewBuffer(reqBody)),
		"tok",
	)
	h.HandlePlaceBet(rr, r)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("want 429 TooManyRequests, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "rate_limit_exceeded" {
		t.Errorf("want code=rate_limit_exceeded, got %q", body["code"])
	}
}
