package authhandlers

import (
	"encoding/json"
	"net/http"
)

type activityTokenExchangeRequest struct {
	Code string `json:"code"`
}

type activityTokenExchangeResponse struct {
	RefreshToken string `json:"refresh_token"`
	Ticket       string `json:"ticket"`
	UserUUID     string `json:"user_uuid"`
}

// HandleActivityTokenExchange exchanges a Discord Activity OAuth code for a
// refresh token, NATS ticket, and user UUID.
//
// This is a public endpoint — no cookie or prior session required.
// The Discord Embedded App SDK returns a short-lived code via JavaScript
// (not a browser redirect), which is sent here for server-side exchange.
// chi's r.Post() already restricts this to POST; the body size is capped by
// MaxBytesReader before decoding.
func (h *AuthHandlers) HandleActivityTokenExchange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var req activityTokenExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		h.httpError(w, r, "invalid request", http.StatusBadRequest, err)
		return
	}

	// Exchange the Discord Activity code for a frolf-bot session.
	// Uses the "discord-activity" OAuth provider (registered with the Activity
	// redirect URL), which exchanges the code with Discord then resolves or
	// creates the canonical frolf-bot user via linked_identities.
	loginResp, err := h.service.HandleOAuthCallback(ctx, "discord-activity", req.Code, "")
	if err != nil {
		h.httpError(w, r, "authentication failed", http.StatusUnauthorized, err)
		return
	}

	// Mint a NATS ticket for the newly-issued refresh token.
	ticketResp, err := h.service.GetTicket(ctx, loginResp.RefreshToken, "")
	if err != nil {
		h.httpError(w, r, "authentication failed", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(activityTokenExchangeResponse{
		RefreshToken: ticketResp.RefreshToken,
		Ticket:       ticketResp.NATSToken,
		UserUUID:     loginResp.UserUUID,
	})
}
