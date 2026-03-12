package clubservice

import (
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/google/uuid"
)

// ChallengeScope identifies the club a request targets.
type ChallengeScope struct {
	ClubUUID *uuid.UUID
	GuildID  string
}

// ChallengeActorIdentity identifies an acting user by canonical UUID or club-scoped external ID.
type ChallengeActorIdentity struct {
	UserUUID   *uuid.UUID
	ExternalID string
}

// ChallengeListRequest requests the challenge board for a club.
type ChallengeListRequest struct {
	Scope    ChallengeScope
	Statuses []clubtypes.ChallengeStatus
}

// ChallengeDetailRequest requests one challenge detail record.
type ChallengeDetailRequest struct {
	Scope       ChallengeScope
	ChallengeID uuid.UUID
}

// ChallengeOpenRequest opens a new challenge.
type ChallengeOpenRequest struct {
	Scope  ChallengeScope
	Actor  ChallengeActorIdentity
	Target ChallengeActorIdentity
}

// ChallengeRespondRequest accepts or declines an open challenge.
type ChallengeRespondRequest struct {
	Scope       ChallengeScope
	Actor       ChallengeActorIdentity
	ChallengeID uuid.UUID
	Response    string
}

// ChallengeActionRequest is used by withdraw, hide, and unlink actions.
type ChallengeActionRequest struct {
	Scope       ChallengeScope
	Actor       ChallengeActorIdentity
	ChallengeID uuid.UUID
}

// ChallengeRoundLinkRequest links a round to a challenge.
type ChallengeRoundLinkRequest struct {
	Scope       ChallengeScope
	Actor       ChallengeActorIdentity
	ChallengeID uuid.UUID
	RoundID     uuid.UUID
}

// ChallengeMessageBindingRequest binds a persistent Discord message.
type ChallengeMessageBindingRequest struct {
	ChallengeID uuid.UUID
	GuildID     string
	ChannelID   string
	MessageID   string
}

// ChallengeExpireRequest is emitted by queue workers.
type ChallengeExpireRequest struct {
	ChallengeID uuid.UUID
	Reason      string
}

// ChallengeRoundEventRequest is used for round lifecycle callbacks.
type ChallengeRoundEventRequest struct {
	RoundID uuid.UUID
}

// ChallengeRefreshRequest refreshes active challenge cards for changed members.
type ChallengeRefreshRequest struct {
	ClubUUID    *uuid.UUID
	GuildID     string
	ExternalIDs []string
}

const (
	ChallengeResponseAccept  = "accept"
	ChallengeResponseDecline = "decline"

	challengeOpenTTL   = 48 * time.Hour
	challengeAcceptTTL = 7 * 24 * time.Hour
)
