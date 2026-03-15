package bettinghandlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

const refreshTokenCookie = "refresh_token"

// betRateLimiter is a per-user token-bucket rate limiter for bet placement.
// It limits each (clubUUID, userUUID) pair to betRateLimitPerMin placements
// per minute, providing a burst of up to betRateLimitBurst.
//
// The limiter map grows with unique users; for a typical Discord guild this is
// bounded and acceptable in v1. Production deployments should add periodic
// cleanup of stale keys.
const (
	betRateLimitPerMin = 10
	betRateLimitBurst  = 5
)

type betRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newBetRateLimiter() *betRateLimiter {
	return &betRateLimiter{limiters: make(map[string]*rate.Limiter)}
}

func (rl *betRateLimiter) allow(clubUUID, userUUID uuid.UUID) bool {
	key := clubUUID.String() + ":" + userUUID.String()
	rl.mu.Lock()
	l, ok := rl.limiters[key]
	if !ok {
		l = rate.NewLimiter(rate.Every(time.Minute/betRateLimitPerMin), betRateLimitBurst)
		rl.limiters[key] = l
	}
	rl.mu.Unlock()
	return l.Allow()
}

type HTTPHandlers struct {
	service     bettingservice.Service
	userRepo    userdb.Repository
	logger      *slog.Logger
	tracer      trace.Tracer
	metrics     bettingmetrics.BettingMetrics
	rateLimiter *betRateLimiter
}

func NewHTTPHandlers(
	service bettingservice.Service,
	userRepo userdb.Repository,
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics bettingmetrics.BettingMetrics,
) *HTTPHandlers {
	return &HTTPHandlers{
		service:     service,
		userRepo:    userRepo,
		logger:      logger,
		tracer:      tracer,
		metrics:     metrics,
		rateLimiter: newBetRateLimiter(),
	}
}

func (h *HTTPHandlers) resolveUserUUID(r *http.Request) (uuid.UUID, error) {
	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || cookie.Value == "" {
		return uuid.Nil, fmt.Errorf("missing session")
	}
	hash := sha256hex(cookie.Value)
	token, err := h.userRepo.GetRefreshToken(r.Context(), nil, hash)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid session")
	}
	if token.Revoked {
		return uuid.Nil, fmt.Errorf("session revoked")
	}
	return token.UserUUID, nil
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{
		"code":  code,
		"error": msg,
	})
}

func (h *HTTPHandlers) HandleGetOverview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleGetOverview")

	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetOverview")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(r.URL.Query().Get("club_uuid"))
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetOverview")
		httpError(w, http.StatusBadRequest, "invalid_club_uuid", "invalid club_uuid")
		return
	}

	overview, err := h.service.GetOverview(r.Context(), clubUUID, userUUID)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetOverview")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleGetOverview")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleGetOverview", time.Since(start))
	writeJSON(w, http.StatusOK, overview)
}

func (h *HTTPHandlers) HandleGetNextRoundMarket(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleGetNextRoundMarket")

	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetNextRoundMarket")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(r.URL.Query().Get("club_uuid"))
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetNextRoundMarket")
		httpError(w, http.StatusBadRequest, "invalid_club_uuid", "invalid club_uuid")
		return
	}

	market, err := h.service.GetNextRoundMarket(r.Context(), clubUUID, userUUID)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetNextRoundMarket")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleGetNextRoundMarket")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleGetNextRoundMarket", time.Since(start))
	writeJSON(w, http.StatusOK, market)
}

func (h *HTTPHandlers) HandleGetAdminMarkets(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleGetAdminMarkets")

	adminUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetAdminMarkets")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	clubUUID, err := uuid.Parse(r.URL.Query().Get("club_uuid"))
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetAdminMarkets")
		httpError(w, http.StatusBadRequest, "invalid_club_uuid", "invalid club_uuid")
		return
	}

	board, err := h.service.GetAdminMarkets(r.Context(), clubUUID, adminUUID)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleGetAdminMarkets")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleGetAdminMarkets")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleGetAdminMarkets", time.Since(start))
	writeJSON(w, http.StatusOK, board)
}

func (h *HTTPHandlers) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleUpdateSettings")

	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleUpdateSettings")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req bettingservice.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleUpdateSettings")
		httpError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	req.UserUUID = userUUID

	settings, err := h.service.UpdateSettings(r.Context(), req)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleUpdateSettings")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleUpdateSettings")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleUpdateSettings", time.Since(start))
	writeJSON(w, http.StatusOK, settings)
}

func (h *HTTPHandlers) HandleAdjustWallet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleAdjustWallet")

	adminUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdjustWallet")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req bettingservice.AdjustWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdjustWallet")
		httpError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	req.AdminUUID = adminUUID

	entry, err := h.service.AdjustWallet(r.Context(), req)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdjustWallet")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleAdjustWallet")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleAdjustWallet", time.Since(start))
	writeJSON(w, http.StatusCreated, entry)
}

func (h *HTTPHandlers) HandlePlaceBet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandlePlaceBet")

	userUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandlePlaceBet")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req bettingservice.PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandlePlaceBet")
		httpError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	req.UserUUID = userUUID

	// Rate limit per (clubUUID, userUUID) to prevent wallet-lock spam.
	if !h.rateLimiter.allow(req.ClubUUID, userUUID) {
		h.metrics.RecordHandlerFailure(r.Context(), "HandlePlaceBet")
		httpError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many bet placements — please wait before trying again")
		return
	}

	ticket, err := h.service.PlaceBet(r.Context(), req)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandlePlaceBet")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandlePlaceBet")
	h.metrics.RecordHandlerDuration(r.Context(), "HandlePlaceBet", time.Since(start))
	writeJSON(w, http.StatusCreated, ticket)
}

func (h *HTTPHandlers) HandleAdminMarketAction(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(r.Context(), "HandleAdminMarketAction")

	adminUUID, err := h.resolveUserUUID(r)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdminMarketAction")
		httpError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req bettingservice.AdminMarketActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdminMarketAction")
		httpError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	req.AdminUUID = adminUUID

	result, err := h.service.AdminMarketAction(r.Context(), req)
	if err != nil {
		h.metrics.RecordHandlerFailure(r.Context(), "HandleAdminMarketAction")
		h.writeServiceError(w, r, err)
		return
	}

	h.metrics.RecordHandlerSuccess(r.Context(), "HandleAdminMarketAction")
	h.metrics.RecordHandlerDuration(r.Context(), "HandleAdminMarketAction", time.Since(start))
	writeJSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, bettingservice.ErrMembershipRequired):
		httpError(w, http.StatusForbidden, "membership_required", "club membership required")
	case errors.Is(err, bettingservice.ErrFeatureDisabled):
		httpError(w, http.StatusForbidden, "feature_disabled", "betting is not enabled for this club")
	case errors.Is(err, bettingservice.ErrFeatureFrozen):
		httpError(w, http.StatusForbidden, "feature_frozen", "betting is currently in read-only freeze")
	case errors.Is(err, bettingservice.ErrAdminRequired):
		httpError(w, http.StatusForbidden, "admin_required", "admin role required")
	case errors.Is(err, bettingservice.ErrTargetMemberNotFound):
		httpError(w, http.StatusNotFound, "target_member_not_found", "target member not found")
	case errors.Is(err, bettingservice.ErrAdjustmentAmountInvalid):
		httpError(w, http.StatusBadRequest, "invalid_adjustment_amount", "adjustment amount must be non-zero")
	case errors.Is(err, bettingservice.ErrAdjustmentReasonRequired):
		httpError(w, http.StatusBadRequest, "reason_required", "reason is required")
	case errors.Is(err, bettingservice.ErrNoEligibleRound):
		httpError(w, http.StatusNotFound, "no_eligible_round", "no eligible betting round found")
	case errors.Is(err, bettingservice.ErrBetStakeInvalid):
		httpError(w, http.StatusBadRequest, "invalid_stake", "stake must be greater than zero")
	case errors.Is(err, bettingservice.ErrSelectionInvalid):
		httpError(w, http.StatusBadRequest, "invalid_selection", "invalid betting selection")
	case errors.Is(err, bettingservice.ErrInsufficientBalance):
		httpError(w, http.StatusBadRequest, "insufficient_balance", "insufficient betting balance")
	case errors.Is(err, bettingservice.ErrMarketLocked):
		httpError(w, http.StatusBadRequest, "market_locked", "market is locked")
	case errors.Is(err, bettingservice.ErrMarketNotFound):
		httpError(w, http.StatusNotFound, "market_not_found", "market not found")
	case errors.Is(err, bettingservice.ErrInvalidMarketAction):
		httpError(w, http.StatusBadRequest, "invalid_market_action", "invalid market action")
	case errors.Is(err, bettingservice.ErrRoundNotFinalized):
		httpError(w, http.StatusBadRequest, "round_not_finalized", "round is not finalized")
	case errors.Is(err, bettingservice.ErrSelfBetProhibited):
		httpError(w, http.StatusUnprocessableEntity, "self_bet_prohibited", "you cannot bet on yourself in this market")
	case errors.Is(err, bettingservice.ErrInvalidMarketType):
		httpError(w, http.StatusBadRequest, "invalid_market_type", "invalid market type")
	default:
		h.logger.ErrorContext(r.Context(), "betting handler failed", slog.String("error", err.Error()))
		httpError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
