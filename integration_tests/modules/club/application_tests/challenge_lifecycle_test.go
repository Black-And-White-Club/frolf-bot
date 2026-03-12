package clubintegrationtests

import (
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	clubqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/queue"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeLifecycle_OpenAcceptLinkResetAndComplete(t *testing.T) {
	deps := SetupTestClubService(t)

	club := seedClub(t, deps, false)
	challenger := seedClubMember(t, deps, club.UUID, "challenger-ext", sharedtypes.UserRoleUser)
	defender := seedClubMember(t, deps, club.UUID, "defender-ext", sharedtypes.UserRoleUser)

	deps.TagReader.SetRows([]leaderboardservice.MemberTagView{
		{MemberID: "challenger-ext", Tag: intPtr(18)},
		{MemberID: "defender-ext", Tag: intPtr(7)},
	})

	roundID := uuid.New()
	deps.RoundReader.SetRound(&roundtypes.Round{
		ID:    sharedtypes.RoundID(roundID),
		State: roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: sharedtypes.DiscordID("challenger-ext"), Response: roundtypes.ResponseAccept},
			{UserID: sharedtypes.DiscordID("defender-ext"), Response: roundtypes.ResponseAccept},
		},
	})

	openDetail, err := deps.Service.OpenChallenge(deps.Ctx, clubservice.ChallengeOpenRequest{
		Scope:  clubservice.ChallengeScope{ClubUUID: &club.UUID},
		Actor:  clubservice.ChallengeActorIdentity{UserUUID: &challenger.UserUUID},
		Target: clubservice.ChallengeActorIdentity{UserUUID: &defender.UserUUID},
	})
	require.NoError(t, err)
	require.NotNil(t, openDetail)

	challengeID := uuid.MustParse(openDetail.ID)
	assert.Equal(t, 1, countPendingChallengeJobs(t, deps, challengeID, clubqueue.OpenChallengeExpiryJob{}.Kind()))
	assert.Equal(t, 0, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))

	acceptedDetail, err := deps.Service.RespondToChallenge(deps.Ctx, clubservice.ChallengeRespondRequest{
		Scope:       clubservice.ChallengeScope{ClubUUID: &club.UUID},
		Actor:       clubservice.ChallengeActorIdentity{UserUUID: &defender.UserUUID},
		ChallengeID: challengeID,
		Response:    clubservice.ChallengeResponseAccept,
	})
	require.NoError(t, err)
	require.NotNil(t, acceptedDetail)
	assert.Equal(t, 0, countPendingChallengeJobs(t, deps, challengeID, clubqueue.OpenChallengeExpiryJob{}.Kind()))
	assert.Equal(t, 1, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))

	linkedDetail, err := deps.Service.LinkChallengeRound(deps.Ctx, clubservice.ChallengeRoundLinkRequest{
		Scope:       clubservice.ChallengeScope{ClubUUID: &club.UUID},
		Actor:       clubservice.ChallengeActorIdentity{UserUUID: &challenger.UserUUID},
		ChallengeID: challengeID,
		RoundID:     roundID,
	})
	require.NoError(t, err)
	require.NotNil(t, linkedDetail)
	require.NotNil(t, linkedDetail.LinkedRound)
	assert.Equal(t, roundID.String(), linkedDetail.LinkedRound.RoundID)
	summaries, err := deps.Service.ListChallenges(deps.Ctx, clubservice.ChallengeListRequest{
		Scope: clubservice.ChallengeScope{ClubUUID: &club.UUID},
	})
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].LinkedRound)
	assert.Equal(t, roundID.String(), summaries[0].LinkedRound.RoundID)
	assert.Equal(t, 0, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))
	require.NotNil(t, loadActiveChallengeLink(t, deps, challengeID))

	resetDetail, err := deps.Service.ResetChallengeForRound(deps.Ctx, clubservice.ChallengeRoundEventRequest{RoundID: roundID})
	require.NoError(t, err)
	require.NotNil(t, resetDetail)
	assert.Equal(t, 1, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))
	assert.Nil(t, loadActiveChallengeLink(t, deps, challengeID))
	require.NotNil(t, resetDetail.AcceptedExpiresAt)

	relinkedDetail, err := deps.Service.LinkChallengeRound(deps.Ctx, clubservice.ChallengeRoundLinkRequest{
		Scope:       clubservice.ChallengeScope{ClubUUID: &club.UUID},
		Actor:       clubservice.ChallengeActorIdentity{UserUUID: &defender.UserUUID},
		ChallengeID: challengeID,
		RoundID:     roundID,
	})
	require.NoError(t, err)
	require.NotNil(t, relinkedDetail)
	assert.Equal(t, 0, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))

	completedDetail, err := deps.Service.CompleteChallengeForRound(deps.Ctx, clubservice.ChallengeRoundEventRequest{RoundID: roundID})
	require.NoError(t, err)
	require.NotNil(t, completedDetail)
	assert.Equal(t, 0, countPendingChallengeJobs(t, deps, challengeID, clubqueue.AcceptedChallengeExpiryJob{}.Kind()))

	stored := loadChallenge(t, deps, challengeID)
	assert.Equal(t, "completed", string(stored.Status))
	require.NotNil(t, stored.CompletedAt)
}

func TestChallengeLifecycle_ExpireChallenge(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, deps ClubTestDeps, clubUUID uuid.UUID, challengerUserUUID, defenderUserUUID uuid.UUID) uuid.UUID
	}{
		{
			name: "expires open challenge after deadline passes",
			setup: func(t *testing.T, deps ClubTestDeps, clubUUID uuid.UUID, challengerUserUUID, defenderUserUUID uuid.UUID) uuid.UUID {
				detail, err := deps.Service.OpenChallenge(deps.Ctx, clubservice.ChallengeOpenRequest{
					Scope:  clubservice.ChallengeScope{ClubUUID: &clubUUID},
					Actor:  clubservice.ChallengeActorIdentity{UserUUID: &challengerUserUUID},
					Target: clubservice.ChallengeActorIdentity{UserUUID: &defenderUserUUID},
				})
				require.NoError(t, err)
				challengeID := uuid.MustParse(detail.ID)
				updateChallengeExpiry(t, deps, challengeID, "open_expires_at", time.Now().UTC().Add(-1*time.Minute))
				return challengeID
			},
		},
		{
			name: "expires accepted challenge without round link after deadline passes",
			setup: func(t *testing.T, deps ClubTestDeps, clubUUID uuid.UUID, challengerUserUUID, defenderUserUUID uuid.UUID) uuid.UUID {
				detail, err := deps.Service.OpenChallenge(deps.Ctx, clubservice.ChallengeOpenRequest{
					Scope:  clubservice.ChallengeScope{ClubUUID: &clubUUID},
					Actor:  clubservice.ChallengeActorIdentity{UserUUID: &challengerUserUUID},
					Target: clubservice.ChallengeActorIdentity{UserUUID: &defenderUserUUID},
				})
				require.NoError(t, err)
				challengeID := uuid.MustParse(detail.ID)

				_, err = deps.Service.RespondToChallenge(deps.Ctx, clubservice.ChallengeRespondRequest{
					Scope:       clubservice.ChallengeScope{ClubUUID: &clubUUID},
					Actor:       clubservice.ChallengeActorIdentity{UserUUID: &defenderUserUUID},
					ChallengeID: challengeID,
					Response:    clubservice.ChallengeResponseAccept,
				})
				require.NoError(t, err)

				updateChallengeExpiry(t, deps, challengeID, "accepted_expires_at", time.Now().UTC().Add(-1*time.Minute))
				return challengeID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestClubService(t)

			club := seedClub(t, deps, false)
			challenger := seedClubMember(t, deps, club.UUID, "challenger-ext", sharedtypes.UserRoleUser)
			defender := seedClubMember(t, deps, club.UUID, "defender-ext", sharedtypes.UserRoleUser)

			deps.TagReader.SetRows([]leaderboardservice.MemberTagView{
				{MemberID: "challenger-ext", Tag: intPtr(18)},
				{MemberID: "defender-ext", Tag: intPtr(7)},
			})

			challengeID := tt.setup(t, deps, club.UUID, challenger.UserUUID, defender.UserUUID)

			detail, err := deps.Service.ExpireChallenge(deps.Ctx, clubservice.ChallengeExpireRequest{
				ChallengeID: challengeID,
				Reason:      "integration_expiry",
			})
			require.NoError(t, err)
			require.NotNil(t, detail)
			assert.Equal(t, "expired", string(detail.Status))

			stored := loadChallenge(t, deps, challengeID)
			assert.Equal(t, "expired", string(stored.Status))
		})
	}
}
