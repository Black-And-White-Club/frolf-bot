package rounddb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func (r *Impl) RoundHasGroups(
	ctx context.Context,
	db bun.IDB,
	roundID sharedtypes.RoundID,
) (bool, error) {
	if db == nil {
		db = r.db
	}
	count, err := db.NewSelect().
		Model((*RoundGroup)(nil)).
		Where("round_id = ?", roundID).
		Count(ctx)
	return count > 0, err
}

func (r *Impl) CreateRoundGroups(
	ctx context.Context,
	db bun.IDB,
	roundID sharedtypes.RoundID,
	participants []roundtypes.Participant,
) error {
	if db == nil {
		db = r.db
	}
	if len(participants) == 0 {
		return nil
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, p := range participants {
			displayName := roundtypes.DisplayName(p.UserIDPointer(), p.RawNameString())
			groupID := uuid.New()

			group := &RoundGroup{
				ID:      groupID,
				RoundID: roundID,
				Name:    displayName,
			}
			if err := r.createRoundGroup(ctx, tx, group); err != nil {
				return err
			}

			member := &RoundGroupParticipant{
				GroupID: groupID,
				RawName: displayName,
			}
			if p.UserID != "" {
				member.UserID = &p.UserID
			}

			if err := r.createRoundGroupParticipants(ctx, tx, []*RoundGroupParticipant{member}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Impl) createRoundGroup(ctx context.Context, tx bun.Tx, group *RoundGroup) error {
	_, err := tx.NewInsert().Model(group).Exec(ctx)
	return err
}

func (r *Impl) createRoundGroupParticipants(ctx context.Context, tx bun.Tx, members []*RoundGroupParticipant) error {
	if len(members) == 0 {
		return nil
	}
	_, err := tx.NewInsert().Model(&members).Exec(ctx)
	return err
}
