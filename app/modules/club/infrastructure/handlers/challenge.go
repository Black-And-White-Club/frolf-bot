package clubhandlers

import (
	"context"
	"fmt"
	"strings"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	"github.com/google/uuid"
)

func (h *ClubHandlers) HandleChallengeListRequest(ctx context.Context, payload *clubevents.ChallengeListRequestPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeListRequest(payload)
	if err != nil {
		return nil, err
	}
	challenges, err := h.service.ListChallenges(ctx, req)
	if err != nil {
		return nil, err
	}
	return []handlerwrapper.Result{{
		Topic:   replyTopicFromContext(ctx, clubevents.ChallengeListResponseV1),
		Payload: &clubevents.ChallengeListResponsePayloadV1{Challenges: challenges},
	}}, nil
}

func (h *ClubHandlers) HandleChallengeDetailRequest(ctx context.Context, payload *clubevents.ChallengeDetailRequestPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeDetailRequest(payload)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.GetChallengeDetail(ctx, req)
	if err != nil {
		return nil, err
	}
	return []handlerwrapper.Result{{
		Topic:   replyTopicFromContext(ctx, clubevents.ChallengeDetailResponseV1),
		Payload: &clubevents.ChallengeDetailResponsePayloadV1{Challenge: challenge},
	}}, nil
}

func (h *ClubHandlers) HandleChallengeOpenRequested(ctx context.Context, payload *clubevents.ChallengeOpenRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeOpenRequest(payload)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.OpenChallenge(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeOpenedV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeRespondRequested(ctx context.Context, payload *clubevents.ChallengeRespondRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeRespondRequest(payload)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.RespondToChallenge(ctx, req)
	if err != nil {
		return nil, err
	}
	topic := clubevents.ChallengeAcceptedV1
	if strings.EqualFold(req.Response, clubservice.ChallengeResponseDecline) {
		topic = clubevents.ChallengeDeclinedV1
	}
	return h.challengeResults(topic, challenge), nil
}

func (h *ClubHandlers) HandleChallengeWithdrawRequested(ctx context.Context, payload *clubevents.ChallengeWithdrawRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeActionRequest(payload.ClubUUID, payload.GuildID, payload.ActorUserUUID, payload.ActorExternalID, payload.ChallengeID)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.WithdrawChallenge(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeWithdrawnV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeHideRequested(ctx context.Context, payload *clubevents.ChallengeHideRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeActionRequest(payload.ClubUUID, payload.GuildID, payload.ActorUserUUID, payload.ActorExternalID, payload.ChallengeID)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.HideChallenge(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeHiddenV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeRoundLinkRequested(ctx context.Context, payload *clubevents.ChallengeRoundLinkRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeRoundLinkRequest(payload)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.LinkChallengeRound(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeRoundLinkedV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeRoundUnlinkRequested(ctx context.Context, payload *clubevents.ChallengeRoundUnlinkRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	req, err := parseChallengeActionRequest(payload.ClubUUID, payload.GuildID, payload.ActorUserUUID, payload.ActorExternalID, payload.ChallengeID)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.UnlinkChallengeRound(ctx, req)
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeRoundUnlinkedV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeMessageBindRequested(ctx context.Context, payload *clubevents.ChallengeMessageBindRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	challengeID, err := uuid.Parse(payload.ChallengeID)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.BindChallengeMessage(ctx, clubservice.ChallengeMessageBindingRequest{
		ChallengeID: challengeID,
		GuildID:     payload.GuildID,
		ChannelID:   payload.ChannelID,
		MessageID:   payload.MessageID,
	})
	if err != nil {
		return nil, err
	}
	return h.challengeResults(clubevents.ChallengeRefreshedV1, challenge), nil
}

func (h *ClubHandlers) HandleChallengeExpireRequested(ctx context.Context, payload *clubevents.ChallengeExpireRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	challengeID, err := uuid.Parse(payload.ChallengeID)
	if err != nil {
		return nil, err
	}
	challenge, err := h.service.ExpireChallenge(ctx, clubservice.ChallengeExpireRequest{
		ChallengeID: challengeID,
		Reason:      payload.Reason,
	})
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, nil
	}
	return h.challengeResults(clubevents.ChallengeExpiredV1, challenge), nil
}

func (h *ClubHandlers) HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedPayloadV1) ([]handlerwrapper.Result, error) {
	challenge, err := h.service.CompleteChallengeForRound(ctx, clubservice.ChallengeRoundEventRequest{RoundID: payload.RoundID.UUID()})
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, nil
	}
	return h.challengeResults(clubevents.ChallengeCompletedV1, challenge), nil
}

func (h *ClubHandlers) HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error) {
	challenge, err := h.service.ResetChallengeForRound(ctx, clubservice.ChallengeRoundEventRequest{RoundID: payload.RoundID.UUID()})
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, nil
	}
	return h.challengeResults(clubevents.ChallengeRoundUnlinkedV1, challenge), nil
}

func (h *ClubHandlers) HandleLeaderboardTagUpdated(ctx context.Context, payload *leaderboardevents.LeaderboardTagUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	var clubUUID *uuid.UUID
	if payload.ClubUUID != nil && *payload.ClubUUID != "" {
		parsed, err := uuid.Parse(*payload.ClubUUID)
		if err != nil {
			return nil, err
		}
		clubUUID = &parsed
	}

	challenges, err := h.service.RefreshChallengesForMembers(ctx, clubservice.ChallengeRefreshRequest{
		ClubUUID:    clubUUID,
		GuildID:     string(payload.GuildID),
		ExternalIDs: []string{string(payload.UserID)},
	})
	if err != nil {
		return nil, err
	}
	results := make([]handlerwrapper.Result, 0, len(challenges)*3)
	for i := range challenges {
		results = append(results, h.challengeResults(clubevents.ChallengeRefreshedV1, &challenges[i])...)
	}
	return results, nil
}

func (h *ClubHandlers) challengeResults(baseTopic string, challenge *clubtypes.ChallengeDetail) []handlerwrapper.Result {
	if challenge == nil {
		return nil
	}
	payload := &clubevents.ChallengeFactPayloadV1{Challenge: *challenge}
	results := []handlerwrapper.Result{{
		Topic:   baseTopic,
		Payload: payload,
	}}

	if challenge.ClubUUID != "" {
		results = append(results, handlerwrapper.Result{
			Topic:   fmt.Sprintf("%s.%s", baseTopic, challenge.ClubUUID),
			Payload: payload,
		})
	}
	if challenge.DiscordGuildID != nil && *challenge.DiscordGuildID != "" {
		results = append(results, handlerwrapper.Result{
			Topic:   fmt.Sprintf("%s.%s", baseTopic, *challenge.DiscordGuildID),
			Payload: payload,
		})
	}
	return results
}

func replyTopicFromContext(ctx context.Context, fallback string) string {
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		return replyTo
	}
	return fallback
}

func parseChallengeListRequest(payload *clubevents.ChallengeListRequestPayloadV1) (clubservice.ChallengeListRequest, error) {
	scope, err := parseChallengeScope(payload.ClubUUID, payload.GuildID)
	if err != nil {
		return clubservice.ChallengeListRequest{}, err
	}
	return clubservice.ChallengeListRequest{
		Scope:    scope,
		Statuses: payload.Statuses,
	}, nil
}

func parseChallengeDetailRequest(payload *clubevents.ChallengeDetailRequestPayloadV1) (clubservice.ChallengeDetailRequest, error) {
	scope, err := parseChallengeScope(payload.ClubUUID, payload.GuildID)
	if err != nil {
		return clubservice.ChallengeDetailRequest{}, err
	}
	challengeID, err := uuid.Parse(payload.ChallengeID)
	if err != nil {
		return clubservice.ChallengeDetailRequest{}, err
	}
	return clubservice.ChallengeDetailRequest{
		Scope:       scope,
		ChallengeID: challengeID,
	}, nil
}

func parseChallengeOpenRequest(payload *clubevents.ChallengeOpenRequestedPayloadV1) (clubservice.ChallengeOpenRequest, error) {
	scope, err := parseChallengeScope(payload.ClubUUID, payload.GuildID)
	if err != nil {
		return clubservice.ChallengeOpenRequest{}, err
	}
	actor, err := parseChallengeActor(payload.ActorUserUUID, payload.ActorExternalID)
	if err != nil {
		return clubservice.ChallengeOpenRequest{}, err
	}
	target, err := parseChallengeActor(payload.TargetUserUUID, payload.TargetExternalID)
	if err != nil {
		return clubservice.ChallengeOpenRequest{}, err
	}
	return clubservice.ChallengeOpenRequest{Scope: scope, Actor: actor, Target: target}, nil
}

func parseChallengeRespondRequest(payload *clubevents.ChallengeRespondRequestedPayloadV1) (clubservice.ChallengeRespondRequest, error) {
	scope, err := parseChallengeScope(payload.ClubUUID, payload.GuildID)
	if err != nil {
		return clubservice.ChallengeRespondRequest{}, err
	}
	actor, err := parseChallengeActor(payload.ActorUserUUID, payload.ActorExternalID)
	if err != nil {
		return clubservice.ChallengeRespondRequest{}, err
	}
	challengeID, err := uuid.Parse(payload.ChallengeID)
	if err != nil {
		return clubservice.ChallengeRespondRequest{}, err
	}
	return clubservice.ChallengeRespondRequest{
		Scope:       scope,
		Actor:       actor,
		ChallengeID: challengeID,
		Response:    payload.Response,
	}, nil
}

func parseChallengeRoundLinkRequest(payload *clubevents.ChallengeRoundLinkRequestedPayloadV1) (clubservice.ChallengeRoundLinkRequest, error) {
	action, err := parseChallengeActionRequest(payload.ClubUUID, payload.GuildID, payload.ActorUserUUID, payload.ActorExternalID, payload.ChallengeID)
	if err != nil {
		return clubservice.ChallengeRoundLinkRequest{}, err
	}
	roundID, err := uuid.Parse(payload.RoundID)
	if err != nil {
		return clubservice.ChallengeRoundLinkRequest{}, err
	}
	return clubservice.ChallengeRoundLinkRequest{
		Scope:       action.Scope,
		Actor:       action.Actor,
		ChallengeID: action.ChallengeID,
		RoundID:     roundID,
	}, nil
}

func parseChallengeActionRequest(clubUUID, guildID, actorUserUUID, actorExternalID, challengeID string) (clubservice.ChallengeActionRequest, error) {
	scope, err := parseChallengeScope(clubUUID, guildID)
	if err != nil {
		return clubservice.ChallengeActionRequest{}, err
	}
	actor, err := parseChallengeActor(actorUserUUID, actorExternalID)
	if err != nil {
		return clubservice.ChallengeActionRequest{}, err
	}
	parsedChallengeID, err := uuid.Parse(challengeID)
	if err != nil {
		return clubservice.ChallengeActionRequest{}, err
	}
	return clubservice.ChallengeActionRequest{
		Scope:       scope,
		Actor:       actor,
		ChallengeID: parsedChallengeID,
	}, nil
}

func parseChallengeScope(clubUUID, guildID string) (clubservice.ChallengeScope, error) {
	scope := clubservice.ChallengeScope{GuildID: guildID}
	if clubUUID == "" {
		return scope, nil
	}
	parsed, err := uuid.Parse(clubUUID)
	if err != nil {
		return clubservice.ChallengeScope{}, err
	}
	scope.ClubUUID = &parsed
	return scope, nil
}

func parseChallengeActor(userUUID, externalID string) (clubservice.ChallengeActorIdentity, error) {
	actor := clubservice.ChallengeActorIdentity{ExternalID: externalID}
	if userUUID == "" {
		return actor, nil
	}
	parsed, err := uuid.Parse(userUUID)
	if err != nil {
		return clubservice.ChallengeActorIdentity{}, err
	}
	actor.UserUUID = &parsed
	return actor, nil
}
