package clubservice

import (
	"context"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/google/uuid"
)

// Service defines the club service contract.
type Service interface {
	// GetClub retrieves club info by UUID.
	GetClub(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error)

	// UpsertClubFromDiscord creates or updates a club from Discord guild info.
	UpsertClubFromDiscord(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error)
}
