package userservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// -----------------------------------------------------------------------------
// Lifecycle & Helper Tests
// -----------------------------------------------------------------------------

func TestNewUserService(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			fakeRepo := NewFakeUserRepository()
			testHandler := loggerfrolfbot.NewTestHandler()
			logger := slog.New(testHandler)
			mockMetrics := &usermetrics.NoOpMetrics{}
			tracer := noop.NewTracerProvider().Tracer("test")
			var db *bun.DB

			service := NewUserService(fakeRepo, logger, mockMetrics, tracer, db)

			if service == nil {
				t.Fatal("NewUserService returned nil")
			}
			if service.repo != fakeRepo {
				t.Error("repo not set correctly")
			}
			if service.logger != logger {
				t.Error("logger not set correctly")
			}
		})
	}
}

func TestUserService_withTelemetry(t *testing.T) {
	s := &UserService{
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &usermetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	type SuccessPayload struct{ Data string }
	type FailurePayload struct{ Reason string }

	tests := []struct {
		name        string
		operation   string
		userID      sharedtypes.DiscordID
		op          operationFunc[SuccessPayload, FailurePayload]
		wantErrSub  string
		checkResult func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload])
	}{
		{
			name:      "handles success result",
			operation: "TestOp",
			userID:    "user-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.SuccessResult[SuccessPayload, FailurePayload](SuccessPayload{Data: "ok"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsSuccess() || res.Success.Data != "ok" {
					t.Errorf("expected success 'ok', got %+v", res.Success)
				}
			},
		},
		{
			name:      "handles infrastructure error",
			operation: "TestOp",
			userID:    "user-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.OperationResult[SuccessPayload, FailurePayload]{}, errors.New("db down")
			},
			wantErrSub: "TestOp: db down",
		},
		{
			name:      "recovers from panic",
			operation: "TestOp",
			userID:    "user-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				panic("boom")
			},
			wantErrSub: "panic in TestOp: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := withTelemetry(s, context.Background(), tt.operation, tt.userID, tt.op)

			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %v", tt.wantErrSub, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, res)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Business Logic Tests
// -----------------------------------------------------------------------------

func TestUserService_MatchParsedScorecard(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	userID := sharedtypes.DiscordID("admin-1")

	tests := []struct {
		name           string
		playerNames    []string
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository)
	}{
		{
			name:        "match by username success",
			playerNames: []string{"PlayerOne"},
			setupFake: func(f *FakeUserRepository) {
				f.GetUsersByUDiscUsernamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, usernames []string) ([]userdb.UserWithMembership, error) {
					for _, u := range usernames {
						if u == "playerone" {
							return []userdb.UserWithMembership{{User: &userdb.User{UserID: pointer(sharedtypes.DiscordID("discord-1")), UDiscUsername: pointer("playerone")}}}, nil
						}
					}
					return []userdb.UserWithMembership{}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure")
				}
				success := *res.Success
				if len(success.Mappings) != 1 {
					t.Fatalf("expected 1 mapping, got %d", len(success.Mappings))
				}
				if success.Mappings[0].DiscordUserID != "discord-1" {
					t.Errorf("expected discord-1, got %s", success.Mappings[0].DiscordUserID)
				}
			},
		},
		{
			name:        "match by name fallback success",
			playerNames: []string{"Real Name"},
			setupFake: func(f *FakeUserRepository) {
				f.GetUsersByUDiscUsernamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, usernames []string) ([]userdb.UserWithMembership, error) {
					return []userdb.UserWithMembership{}, nil
				}
				f.GetUsersByUDiscNamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, names []string) ([]userdb.UserWithMembership, error) {
					for _, n := range names {
						if n == "real name" {
							return []userdb.UserWithMembership{{User: &userdb.User{UserID: pointer(sharedtypes.DiscordID("discord-2")), UDiscName: pointer("real name")}}}, nil
						}
					}
					return []userdb.UserWithMembership{}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure")
				}
				success := *res.Success
				if len(success.Mappings) != 1 || success.Mappings[0].DiscordUserID != "discord-2" {
					t.Errorf("expected fallback match to discord-2, got %+v", success.Mappings)
				}
			},
		},
		{
			name:        "unmatched player",
			playerNames: []string{"Ghost"},
			setupFake: func(f *FakeUserRepository) {
				// Default behavior is ErrNotFound
			},
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure")
				}
				success := *res.Success
				if len(success.Unmatched) != 1 || success.Unmatched[0] != "Ghost" {
					t.Errorf("expected Ghost to be unmatched, got %v", success.Unmatched)
				}
			},
		},
		{
			name:        "match by first-name fuzzy fallback success",
			playerNames: []string{"Eric"},
			setupFake: func(f *FakeUserRepository) {
				f.GetUsersByUDiscUsernamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, usernames []string) ([]userdb.UserWithMembership, error) {
					return []userdb.UserWithMembership{}, nil
				}
				f.GetUsersByUDiscNamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, names []string) ([]userdb.UserWithMembership, error) {
					return []userdb.UserWithMembership{}, nil
				}
				f.FindByUDiscNameFuzzyFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, partialName string) ([]*userdb.UserWithMembership, error) {
					if partialName != "eric" {
						return []*userdb.UserWithMembership{}, nil
					}
					return []*userdb.UserWithMembership{
						{
							User: &userdb.User{
								UserID:    pointer(sharedtypes.DiscordID("discord-3")),
								UDiscName: pointer("Eric Goodyke"),
							},
						},
					}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure")
				}
				success := *res.Success
				if len(success.Mappings) != 1 || success.Mappings[0].DiscordUserID != "discord-3" {
					t.Errorf("expected fuzzy first-name match to discord-3, got %+v", success.Mappings)
				}
			},
		},
		{
			name:        "first-name fuzzy fallback ambiguous remains unmatched",
			playerNames: []string{"Mark"},
			setupFake: func(f *FakeUserRepository) {
				f.GetUsersByUDiscUsernamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, usernames []string) ([]userdb.UserWithMembership, error) {
					return []userdb.UserWithMembership{}, nil
				}
				f.GetUsersByUDiscNamesFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, names []string) ([]userdb.UserWithMembership, error) {
					return []userdb.UserWithMembership{}, nil
				}
				f.FindByUDiscNameFuzzyFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, partialName string) ([]*userdb.UserWithMembership, error) {
					if partialName != "mark" {
						return []*userdb.UserWithMembership{}, nil
					}
					return []*userdb.UserWithMembership{
						{
							User: &userdb.User{
								UserID:    pointer(sharedtypes.DiscordID("discord-4")),
								UDiscName: pointer("Mark Newton"),
							},
						},
						{
							User: &userdb.User{
								UserID:    pointer(sharedtypes.DiscordID("discord-5")),
								UDiscName: pointer("Mark Jones"),
							},
						},
					}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure")
				}
				success := *res.Success
				if len(success.Mappings) != 0 {
					t.Errorf("expected no mapping due to ambiguity, got %+v", success.Mappings)
				}
				if len(success.Unmatched) != 1 || success.Unmatched[0] != "Mark" {
					t.Errorf("expected Mark to remain unmatched, got %v", success.Unmatched)
				}
			},
		},
		{
			name:           "too many players error",
			playerNames:    make([]string, 513),
			expectInfraErr: false,
			verify: func(t *testing.T, res results.OperationResult[*MatchResult, error], fake *FakeUserRepository) {
				if !res.IsFailure() || !strings.Contains((*res.Failure).Error(), "too many players") {
					t.Errorf("expected too many players failure, got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeUserRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := NewUserService(fakeRepo, loggerfrolfbot.NoOpLogger, &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)
			res, err := s.MatchParsedScorecard(ctx, guildID, userID, tt.playerNames)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, res, fakeRepo)
			}
		})
	}
}

func TestUserService_UpdateUDiscIdentity(t *testing.T) {
	ctx := context.Background()
	userID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name           string
		username       *string
		nameVal        *string
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[bool, error], err error, fake *FakeUserRepository)
	}{
		{
			name:     "successful global update",
			username: pointer("NewUser"),
			setupFake: func(f *FakeUserRepository) {
				f.UpdateGlobalUserFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, updates *userdb.UserUpdateFields) error {
					if *updates.UDiscUsername != "newuser" {
						return errors.New("not normalized")
					}
					return nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], err error, fake *FakeUserRepository) {
				if err != nil || !res.IsSuccess() {
					t.Errorf("expected success, got err: %v, res: %v", err, res)
				}
				if fake.Trace()[0] != "UpdateGlobalUser" {
					t.Errorf("expected UpdateGlobalUser trace")
				}
			},
		},
		{
			name:     "user not found",
			username: pointer("Nobody"),
			setupFake: func(f *FakeUserRepository) {
				f.UpdateGlobalUserFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, updates *userdb.UserUpdateFields) error {
					return userdb.ErrNoRowsAffected
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], err error, fake *FakeUserRepository) {
				if !res.IsFailure() || !errors.Is(*res.Failure, userdb.ErrNotFound) {
					t.Errorf("expected domain failure ErrNotFound, got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeUserRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := NewUserService(fakeRepo, loggerfrolfbot.NoOpLogger, &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)
			res, err := s.UpdateUDiscIdentity(ctx, userID, tt.username, tt.nameVal)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error")
			}
			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}

func TestUserService_UpdateUserProfile(t *testing.T) {
	ctx := context.Background()
	userID := sharedtypes.DiscordID("user-1")

	tests := []struct {
		name        string
		displayName string
		avatarHash  string
		setupFake   func(*FakeUserRepository)
		verify      func(t *testing.T, err error, fake *FakeUserRepository)
	}{
		{
			name:        "success",
			displayName: "New Name",
			avatarHash:  "hash123",
			setupFake: func(f *FakeUserRepository) {
				f.UpdateProfileFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, name string, hash string) error {
					return nil
				}
			},
			verify: func(t *testing.T, err error, fake *FakeUserRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if fake.Trace()[0] != "UpdateProfile" {
					t.Errorf("expected UpdateProfile trace")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeUserRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}
			s := NewUserService(fakeRepo, loggerfrolfbot.NoOpLogger, &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)
			err := s.UpdateUserProfile(ctx, userID, "", tt.displayName, tt.avatarHash)
			tt.verify(t, err, fakeRepo)
		})
	}
}

func TestUserService_LookupProfiles(t *testing.T) {
	ctx := context.Background()
	userIDs := []sharedtypes.DiscordID{"user-1", "user-2"}

	tests := []struct {
		name      string
		ids       []sharedtypes.DiscordID
		setupFake func(*FakeUserRepository)
		verify    func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error)
	}{
		{
			name: "success with partial match",
			ids:  userIDs,
			setupFake: func(f *FakeUserRepository) {
				f.GetByUserIDsFunc = func(ctx context.Context, db bun.IDB, ids []sharedtypes.DiscordID) ([]*userdb.User, error) {
					return []*userdb.User{
						{UserID: pointer(sharedtypes.DiscordID("user-1")), DisplayName: pointer("UserOne")},
					}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success")
				}
				resp := *res.Success
				profiles := resp.Profiles
				if len(profiles) != 2 {
					t.Errorf("expected 2 profiles, got %d", len(profiles))
				}
				if p1, ok := profiles["user-1"]; !ok || p1.DisplayName != "UserOne" {
					t.Errorf("expected UserOne for user-1")
				}
				if p2, ok := profiles["user-2"]; !ok || !strings.HasPrefix(p2.DisplayName, "User") {
					t.Errorf("expected default profile for user-2")
				}
			},
		},
		{
			name: "non-discord numeric ids do not enqueue sync requests",
			ids:  []sharedtypes.DiscordID{"5", "23"},
			verify: func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success")
				}
				resp := *res.Success
				if got := len(resp.SyncRequests); got != 0 {
					t.Fatalf("expected no sync requests, got %d", got)
				}
			},
		},
		{
			name: "club membership external id is used for manual profiles",
			ids:  []sharedtypes.DiscordID{"23"},
			setupFake: func(f *FakeUserRepository) {
				clubUUID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
				displayName := "muffinmaster123"
				f.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				f.GetClubMembershipsByExternalIDsFn = func(ctx context.Context, db bun.IDB, externalIDs []string, gotClubUUID uuid.UUID) ([]*userdb.ClubMembership, error) {
					if gotClubUUID != clubUUID {
						t.Fatalf("unexpected club UUID %s", gotClubUUID)
					}
					var out []*userdb.ClubMembership
					for _, id := range externalIDs {
						if id == "23" {
							out = append(out, &userdb.ClubMembership{
								ClubUUID:    clubUUID,
								ExternalID:  pointer("23"),
								DisplayName: &displayName,
							})
						}
					}
					return out, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success")
				}
				resp := *res.Success
				profile := resp.Profiles["23"]
				if profile == nil {
					t.Fatalf("expected profile for external id 23")
				}
				if profile.DisplayName != "muffinmaster123" {
					t.Fatalf("expected muffinmaster123, got %q", profile.DisplayName)
				}
				if got := len(resp.SyncRequests); got != 0 {
					t.Fatalf("expected no sync requests for manual external profile, got %d", got)
				}
			},
		},
		{
			name: "club membership external id overrides placeholder user profile",
			ids:  []sharedtypes.DiscordID{"23"},
			setupFake: func(f *FakeUserRepository) {
				clubUUID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
				placeholderDisplayName := "Tag 23 Placeholder"
				manualDisplayName := "muffinmaster123"
				f.GetByUserIDsFunc = func(ctx context.Context, db bun.IDB, ids []sharedtypes.DiscordID) ([]*userdb.User, error) {
					return []*userdb.User{
						{
							UserID:      pointer(sharedtypes.DiscordID("23")),
							DisplayName: &placeholderDisplayName,
						},
					}, nil
				}
				f.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				f.GetClubMembershipsByExternalIDsFn = func(ctx context.Context, db bun.IDB, externalIDs []string, gotClubUUID uuid.UUID) ([]*userdb.ClubMembership, error) {
					if gotClubUUID != clubUUID {
						t.Fatalf("unexpected club UUID %s", gotClubUUID)
					}
					var out []*userdb.ClubMembership
					for _, id := range externalIDs {
						if id == "23" {
							out = append(out, &userdb.ClubMembership{
								ClubUUID:    clubUUID,
								ExternalID:  pointer("23"),
								DisplayName: &manualDisplayName,
							})
						}
					}
					return out, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success")
				}
				resp := *res.Success
				profile := resp.Profiles["23"]
				if profile == nil {
					t.Fatalf("expected profile for external id 23")
				}
				if profile.DisplayName != "muffinmaster123" {
					t.Fatalf("expected muffinmaster123, got %q", profile.DisplayName)
				}
			},
		},
		{
			name: "batch lookup is called once for multiple users",
			ids:  []sharedtypes.DiscordID{"5", "23", "42"},
			setupFake: func(f *FakeUserRepository) {
				clubUUID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
				callCount := 0
				f.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				f.GetClubMembershipsByExternalIDsFn = func(ctx context.Context, db bun.IDB, externalIDs []string, gotClubUUID uuid.UUID) ([]*userdb.ClubMembership, error) {
					callCount++
					if callCount > 1 {
						t.Errorf("GetClubMembershipsByExternalIDs called %d times, expected 1", callCount)
					}
					if len(externalIDs) != 3 {
						t.Errorf("expected 3 externalIDs, got %d", len(externalIDs))
					}
					return nil, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*LookupProfilesResponse, error], err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeUserRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}
			s := NewUserService(fakeRepo, loggerfrolfbot.NoOpLogger, &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)
			res, err := s.LookupProfiles(ctx, tt.ids, "guild-1")
			tt.verify(t, res, err)
		})
	}
}

func TestIsDiscordSnowflake(t *testing.T) {
	tests := []struct {
		userID sharedtypes.DiscordID
		want   bool
	}{
		{"5", false},
		{"23", false},
		{"123456789012345", false},      // 15 digits — too short
		{"1234567890123456", false},     // 16 digits — too short
		{"12345678901234567", true},     // 17 digits
		{"123456789012345678", true},    // 18 digits
		{"1234567890123456789", true},   // 19 digits
		{"12345678901234567890", false}, // 20 digits — too long
		{"not-a-number", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.userID), func(t *testing.T) {
			got := isDiscordSnowflake(tt.userID)
			if got != tt.want {
				t.Errorf("isDiscordSnowflake(%q) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}

func TestGenerateDefaultDisplayName_RoundTrip(t *testing.T) {
	ids := []sharedtypes.DiscordID{
		"5",
		"23",
		"123456",
		"1234567",
		"123456789012345678",
	}

	for _, id := range ids {
		t.Run(string(id), func(t *testing.T) {
			generated := generateDefaultDisplayName(id)
			if !isSyntheticLookupDisplayName(id, generated) {
				t.Errorf("isSyntheticLookupDisplayName(%q, %q) = false, want true", id, generated)
			}
			if isSyntheticLookupDisplayName(id, "Real Name") {
				t.Errorf("isSyntheticLookupDisplayName(%q, \"Real Name\") = true, want false", id)
			}
		})
	}
}
