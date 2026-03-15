package clubservice

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	discordAPIBase  = "https://discord.com/api/v10"
	inviteCodeChars = "abcdefghjkmnpqrstuvwxyz23456789" // unambiguous alphanumeric
	inviteCodeLen   = 8
)

// discordGuild is the shape returned by /users/@me/guilds.
type discordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
}

// hasManageGuild reports whether the guild has MANAGE_GUILD (0x20) or ADMINISTRATOR (0x8).
func (g *discordGuild) hasManageGuild() bool {
	var perms int64
	fmt.Sscanf(g.Permissions, "%d", &perms)
	return (perms&0x20) != 0 || (perms&0x8) != 0
}

// ClubService implements the Service interface.
type ClubService struct {
	repo              clubdb.Repository
	guildRepo         guilddb.Repository
	userRepo          userdb.Repository
	queueService      ChallengeQueueService
	leaderboardReader ChallengeTagReader
	roundReader       ChallengeRoundReader
	logger            *slog.Logger
	metrics           clubmetrics.ClubMetrics
	tracer            trace.Tracer
	db                *bun.DB
}

// NewClubService creates a new ClubService.
func NewClubService(
	repo clubdb.Repository,
	guildRepo guilddb.Repository,
	userRepo userdb.Repository,
	queueService ChallengeQueueService,
	leaderboardReader ChallengeTagReader,
	roundReader ChallengeRoundReader,
	logger *slog.Logger,
	metrics clubmetrics.ClubMetrics,
	tracer trace.Tracer,
	db *bun.DB,
) *ClubService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ClubService{
		repo:              repo,
		guildRepo:         guildRepo,
		userRepo:          userRepo,
		queueService:      queueService,
		leaderboardReader: leaderboardReader,
		roundReader:       roundReader,
		logger:            logger,
		metrics:           metrics,
		tracer:            tracer,
		db:                db,
	}
}

// GetClub retrieves club info by UUID.
func (s *ClubService) GetClub(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
	getClubTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return s.getClubLogic(ctx, db, clubUUID)
	}

	result, err := withTelemetry(s, ctx, "GetClub", clubUUID.String(), func(ctx context.Context) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return runInTx(s, ctx, getClubTx)
	})
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}
	return *result.Success, nil
}

// getClubLogic contains the core logic.
func (s *ClubService) getClubLogic(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
	club, err := s.repo.GetByUUID(ctx, db, clubUUID)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return results.FailureResult[*clubtypes.ClubInfo, error](err), nil
		}
		return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to get club: %w", err)
	}

	return results.SuccessResult[*clubtypes.ClubInfo, error](&clubtypes.ClubInfo{
		UUID:         club.UUID.String(),
		Name:         club.Name,
		IconURL:      club.IconURL,
		Entitlements: s.resolveEntitlements(ctx, db, club.UUID),
	}), nil
}

func (s *ClubService) resolveEntitlements(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) guildtypes.ResolvedClubEntitlements {
	if s.guildRepo == nil || clubUUID == uuid.Nil {
		return guildtypes.ResolvedClubEntitlements{}
	}

	entitlements, err := s.guildRepo.ResolveEntitlements(ctx, db, sharedtypes.GuildID(clubUUID.String()))
	if err != nil {
		if errors.Is(err, guilddb.ErrNotFound) {
			return guildtypes.ResolvedClubEntitlements{}
		}
		s.logger.WarnContext(ctx, "Failed to resolve club entitlements",
			attr.String("club_uuid", clubUUID.String()),
			attr.Error(err),
		)
		return guildtypes.ResolvedClubEntitlements{}
	}

	return entitlements
}

// UpsertClubFromDiscord creates or updates a club from Discord guild info.
func (s *ClubService) UpsertClubFromDiscord(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error) {
	upsertTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return s.upsertClubFromDiscordLogic(ctx, db, guildID, name, iconURL)
	}

	result, err := withTelemetry(s, ctx, "UpsertClubFromDiscord", guildID, func(ctx context.Context) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return runInTx(s, ctx, upsertTx)
	})
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}
	return *result.Success, nil
}

// upsertClubFromDiscordLogic contains the core logic.
func (s *ClubService) upsertClubFromDiscordLogic(ctx context.Context, db bun.IDB, guildID, name string, iconURL *string) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
	existing, err := s.repo.GetByDiscordGuildID(ctx, db, guildID)
	if err != nil && !errors.Is(err, clubdb.ErrNotFound) {
		return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to check existing club: %w", err)
	}

	var club *clubdb.Club
	if existing != nil {
		existing.Name = name
		existing.IconURL = iconURL
		if err := s.repo.Upsert(ctx, db, existing); err != nil {
			return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to update club: %w", err)
		}
		club = existing
	} else {
		club = &clubdb.Club{
			UUID:           uuid.New(),
			Name:           name,
			IconURL:        iconURL,
			DiscordGuildID: &guildID,
		}
		if err := s.repo.Upsert(ctx, db, club); err != nil {
			// Retry once in case of race condition on discord_guild_id
			existing, retryErr := s.repo.GetByDiscordGuildID(ctx, db, guildID)
			if retryErr == nil && existing != nil {
				existing.Name = name
				existing.IconURL = iconURL
				if updateErr := s.repo.Upsert(ctx, db, existing); updateErr != nil {
					return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to update club on retry: %w", updateErr)
				}
				club = existing
			} else {
				return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to create club: %w", err)
			}
		}
	}

	return results.SuccessResult[*clubtypes.ClubInfo, error](&clubtypes.ClubInfo{
		UUID:    club.UUID.String(),
		Name:    club.Name,
		IconURL: club.IconURL,
	}), nil
}

// -----------------------------------------------------------------------------
// Club Discovery & Join (Phase 4)
// -----------------------------------------------------------------------------

// GetClubSuggestions returns clubs that match the user's Discord guilds and
// that the user is not already a member of.
func (s *ClubService) GetClubSuggestions(ctx context.Context, userUUID uuid.UUID) ([]ClubSuggestion, error) {
	ctx, span := s.tracer.Start(ctx, "ClubService.GetClubSuggestions")
	defer span.End()

	// 1. Get the user's Discord linked identity (needs stored access token).
	identity, err := s.userRepo.GetLinkedIdentityByProvider(ctx, nil, userUUID, "discord")
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return []ClubSuggestion{}, nil // No Discord linked — no suggestions
		}
		return nil, fmt.Errorf("failed to get Discord identity: %w", err)
	}
	if identity.AccessToken == nil || *identity.AccessToken == "" {
		return []ClubSuggestion{}, nil // No stored token — can't call Discord API
	}

	// 2. Fetch the user's Discord guilds using the stored access token.
	guilds, err := fetchDiscordGuilds(ctx, *identity.AccessToken)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to fetch Discord guilds", attr.Error(err))
		return []ClubSuggestion{}, nil
	}
	if len(guilds) == 0 {
		return []ClubSuggestion{}, nil
	}

	guildIDs := make([]string, len(guilds))
	for i, g := range guilds {
		guildIDs[i] = g.ID
	}

	// 3. Match guild IDs against clubs in the DB.
	clubs, err := s.repo.GetClubsByDiscordGuildIDs(ctx, nil, guildIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get clubs: %w", err)
	}
	if len(clubs) == 0 {
		return []ClubSuggestion{}, nil
	}

	// 4. Exclude clubs the user is already a member of.
	memberships, err := s.userRepo.GetClubMembershipsByUserUUID(ctx, nil, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memberships: %w", err)
	}
	memberSet := make(map[uuid.UUID]struct{}, len(memberships))
	for _, m := range memberships {
		memberSet[m.ClubUUID] = struct{}{}
	}

	suggestions := make([]ClubSuggestion, 0, len(clubs))
	for _, c := range clubs {
		if _, alreadyMember := memberSet[c.UUID]; alreadyMember {
			continue
		}
		suggestions = append(suggestions, ClubSuggestion{
			UUID:    c.UUID.String(),
			Name:    c.Name,
			IconURL: c.IconURL,
		})
	}
	return suggestions, nil
}

// JoinClub adds the user as a member of the specified club. Verifies Discord
// guild membership via the stored access token and assigns the appropriate role.
func (s *ClubService) JoinClub(ctx context.Context, userUUID, clubUUID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "ClubService.JoinClub")
	defer span.End()

	// 1. Verify the club exists and has a Discord guild ID.
	club, err := s.repo.GetByUUID(ctx, nil, clubUUID)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return fmt.Errorf("club not found")
		}
		return fmt.Errorf("failed to get club: %w", err)
	}
	if club.DiscordGuildID == nil {
		return fmt.Errorf("club has no associated Discord server")
	}

	// 2. Get user's Discord access token.
	identity, err := s.userRepo.GetLinkedIdentityByProvider(ctx, nil, userUUID, "discord")
	if err != nil {
		return fmt.Errorf("Discord account not linked")
	}
	if identity.AccessToken == nil || *identity.AccessToken == "" {
		return fmt.Errorf("Discord access token not available; please sign in again")
	}

	// 3. Verify user is a member of the Discord guild and determine role.
	guilds, err := fetchDiscordGuilds(ctx, *identity.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to verify Discord membership: %w", err)
	}

	var role sharedtypes.UserRoleEnum
	found := false
	for _, g := range guilds {
		if g.ID == *club.DiscordGuildID {
			found = true
			if g.hasManageGuild() {
				role = sharedtypes.UserRoleAdmin
			} else {
				role = sharedtypes.UserRoleUser
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("you are not a member of the Discord server for this club")
	}

	// 4. Upsert club membership.
	membership := &userdb.ClubMembership{
		UserUUID: userUUID,
		ClubUUID: clubUUID,
		Role:     role,
		Source:   "discord",
	}
	if err := s.userRepo.UpsertClubMembership(ctx, nil, membership); err != nil {
		return fmt.Errorf("failed to create membership: %w", err)
	}

	s.logger.InfoContext(ctx, "User joined club via Discord",
		attr.String("user_uuid", userUUID.String()),
		attr.String("club_uuid", clubUUID.String()),
		attr.String("role", string(role)),
	)
	return nil
}

// -----------------------------------------------------------------------------
// Invite Codes (Phase 5)
// -----------------------------------------------------------------------------

// CreateInvite generates an invite code for a club. The caller must be admin or editor.
func (s *ClubService) CreateInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, req CreateInviteRequest) (*InviteInfo, error) {
	ctx, span := s.tracer.Start(ctx, "ClubService.CreateInvite")
	defer span.End()

	if err := s.requireAdminOrEditor(ctx, callerUUID, clubUUID); err != nil {
		return nil, err
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	role := req.Role
	if role == "" {
		role = "player"
	}

	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		t := time.Now().UTC().Add(time.Duration(*req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	invite := &clubdb.ClubInvite{
		ClubUUID:  clubUUID,
		CreatedBy: callerUUID,
		Code:      code,
		Role:      role,
		MaxUses:   req.MaxUses,
		ExpiresAt: expiresAt,
	}
	if err := s.repo.CreateInvite(ctx, nil, invite); err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	return &InviteInfo{
		Code:      invite.Code,
		ClubUUID:  clubUUID.String(),
		Role:      invite.Role,
		MaxUses:   invite.MaxUses,
		UseCount:  0,
		ExpiresAt: invite.ExpiresAt,
		CreatedAt: invite.CreatedAt,
	}, nil
}

// ListInvites returns active invite codes for a club. Requires admin or editor role.
func (s *ClubService) ListInvites(ctx context.Context, callerUUID, clubUUID uuid.UUID) ([]*InviteInfo, error) {
	ctx, span := s.tracer.Start(ctx, "ClubService.ListInvites")
	defer span.End()

	if err := s.requireAdminOrEditor(ctx, callerUUID, clubUUID); err != nil {
		return nil, err
	}

	invites, err := s.repo.GetInvitesByClub(ctx, nil, clubUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invites: %w", err)
	}

	result := make([]*InviteInfo, len(invites))
	for i, inv := range invites {
		result[i] = &InviteInfo{
			Code:      inv.Code,
			ClubUUID:  inv.ClubUUID.String(),
			Role:      inv.Role,
			MaxUses:   inv.MaxUses,
			UseCount:  inv.UseCount,
			ExpiresAt: inv.ExpiresAt,
			CreatedAt: inv.CreatedAt,
		}
	}
	return result, nil
}

// RevokeInvite revokes an invite code. Requires admin or editor role.
func (s *ClubService) RevokeInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, code string) error {
	ctx, span := s.tracer.Start(ctx, "ClubService.RevokeInvite")
	defer span.End()

	if err := s.requireAdminOrEditor(ctx, callerUUID, clubUUID); err != nil {
		return err
	}

	if err := s.repo.RevokeInvite(ctx, nil, clubUUID, code); err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return fmt.Errorf("invite not found")
		}
		return fmt.Errorf("failed to revoke invite: %w", err)
	}
	return nil
}

// GetInvitePreview validates an invite code and returns club preview info (public).
func (s *ClubService) GetInvitePreview(ctx context.Context, code string) (*InvitePreview, error) {
	ctx, span := s.tracer.Start(ctx, "ClubService.GetInvitePreview")
	defer span.End()

	invite, err := s.repo.GetInviteByCode(ctx, nil, code)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return nil, fmt.Errorf("invite not found")
		}
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}
	if invite.Revoked {
		return nil, fmt.Errorf("invite has been revoked")
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, fmt.Errorf("invite has expired")
	}
	if invite.MaxUses != nil && invite.UseCount >= *invite.MaxUses {
		return nil, fmt.Errorf("invite has reached its maximum uses")
	}

	club, err := s.repo.GetByUUID(ctx, nil, invite.ClubUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get club: %w", err)
	}

	return &InvitePreview{
		ClubUUID: club.UUID.String(),
		ClubName: club.Name,
		IconURL:  club.IconURL,
		Role:     invite.Role,
	}, nil
}

// JoinByCode uses an invite code to join a club.
func (s *ClubService) JoinByCode(ctx context.Context, userUUID uuid.UUID, code string) error {
	ctx, span := s.tracer.Start(ctx, "ClubService.JoinByCode")
	defer span.End()

	invite, err := s.repo.GetInviteByCode(ctx, nil, code)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return fmt.Errorf("invalid invite code")
		}
		return fmt.Errorf("failed to get invite: %w", err)
	}
	if invite.Revoked {
		return fmt.Errorf("invite has been revoked")
	}
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return fmt.Errorf("invite has expired")
	}
	if invite.MaxUses != nil && invite.UseCount >= *invite.MaxUses {
		return fmt.Errorf("invite has reached its maximum uses")
	}

	membership := &userdb.ClubMembership{
		UserUUID: userUUID,
		ClubUUID: invite.ClubUUID,
		Role:     sharedtypes.UserRoleEnum(invite.Role),
		Source:   "invite",
	}
	if err := s.userRepo.UpsertClubMembership(ctx, nil, membership); err != nil {
		return fmt.Errorf("failed to create membership: %w", err)
	}

	if err := s.repo.IncrementInviteUseCount(ctx, nil, code); err != nil {
		// Non-fatal: log and continue
		s.logger.WarnContext(ctx, "Failed to increment invite use count", attr.Error(err))
	}

	s.logger.InfoContext(ctx, "User joined club via invite code",
		attr.String("user_uuid", userUUID.String()),
		attr.String("club_uuid", invite.ClubUUID.String()),
		attr.String("role", invite.Role),
	)
	return nil
}

// -----------------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------------

func (s *ClubService) requireAdminOrEditor(ctx context.Context, userUUID, clubUUID uuid.UUID) error {
	membership, err := s.userRepo.GetClubMembership(ctx, nil, userUUID, clubUUID)
	if err != nil {
		if errors.Is(err, userdb.ErrNotFound) {
			return fmt.Errorf("forbidden: not a member of this club")
		}
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if membership.Role != sharedtypes.UserRoleAdmin && membership.Role != sharedtypes.UserRoleEditor {
		return fmt.Errorf("forbidden: requires admin or editor role")
	}
	return nil
}

func fetchDiscordGuilds(ctx context.Context, accessToken string) ([]discordGuild, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me/guilds", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var guilds []discordGuild
	if err := json.Unmarshal(body, &guilds); err != nil {
		return nil, fmt.Errorf("failed to parse guilds: %w", err)
	}
	return guilds, nil
}

func generateInviteCode() (string, error) {
	b := make([]byte, inviteCodeLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			return "", err
		}
		b[i] = inviteCodeChars[n.Int64()]
	}
	return string(b), nil
}

// -----------------------------------------------------------------------------
// Generic Helpers (Defined as functions because methods cannot have type params)
// -----------------------------------------------------------------------------

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// valueOperationFunc is the generic signature for value-returning service operations.
type valueOperationFunc[T any] func(ctx context.Context) (T, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func withTelemetry[S any, F any](
	s *ClubService,
	ctx context.Context,
	operationName string,
	identifier string,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {

	// Start span
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("identifier", identifier),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	// Record attempt
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, "ClubService")
	}

	// Track duration
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, "ClubService", time.Since(startTime))
		}
	}()

	// Log operation start
	s.logger.InfoContext(ctx, "Operation triggered", attr.ExtractCorrelationID(ctx), attr.String("operation", operationName))

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("identifier", identifier),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
			}
			span.RecordError(err)
			result = results.OperationResult[S, F]{}
		}
	}()

	// Execute operation
	result, err = op(ctx)

	// Handle Infrastructure Error
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Handle Domain Failure
	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
			attr.Any("failure_payload", *result.Failure),
		)
	}

	// Handle Success
	if result.IsSuccess() {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
		)
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, "ClubService")
	}

	return result, nil
}

// withValueTelemetry wraps a value-returning service operation with tracing,
// metrics, structured logging, and panic recovery.
func withValueTelemetry[T any](
	s *ClubService,
	ctx context.Context,
	operationName string,
	identifier string,
	op valueOperationFunc[T],
) (value T, err error) {
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("identifier", identifier),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, "ClubService")
	}

	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, "ClubService", time.Since(startTime))
		}
	}()

	s.logger.InfoContext(ctx, "Operation triggered", attr.ExtractCorrelationID(ctx), attr.String("operation", operationName))

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("identifier", identifier),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
			}
			span.RecordError(err)
			var zero T
			value = zero
		}
	}()

	value, err = op(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
		}
		span.RecordError(wrappedErr)
		var zero T
		return zero, wrappedErr
	}

	s.logger.InfoContext(ctx, "Operation completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("identifier", identifier),
	)

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, "ClubService")
	}

	return value, nil
}

// runInTx ensures the operation runs within a transaction.
func runInTx[S any, F any](
	s *ClubService,
	ctx context.Context,
	fn func(ctx context.Context, db bun.IDB) (results.OperationResult[S, F], error),
) (results.OperationResult[S, F], error) {

	if s.db == nil {
		return fn(ctx, nil)
	}

	var result results.OperationResult[S, F]

	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var txErr error
		result, txErr = fn(ctx, tx)
		return txErr
	})

	return result, err
}
