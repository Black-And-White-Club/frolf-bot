package clubservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func (s *ClubService) ListChallenges(ctx context.Context, req ChallengeListRequest) ([]clubtypes.ChallengeSummary, error) {
	return withValueTelemetry(s, ctx, "ListChallenges", challengeScopeIdentifier(req.Scope), func(ctx context.Context) ([]clubtypes.ChallengeSummary, error) {
		club, err := s.resolveChallengeClub(ctx, nil, req.Scope)
		if err != nil {
			return nil, err
		}

		statuses := req.Statuses
		if len(statuses) == 0 {
			statuses = []clubtypes.ChallengeStatus{
				clubtypes.ChallengeStatusOpen,
				clubtypes.ChallengeStatusAccepted,
			}
		}

		challenges, err := s.repo.ListChallenges(ctx, nil, club.UUID, statuses)
		if err != nil {
			return nil, err
		}
		if len(challenges) == 0 {
			return nil, nil
		}

		return s.buildChallengeSummaries(ctx, club, challenges)
	})
}

// buildChallengeSummaries builds summaries for a slice of challenges using a single
// batch membership fetch and a single tag-list fetch, avoiding per-challenge N+1 queries.
func (s *ClubService) buildChallengeSummaries(ctx context.Context, club *clubdb.Club, challenges []*clubdb.ClubChallenge) ([]clubtypes.ChallengeSummary, error) {
	// Collect all unique user UUIDs across all challenges.
	seen := make(map[uuid.UUID]struct{}, len(challenges)*2)
	allUserUUIDs := make([]uuid.UUID, 0, len(challenges)*2)
	for _, c := range challenges {
		for _, uid := range []uuid.UUID{c.ChallengerUserUUID, c.DefenderUserUUID} {
			if _, ok := seen[uid]; !ok {
				seen[uid] = struct{}{}
				allUserUUIDs = append(allUserUUIDs, uid)
			}
		}
	}

	// Batch fetch all memberships in one call.
	memberships, err := s.userRepo.GetClubMembershipsByUserUUIDs(ctx, nil, allUserUUIDs)
	if err != nil {
		return nil, err
	}
	membershipMap := make(map[uuid.UUID]*userdb.ClubMembership, len(memberships))
	for _, m := range memberships {
		if m.ClubUUID == club.UUID {
			membershipMap[m.UserUUID] = m
		}
	}

	// Fetch the tag list once for all users.
	tagsByUserUUID, err := s.batchTagsByUserUUID(ctx, club, membershipMap)
	if err != nil {
		return nil, err
	}

	summaries := make([]clubtypes.ChallengeSummary, 0, len(challenges))
	for _, challenge := range challenges {
		linkedRound, err := s.loadActiveChallengeRoundLinkView(ctx, nil, challenge.UUID)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, buildSummaryFromMaps(club, challenge, membershipMap, tagsByUserUUID, linkedRound))
	}
	return summaries, nil
}

// batchTagsByUserUUID fetches the leaderboard tag list once and returns a map of
// userUUID → current tag, using the pre-fetched membershipMap for ExternalID lookup.
func (s *ClubService) batchTagsByUserUUID(ctx context.Context, club *clubdb.Club, membershipMap map[uuid.UUID]*userdb.ClubMembership) (map[uuid.UUID]*int, error) {
	result := make(map[uuid.UUID]*int, len(membershipMap))
	if s.leaderboardReader == nil {
		return result, fmt.Errorf("leaderboard reader unavailable")
	}

	clubUUID := club.UUID.String()
	tagRows, err := s.leaderboardReader.GetTagList(ctx, s.challengeGuildID(club), &clubUUID)
	if err != nil {
		return nil, err
	}

	tagsByExternalID := make(map[string]*int, len(tagRows))
	for _, row := range tagRows {
		tagsByExternalID[row.MemberID] = cloneInt(row.Tag)
	}
	for userUUID, m := range membershipMap {
		if m.ExternalID != nil {
			result[userUUID] = cloneInt(tagsByExternalID[*m.ExternalID])
		}
	}
	return result, nil
}

// buildSummaryFromMaps constructs a ChallengeSummary from pre-fetched membership and tag maps.
func buildSummaryFromMaps(
	club *clubdb.Club,
	challenge *clubdb.ClubChallenge,
	membershipMap map[uuid.UUID]*userdb.ClubMembership,
	tagsByUserUUID map[uuid.UUID]*int,
	linkedRound *clubtypes.ChallengeRoundLink,
) clubtypes.ChallengeSummary {
	summary := clubtypes.ChallengeSummary{
		ID:                 challenge.UUID.String(),
		ClubUUID:           challenge.ClubUUID.String(),
		DiscordGuildID:     club.DiscordGuildID,
		Status:             challenge.Status,
		ChallengerUserUUID: challenge.ChallengerUserUUID.String(),
		DefenderUserUUID:   challenge.DefenderUserUUID.String(),
		OriginalTags:       clubtypes.ChallengeTagSnapshot{Challenger: cloneInt(challenge.OriginalChallengerTag), Defender: cloneInt(challenge.OriginalDefenderTag)},
		CurrentTags:        clubtypes.ChallengeTagSnapshot{Challenger: cloneInt(tagsByUserUUID[challenge.ChallengerUserUUID]), Defender: cloneInt(tagsByUserUUID[challenge.DefenderUserUUID])},
		OpenedAt:           challenge.OpenedAt,
		OpenExpiresAt:      challenge.OpenExpiresAt,
		AcceptedAt:         challenge.AcceptedAt,
		AcceptedExpiresAt:  challenge.AcceptedExpiresAt,
		LinkedRound:        linkedRound,
	}
	if m, ok := membershipMap[challenge.ChallengerUserUUID]; ok {
		summary.ChallengerExternalID = m.ExternalID
	}
	if m, ok := membershipMap[challenge.DefenderUserUUID]; ok {
		summary.DefenderExternalID = m.ExternalID
	}
	return summary
}

func (s *ClubService) GetChallengeDetail(ctx context.Context, req ChallengeDetailRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "GetChallengeDetail", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		club, err := s.resolveChallengeClub(ctx, nil, req.Scope)
		if err != nil {
			return nil, err
		}

		challenge, err := s.repo.GetChallengeByUUID(ctx, nil, req.ChallengeID)
		if err != nil {
			if errors.Is(err, clubdb.ErrNotFound) {
				return nil, fmt.Errorf("challenge not found")
			}
			return nil, err
		}
		if challenge.ClubUUID != club.UUID {
			return nil, fmt.Errorf("challenge not found")
		}

		return s.buildChallengeDetail(ctx, nil, club, challenge)
	})
}

func (s *ClubService) OpenChallenge(ctx context.Context, req ChallengeOpenRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "OpenChallenge", challengeScopeIdentifier(req.Scope), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			club, err := s.resolveChallengeClub(ctx, db, req.Scope)
			if err != nil {
				return err
			}

			challengerMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
			if err != nil {
				return err
			}
			defenderMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Target)
			if err != nil {
				return err
			}
			if challengerMembership.UserUUID == defenderMembership.UserUUID {
				return fmt.Errorf("you cannot challenge yourself")
			}
			if err := s.lockChallengeParticipants(ctx, db, club.UUID, challengerMembership.UserUUID, defenderMembership.UserUUID); err != nil {
				return err
			}

			if _, err := s.repo.GetOpenOutgoingChallenge(ctx, db, club.UUID, challengerMembership.UserUUID); err == nil {
				return fmt.Errorf("you already have an open outgoing challenge")
			} else if !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}

			if _, err := s.repo.GetAcceptedChallengeForUser(ctx, db, club.UUID, challengerMembership.UserUUID); err == nil {
				return fmt.Errorf("you already have an accepted challenge")
			} else if !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}
			if _, err := s.repo.GetAcceptedChallengeForUser(ctx, db, club.UUID, defenderMembership.UserUUID); err == nil {
				return fmt.Errorf("that player already has an accepted challenge")
			} else if !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}

			if _, err := s.repo.GetActiveChallengeByPair(ctx, db, club.UUID, challengerMembership.UserUUID, defenderMembership.UserUUID); err == nil {
				return fmt.Errorf("an active challenge already exists for this pair")
			} else if !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}

			currentTags, err := s.getCurrentTagsForUsers(ctx, db, club, []uuid.UUID{challengerMembership.UserUUID, defenderMembership.UserUUID})
			if err != nil {
				return err
			}
			challengerTag := currentTags[challengerMembership.UserUUID]
			defenderTag := currentTags[defenderMembership.UserUUID]
			if challengerTag == nil || defenderTag == nil {
				return fmt.Errorf("both players must currently hold tags")
			}
			if *challengerTag <= *defenderTag {
				return fmt.Errorf("you can only challenge a better tag than your own")
			}

			now := time.Now().UTC()
			challenge := &clubdb.ClubChallenge{
				UUID:                  uuid.New(),
				ClubUUID:              club.UUID,
				ChallengerUserUUID:    challengerMembership.UserUUID,
				DefenderUserUUID:      defenderMembership.UserUUID,
				Status:                clubtypes.ChallengeStatusOpen,
				OriginalChallengerTag: cloneInt(challengerTag),
				OriginalDefenderTag:   cloneInt(defenderTag),
				OpenedAt:              now,
				OpenExpiresAt:         ptrTime(now.Add(challengeOpenTTL)),
				DiscordGuildID:        club.DiscordGuildID,
			}
			if err := s.repo.CreateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			if err != nil {
				return err
			}
			return s.scheduleOpenExpiry(ctx, challenge.UUID, challenge.OpenExpiresAt)
		})
		if err != nil {
			return nil, err
		}

		if s.metrics != nil {
			s.metrics.RecordChallengeOpened(ctx)
		}
		return detail, nil
	})
}

func (s *ClubService) RespondToChallenge(ctx context.Context, req ChallengeRespondRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "RespondToChallenge", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		response := strings.ToLower(strings.TrimSpace(req.Response))
		if response != ChallengeResponseAccept && response != ChallengeResponseDecline {
			return nil, fmt.Errorf("response must be accept or decline")
		}

		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			club, err := s.resolveChallengeClub(ctx, db, req.Scope)
			if err != nil {
				return err
			}
			actorMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
			if err != nil {
				return err
			}
			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if err := s.lockChallengeParticipants(ctx, db, club.UUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID); err != nil {
				return err
			}
			challenge, err = s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.ClubUUID != club.UUID {
				return fmt.Errorf("challenge not found")
			}
			if challenge.DefenderUserUUID != actorMembership.UserUUID {
				return fmt.Errorf("only the challenged player can respond")
			}
			if challenge.Status != clubtypes.ChallengeStatusOpen {
				return fmt.Errorf("only open challenges can be responded to")
			}

			now := time.Now().UTC()
			if response == ChallengeResponseAccept {
				if err := s.ensureNoOtherAcceptedChallenge(ctx, db, club.UUID, challenge.ChallengerUserUUID, "that player already has an accepted challenge"); err != nil {
					return err
				}
				if err := s.ensureNoOtherAcceptedChallenge(ctx, db, club.UUID, challenge.DefenderUserUUID, "you already have an accepted challenge"); err != nil {
					return err
				}
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = ptrTime(now.Add(challengeAcceptTTL))
			} else {
				challenge.Status = clubtypes.ChallengeStatusDeclined
				challenge.AcceptedAt = nil
				challenge.AcceptedExpiresAt = nil
			}
			challenge.OpenExpiresAt = nil
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			if err != nil {
				return err
			}
			if response == ChallengeResponseAccept {
				return s.scheduleAcceptedExpiry(ctx, challenge.UUID, challenge.AcceptedExpiresAt)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		s.cancelChallengeOpenExpiry(ctx, detail.ID)
		if response == ChallengeResponseAccept {
			if s.metrics != nil {
				s.metrics.RecordChallengeAccepted(ctx)
			}
		} else if s.metrics != nil {
			s.metrics.RecordChallengeDeclined(ctx)
		}
		return detail, nil
	})
}

func (s *ClubService) WithdrawChallenge(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "WithdrawChallenge", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		detail, err := s.updateChallengeStatusByAction(ctx, req, clubtypes.ChallengeStatusWithdrawn)
		if err == nil && detail != nil && s.metrics != nil {
			s.metrics.RecordChallengeWithdrawn(ctx)
		}
		return detail, err
	})
}

func (s *ClubService) HideChallenge(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "HideChallenge", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			club, err := s.resolveChallengeClub(ctx, db, req.Scope)
			if err != nil {
				return err
			}
			actorMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
			if err != nil {
				return err
			}
			if err := s.requireAdminOrEditor(ctx, actorMembership.UserUUID, club.UUID); err != nil {
				return err
			}

			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.ClubUUID != club.UUID {
				return fmt.Errorf("challenge not found")
			}

			now := time.Now().UTC()
			challenge.Status = clubtypes.ChallengeStatusHidden
			challenge.HiddenAt = &now
			challenge.HiddenByUserUUID = &actorMembership.UserUUID
			challenge.OpenExpiresAt = nil
			challenge.AcceptedExpiresAt = nil
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			return err
		})
		if err != nil {
			return nil, err
		}

		s.cancelChallengeJobs(ctx, detail.ID)
		if s.metrics != nil {
			s.metrics.RecordChallengeHidden(ctx)
		}
		return detail, nil
	})
}

func (s *ClubService) LinkChallengeRound(ctx context.Context, req ChallengeRoundLinkRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "LinkChallengeRound", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			club, err := s.resolveChallengeClub(ctx, db, req.Scope)
			if err != nil {
				return err
			}
			actorMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
			if err != nil {
				return err
			}
			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.ClubUUID != club.UUID {
				return fmt.Errorf("challenge not found")
			}
			if err := s.lockChallengeParticipants(ctx, db, club.UUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID); err != nil {
				return err
			}
			challenge, err = s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.Status != clubtypes.ChallengeStatusAccepted {
				return fmt.Errorf("only accepted challenges can be linked to a round")
			}
			if err := s.requireChallengeParticipantOrAdmin(ctx, db, club.UUID, actorMembership.UserUUID, challenge); err != nil {
				return err
			}
			if err := s.validateRoundForChallenge(ctx, db, club, challenge, req.RoundID); err != nil {
				return err
			}

			activeLink, err := s.repo.GetActiveChallengeRoundLink(ctx, db, challenge.UUID)
			if err == nil && activeLink.RoundID == req.RoundID {
				detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
				return err
			} else if err != nil && !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}

			existingChallenge, err := s.repo.GetChallengeByActiveRound(ctx, db, req.RoundID)
			if err == nil {
				if existingChallenge.UUID != challenge.UUID {
					return fmt.Errorf("that round is already linked to a challenge")
				}
				detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
				return err
			} else if !errors.Is(err, clubdb.ErrNotFound) {
				return err
			}

			if activeLink != nil {
				now := time.Now().UTC()
				if unlinkErr := s.repo.UnlinkActiveChallengeRound(ctx, db, challenge.UUID, &actorMembership.UserUUID, now); unlinkErr != nil && !errors.Is(unlinkErr, clubdb.ErrNotFound) {
					return unlinkErr
				}
			}

			now := time.Now().UTC()
			if err := s.repo.CreateChallengeRoundLink(ctx, db, &clubdb.ClubChallengeRoundLink{
				UUID:             uuid.New(),
				ChallengeUUID:    challenge.UUID,
				RoundID:          req.RoundID,
				LinkedByUserUUID: &actorMembership.UserUUID,
				LinkedAt:         now,
			}); err != nil {
				return err
			}
			challenge.AcceptedExpiresAt = nil
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			return err
		})
		if err != nil {
			return nil, err
		}

		s.cancelChallengeJobs(ctx, detail.ID)
		if s.metrics != nil {
			s.metrics.RecordChallengeRoundLinked(ctx)
		}
		return detail, nil
	})
}

func (s *ClubService) UnlinkChallengeRound(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "UnlinkChallengeRound", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			club, err := s.resolveChallengeClub(ctx, db, req.Scope)
			if err != nil {
				return err
			}
			actorMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
			if err != nil {
				return err
			}
			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.ClubUUID != club.UUID {
				return fmt.Errorf("challenge not found")
			}
			challenge, err = s.lockAndReloadChallenge(ctx, db, challenge)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if challenge.ClubUUID != club.UUID {
				return fmt.Errorf("challenge not found")
			}
			if challenge.Status != clubtypes.ChallengeStatusAccepted {
				return fmt.Errorf("only accepted challenges can be unlinked")
			}
			if err := s.requireChallengeParticipantOrAdmin(ctx, db, club.UUID, actorMembership.UserUUID, challenge); err != nil {
				return err
			}

			now := time.Now().UTC()
			if err := s.repo.UnlinkActiveChallengeRound(ctx, db, challenge.UUID, &actorMembership.UserUUID, now); err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge has no linked round")
				}
				return err
			}
			challenge.AcceptedExpiresAt = ptrTime(now.Add(challengeAcceptTTL))
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			if err != nil {
				return err
			}
			return s.scheduleAcceptedExpiry(ctx, challenge.UUID, challenge.AcceptedExpiresAt)
		})
		if err != nil {
			return nil, err
		}

		if s.metrics != nil {
			s.metrics.RecordChallengeRoundUnlinked(ctx)
		}
		return detail, nil
	})
}

func (s *ClubService) BindChallengeMessage(ctx context.Context, req ChallengeMessageBindingRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "BindChallengeMessage", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return fmt.Errorf("challenge not found")
				}
				return err
			}
			if err := s.repo.BindChallengeMessage(ctx, db, challenge.UUID, req.GuildID, req.ChannelID, req.MessageID); err != nil {
				return err
			}
			challenge.DiscordGuildID = ptrString(req.GuildID)
			challenge.DiscordChannelID = ptrString(req.ChannelID)
			challenge.DiscordMessageID = ptrString(req.MessageID)

			club, err := s.repo.GetByUUID(ctx, db, challenge.ClubUUID)
			if err != nil {
				return err
			}
			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			return err
		})
		if err != nil {
			return nil, err
		}
		return detail, nil
	})
}

func (s *ClubService) ExpireChallenge(ctx context.Context, req ChallengeExpireRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "ExpireChallenge", req.ChallengeID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
			if err != nil {
				// Challenge already gone — expiry is idempotent, treat as success.
				if errors.Is(err, clubdb.ErrNotFound) {
					return nil
				}
				return err
			}
			if challenge.Status == clubtypes.ChallengeStatusOpen || challenge.Status == clubtypes.ChallengeStatusAccepted {
				challenge, err = s.lockAndReloadChallenge(ctx, db, challenge)
				if err != nil {
					// Challenge already gone — expiry is idempotent, treat as success.
					if errors.Is(err, clubdb.ErrNotFound) {
						return nil
					}
					return err
				}
			}

			now := time.Now().UTC()
			switch challenge.Status {
			case clubtypes.ChallengeStatusOpen:
				// Guard: worker may fire slightly early; re-check the wall clock.
				if challenge.OpenExpiresAt == nil || now.Before(*challenge.OpenExpiresAt) {
					return nil
				}
				challenge.Status = clubtypes.ChallengeStatusExpired
				challenge.OpenExpiresAt = nil
			case clubtypes.ChallengeStatusAccepted:
				if _, err := s.repo.GetActiveChallengeRoundLink(ctx, db, challenge.UUID); err == nil {
					return nil
				} else if !errors.Is(err, clubdb.ErrNotFound) {
					return err
				}
				if challenge.AcceptedExpiresAt == nil || now.Before(*challenge.AcceptedExpiresAt) {
					return nil
				}
				challenge.Status = clubtypes.ChallengeStatusExpired
				challenge.AcceptedExpiresAt = nil
			default:
				return nil
			}

			club, err := s.repo.GetByUUID(ctx, db, challenge.ClubUUID)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}
			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			return err
		})
		if err != nil {
			return nil, err
		}
		if detail != nil && s.metrics != nil {
			s.metrics.RecordChallengeExpired(ctx, req.Reason)
		}
		return detail, nil
	})
}

func (s *ClubService) CompleteChallengeForRound(ctx context.Context, req ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "CompleteChallengeForRound", req.RoundID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			challenge, err := s.lockAndReloadChallengeByRound(ctx, db, req.RoundID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return nil
				}
				return err
			}
			if challenge.Status != clubtypes.ChallengeStatusAccepted {
				return nil
			}
			now := time.Now().UTC()
			challenge.Status = clubtypes.ChallengeStatusCompleted
			challenge.CompletedAt = &now
			challenge.AcceptedExpiresAt = nil
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			club, err := s.repo.GetByUUID(ctx, db, challenge.ClubUUID)
			if err != nil {
				return err
			}
			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			return err
		})
		if err != nil {
			return nil, err
		}
		if detail != nil {
			s.cancelChallengeJobs(ctx, detail.ID)
			if s.metrics != nil {
				s.metrics.RecordChallengeCompleted(ctx)
			}
		}
		return detail, nil
	})
}

func (s *ClubService) ResetChallengeForRound(ctx context.Context, req ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "ResetChallengeForRound", req.RoundID.String(), func(ctx context.Context) (*clubtypes.ChallengeDetail, error) {
		var detail *clubtypes.ChallengeDetail
		err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
			challenge, err := s.lockAndReloadChallengeByRound(ctx, db, req.RoundID)
			if err != nil {
				if errors.Is(err, clubdb.ErrNotFound) {
					return nil
				}
				return err
			}
			if challenge.Status != clubtypes.ChallengeStatusAccepted {
				return nil
			}

			now := time.Now().UTC()
			if err := s.repo.UnlinkActiveChallengeRound(ctx, db, challenge.UUID, nil, now); err != nil {
				return err
			}
			challenge.AcceptedExpiresAt = ptrTime(now.Add(challengeAcceptTTL))
			if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
				return err
			}

			club, err := s.repo.GetByUUID(ctx, db, challenge.ClubUUID)
			if err != nil {
				return err
			}
			detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
			if err != nil {
				return err
			}
			return s.scheduleAcceptedExpiry(ctx, challenge.UUID, challenge.AcceptedExpiresAt)
		})
		if err != nil {
			return nil, err
		}
		if detail != nil {
			if s.metrics != nil {
				s.metrics.RecordChallengeRoundUnlinked(ctx)
			}
		}
		return detail, nil
	})
}

func (s *ClubService) RefreshChallengesForMembers(ctx context.Context, req ChallengeRefreshRequest) ([]clubtypes.ChallengeDetail, error) {
	return withValueTelemetry(s, ctx, "RefreshChallengesForMembers", challengeRefreshIdentifier(req), func(ctx context.Context) ([]clubtypes.ChallengeDetail, error) {
		if req.ClubUUID == nil && req.GuildID == "" {
			return nil, nil
		}
		if len(req.ExternalIDs) == 0 {
			return nil, nil
		}

		clubUUID, err := s.resolveChallengeRefreshClubUUID(ctx, req)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}
		club, err := s.repo.GetByUUID(ctx, nil, clubUUID)
		if err != nil {
			return nil, err
		}

		userUUIDs := make([]uuid.UUID, 0, len(req.ExternalIDs))
		for _, externalID := range req.ExternalIDs {
			membership, membershipErr := s.userRepo.GetClubMembershipByExternalID(ctx, nil, externalID, club.UUID)
			if membershipErr != nil {
				if errors.Is(membershipErr, userdb.ErrNotFound) {
					continue
				}
				return nil, membershipErr
			}
			userUUIDs = append(userUUIDs, membership.UserUUID)
		}
		if len(userUUIDs) == 0 {
			return nil, nil
		}

		challenges, err := s.repo.ListActiveChallengesByUsers(ctx, nil, club.UUID, userUUIDs)
		if err != nil {
			return nil, err
		}

		details := make([]clubtypes.ChallengeDetail, 0, len(challenges))
		seen := make(map[string]struct{}, len(challenges))
		for _, challenge := range challenges {
			if _, ok := seen[challenge.UUID.String()]; ok {
				continue
			}
			seen[challenge.UUID.String()] = struct{}{}
			detail, detailErr := s.buildChallengeDetail(ctx, nil, club, challenge)
			if detailErr != nil {
				return nil, detailErr
			}
			details = append(details, *detail)
		}
		if s.metrics != nil {
			s.metrics.RecordChallengeRefreshed(ctx, len(details))
		}
		return details, nil
	})
}

func (s *ClubService) ensureNoOtherAcceptedChallenge(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID, message string) error {
	existing, err := s.repo.GetAcceptedChallengeForUser(ctx, db, clubUUID, userUUID)
	if err == nil && existing != nil {
		return errors.New(message)
	}
	if errors.Is(err, clubdb.ErrNotFound) {
		return nil
	}
	return err
}

func (s *ClubService) resolveChallengeRefreshClubUUID(ctx context.Context, req ChallengeRefreshRequest) (uuid.UUID, error) {
	if req.ClubUUID != nil && *req.ClubUUID != uuid.Nil {
		return *req.ClubUUID, nil
	}

	if parsedGuildUUID, err := uuid.Parse(strings.TrimSpace(req.GuildID)); err == nil {
		return parsedGuildUUID, nil
	}

	return s.userRepo.GetClubUUIDByDiscordGuildID(ctx, nil, sharedtypes.GuildID(req.GuildID))
}

func (s *ClubService) updateChallengeStatusByAction(ctx context.Context, req ChallengeActionRequest, nextStatus clubtypes.ChallengeStatus) (*clubtypes.ChallengeDetail, error) {
	var detail *clubtypes.ChallengeDetail
	err := s.runChallengeTx(ctx, func(ctx context.Context, db bun.IDB) error {
		club, err := s.resolveChallengeClub(ctx, db, req.Scope)
		if err != nil {
			return err
		}
		actorMembership, err := s.resolveChallengeMember(ctx, db, club.UUID, req.Actor)
		if err != nil {
			return err
		}
		challenge, err := s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
		if err != nil {
			if errors.Is(err, clubdb.ErrNotFound) {
				return fmt.Errorf("challenge not found")
			}
			return err
		}
		if challenge.ClubUUID != club.UUID {
			return fmt.Errorf("challenge not found")
		}
		if err := s.lockChallengeParticipants(ctx, db, club.UUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID); err != nil {
			return err
		}
		challenge, err = s.repo.GetChallengeByUUID(ctx, db, req.ChallengeID)
		if err != nil {
			if errors.Is(err, clubdb.ErrNotFound) {
				return fmt.Errorf("challenge not found")
			}
			return err
		}
		if challenge.ClubUUID != club.UUID {
			return fmt.Errorf("challenge not found")
		}
		if !slices.Contains([]clubtypes.ChallengeStatus{clubtypes.ChallengeStatusOpen, clubtypes.ChallengeStatusAccepted}, challenge.Status) {
			return fmt.Errorf("only open or accepted challenges can be updated")
		}
		if err := s.requireChallengeParticipantOrAdmin(ctx, db, club.UUID, actorMembership.UserUUID, challenge); err != nil {
			return err
		}

		challenge.Status = nextStatus
		challenge.OpenExpiresAt = nil
		challenge.AcceptedExpiresAt = nil
		if err := s.repo.UpdateChallenge(ctx, db, challenge); err != nil {
			return err
		}
		detail, err = s.buildChallengeDetail(ctx, db, club, challenge)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.cancelChallengeJobs(ctx, detail.ID)
	return detail, nil
}

func (s *ClubService) resolveChallengeClub(ctx context.Context, db bun.IDB, scope ChallengeScope) (*clubdb.Club, error) {
	if scope.ClubUUID != nil && *scope.ClubUUID != uuid.Nil {
		return s.repo.GetByUUID(ctx, db, *scope.ClubUUID)
	}
	if scope.GuildID != "" {
		return s.repo.GetByDiscordGuildID(ctx, db, scope.GuildID)
	}
	return nil, fmt.Errorf("club scope is required")
}

func (s *ClubService) resolveChallengeMember(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, actor ChallengeActorIdentity) (*userdb.ClubMembership, error) {
	if actor.UserUUID != nil && *actor.UserUUID != uuid.Nil {
		membership, err := s.userRepo.GetClubMembership(ctx, db, *actor.UserUUID, clubUUID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, fmt.Errorf("user is not a member of this club")
			}
			return nil, err
		}
		return membership, nil
	}
	if actor.ExternalID != "" {
		membership, err := s.userRepo.GetClubMembershipByExternalID(ctx, db, actor.ExternalID, clubUUID)
		if err != nil {
			if errors.Is(err, userdb.ErrNotFound) {
				return nil, fmt.Errorf("user is not a member of this club")
			}
			return nil, err
		}
		return membership, nil
	}
	return nil, fmt.Errorf("actor identity is required")
}

func (s *ClubService) requireChallengeParticipantOrAdmin(ctx context.Context, db bun.IDB, clubUUID, actorUserUUID uuid.UUID, challenge *clubdb.ClubChallenge) error {
	if challenge.ChallengerUserUUID == actorUserUUID || challenge.DefenderUserUUID == actorUserUUID {
		return nil
	}
	membership, err := s.userRepo.GetClubMembership(ctx, db, actorUserUUID, clubUUID)
	if err != nil {
		return fmt.Errorf("forbidden")
	}
	if membership.Role != sharedtypes.UserRoleAdmin && membership.Role != sharedtypes.UserRoleEditor {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func (s *ClubService) validateRoundForChallenge(ctx context.Context, db bun.IDB, club *clubdb.Club, challenge *clubdb.ClubChallenge, roundID uuid.UUID) error {
	if s.roundReader == nil {
		return fmt.Errorf("round reader unavailable")
	}

	guildID := s.challengeGuildID(club)
	result, err := s.roundReader.GetRound(ctx, guildID, sharedtypes.RoundID(roundID))
	if err != nil {
		return err
	}
	if result.IsFailure() || result.Success == nil || *result.Success == nil {
		return fmt.Errorf("round not found")
	}
	round := *result.Success
	if round.State == roundtypes.RoundStateDeleted || round.State == roundtypes.RoundStateFinalized {
		return fmt.Errorf("only upcoming or in-progress rounds can be linked")
	}
	return nil
}

func (s *ClubService) buildChallengeDetail(ctx context.Context, db bun.IDB, club *clubdb.Club, challenge *clubdb.ClubChallenge) (*clubtypes.ChallengeDetail, error) {
	membershipMap, err := s.loadChallengeMemberships(ctx, db, club.UUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID)
	if err != nil {
		return nil, err
	}

	currentTags, err := s.getCurrentTagsForUsers(ctx, db, club, []uuid.UUID{challenge.ChallengerUserUUID, challenge.DefenderUserUUID})
	if err != nil {
		return nil, err
	}

	detail := &clubtypes.ChallengeDetail{
		ChallengeSummary: clubtypes.ChallengeSummary{
			ID:                 challenge.UUID.String(),
			ClubUUID:           challenge.ClubUUID.String(),
			DiscordGuildID:     club.DiscordGuildID,
			Status:             challenge.Status,
			ChallengerUserUUID: challenge.ChallengerUserUUID.String(),
			DefenderUserUUID:   challenge.DefenderUserUUID.String(),
			OriginalTags:       clubtypes.ChallengeTagSnapshot{Challenger: cloneInt(challenge.OriginalChallengerTag), Defender: cloneInt(challenge.OriginalDefenderTag)},
			CurrentTags:        clubtypes.ChallengeTagSnapshot{Challenger: cloneInt(currentTags[challenge.ChallengerUserUUID]), Defender: cloneInt(currentTags[challenge.DefenderUserUUID])},
			OpenedAt:           challenge.OpenedAt,
			OpenExpiresAt:      challenge.OpenExpiresAt,
			AcceptedAt:         challenge.AcceptedAt,
			AcceptedExpiresAt:  challenge.AcceptedExpiresAt,
		},
		CompletedAt: challenge.CompletedAt,
		HiddenAt:    challenge.HiddenAt,
	}

	if hiddenBy := challenge.HiddenByUserUUID; hiddenBy != nil {
		value := hiddenBy.String()
		detail.HiddenByUserID = &value
	}
	if challenge.DiscordGuildID != nil && challenge.DiscordChannelID != nil && challenge.DiscordMessageID != nil {
		detail.MessageBinding = &clubtypes.ChallengeMessageBinding{
			GuildID:   *challenge.DiscordGuildID,
			ChannelID: *challenge.DiscordChannelID,
			MessageID: *challenge.DiscordMessageID,
		}
	}

	if membership, ok := membershipMap[challenge.ChallengerUserUUID]; ok {
		detail.ChallengerExternalID = membership.ExternalID
	}
	if membership, ok := membershipMap[challenge.DefenderUserUUID]; ok {
		detail.DefenderExternalID = membership.ExternalID
	}

	detail.LinkedRound, err = s.loadActiveChallengeRoundLinkView(ctx, db, challenge.UUID)
	if err != nil {
		return nil, err
	}

	return detail, nil
}

func (s *ClubService) loadActiveChallengeRoundLinkView(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubtypes.ChallengeRoundLink, error) {
	link, err := s.repo.GetActiveChallengeRoundLink(ctx, db, challengeUUID)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	view := &clubtypes.ChallengeRoundLink{
		RoundID:    link.RoundID.String(),
		LinkedAt:   link.LinkedAt,
		UnlinkedAt: link.UnlinkedAt,
		IsActive:   link.UnlinkedAt == nil,
	}
	if link.LinkedByUserUUID != nil {
		value := link.LinkedByUserUUID.String()
		view.LinkedBy = &value
	}
	if link.UnlinkedByUserUUID != nil {
		value := link.UnlinkedByUserUUID.String()
		view.UnlinkedBy = &value
	}
	return view, nil
}

func (s *ClubService) loadChallengeMemberships(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs ...uuid.UUID) (map[uuid.UUID]*userdb.ClubMembership, error) {
	memberships, err := s.userRepo.GetClubMembershipsByUserUUIDs(ctx, db, userUUIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*userdb.ClubMembership, len(userUUIDs))
	for _, membership := range memberships {
		if membership.ClubUUID != clubUUID {
			continue
		}
		result[membership.UserUUID] = membership
	}
	return result, nil
}

func (s *ClubService) getCurrentTagsForUsers(ctx context.Context, db bun.IDB, club *clubdb.Club, userUUIDs []uuid.UUID) (map[uuid.UUID]*int, error) {
	result := make(map[uuid.UUID]*int, len(userUUIDs))
	if s.leaderboardReader == nil {
		return result, fmt.Errorf("leaderboard reader unavailable")
	}

	membershipMap, err := s.loadChallengeMemberships(ctx, db, club.UUID, userUUIDs...)
	if err != nil {
		return nil, err
	}

	clubUUID := club.UUID.String()
	tagRows, err := s.leaderboardReader.GetTagList(ctx, s.challengeGuildID(club), &clubUUID)
	if err != nil {
		return nil, err
	}

	tagsByExternalID := make(map[string]*int, len(tagRows))
	for _, row := range tagRows {
		tagsByExternalID[row.MemberID] = cloneInt(row.Tag)
	}

	for _, userUUID := range userUUIDs {
		membership, ok := membershipMap[userUUID]
		if !ok || membership.ExternalID == nil {
			result[userUUID] = nil
			continue
		}
		result[userUUID] = cloneInt(tagsByExternalID[*membership.ExternalID])
	}
	return result, nil
}

func (s *ClubService) lockChallengeParticipants(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs ...uuid.UUID) error {
	if db == nil {
		return nil
	}

	keys := []string{fmt.Sprintf("club-challenge:club:%s", clubUUID.String())}
	seen := map[string]struct{}{
		keys[0]: {},
	}

	for _, userUUID := range userUUIDs {
		if userUUID == uuid.Nil {
			continue
		}
		key := fmt.Sprintf("club-challenge:user:%s:%s", clubUUID.String(), userUUID.String())
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	slices.Sort(keys)
	for _, key := range keys {
		if _, err := db.NewRaw("SELECT pg_advisory_xact_lock(('x' || substr(md5(?), 1, 16))::bit(64)::bigint)", key).Exec(ctx); err != nil {
			return fmt.Errorf("acquire challenge lock: %w", err)
		}
	}

	return nil
}

func (s *ClubService) lockAndReloadChallenge(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) (*clubdb.ClubChallenge, error) {
	if challenge == nil {
		return nil, clubdb.ErrNotFound
	}
	if err := s.lockChallengeParticipants(ctx, db, challenge.ClubUUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID); err != nil {
		return nil, err
	}
	return s.repo.GetChallengeByUUID(ctx, db, challenge.UUID)
}

func (s *ClubService) lockAndReloadChallengeByRound(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
	challenge, err := s.repo.GetChallengeByActiveRound(ctx, db, roundID)
	if err != nil {
		return nil, err
	}
	if err := s.lockChallengeParticipants(ctx, db, challenge.ClubUUID, challenge.ChallengerUserUUID, challenge.DefenderUserUUID); err != nil {
		return nil, err
	}
	return s.repo.GetChallengeByActiveRound(ctx, db, roundID)
}

func roundParticipantIDs(round *roundtypes.Round) map[sharedtypes.DiscordID]struct{} {
	participantIDs := make(map[sharedtypes.DiscordID]struct{}, len(round.Participants))
	for _, participant := range round.Participants {
		if participant.UserID == "" {
			continue
		}
		participantIDs[participant.UserID] = struct{}{}
	}
	for _, team := range round.Teams {
		for _, member := range team.Members {
			if member.UserID == nil || *member.UserID == "" {
				continue
			}
			participantIDs[*member.UserID] = struct{}{}
		}
	}
	return participantIDs
}

func (s *ClubService) challengeGuildID(club *clubdb.Club) sharedtypes.GuildID {
	if club.DiscordGuildID != nil && *club.DiscordGuildID != "" {
		return sharedtypes.GuildID(*club.DiscordGuildID)
	}
	return sharedtypes.GuildID(club.UUID.String())
}

func (s *ClubService) scheduleOpenExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt *time.Time) error {
	if s.queueService == nil || expiresAt == nil {
		return nil
	}
	if err := s.queueService.ScheduleOpenExpiry(ctx, challengeID, *expiresAt); err != nil {
		return fmt.Errorf("schedule open challenge expiry: %w", err)
	}
	return nil
}

func (s *ClubService) scheduleAcceptedExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt *time.Time) error {
	if s.queueService == nil || expiresAt == nil {
		return nil
	}
	if err := s.queueService.ScheduleAcceptedExpiry(ctx, challengeID, *expiresAt); err != nil {
		return fmt.Errorf("schedule accepted challenge expiry: %w", err)
	}
	return nil
}

func (s *ClubService) cancelChallengeJobs(ctx context.Context, challengeID string) {
	if s.queueService == nil {
		return
	}
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return
	}
	if err := s.queueService.CancelChallengeJobs(ctx, id); err != nil {
		s.logger.WarnContext(ctx, "failed to cancel challenge jobs", slog.String("challenge_id", challengeID), slog.String("error", err.Error()))
	}
}

func (s *ClubService) cancelChallengeOpenExpiry(ctx context.Context, challengeID string) {
	if s.queueService == nil {
		return
	}
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return
	}
	if err := s.queueService.CancelOpenExpiry(ctx, id); err != nil {
		s.logger.WarnContext(ctx, "failed to cancel open challenge expiry", slog.String("challenge_id", challengeID), slog.String("error", err.Error()))
	}
}

func challengeScopeIdentifier(scope ChallengeScope) string {
	if scope.ClubUUID != nil && *scope.ClubUUID != uuid.Nil {
		return scope.ClubUUID.String()
	}
	if scope.GuildID != "" {
		return scope.GuildID
	}
	return "unknown"
}

func challengeRefreshIdentifier(req ChallengeRefreshRequest) string {
	if req.ClubUUID != nil && *req.ClubUUID != uuid.Nil {
		return req.ClubUUID.String()
	}
	if req.GuildID != "" {
		return req.GuildID
	}
	return "unknown"
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func ptrTime(value time.Time) *time.Time {
	copyValue := value
	return &copyValue
}

func ptrString(value string) *string {
	copyValue := value
	return &copyValue
}

func (s *ClubService) runChallengeTx(ctx context.Context, fn func(ctx context.Context, db bun.IDB) error) error {
	if s.db == nil {
		return fn(ctx, nil)
	}
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, tx)
	})
}
