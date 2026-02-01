package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_UpdateScheduledRoundsWithNewTags(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	// Test data
	user1ID := sharedtypes.DiscordID("user1")
	user2ID := sharedtypes.DiscordID("user2")
	user3ID := sharedtypes.DiscordID("user3")

	round1ID := sharedtypes.RoundID(uuid.New())
	round2ID := sharedtypes.RoundID(uuid.New())

	// Define tag numbers
	tag1 := sharedtypes.TagNumber(42)
	tag2 := sharedtypes.TagNumber(17)
	tag3 := sharedtypes.TagNumber(99)

	newTag1 := sharedtypes.TagNumber(23)
	newTag2 := sharedtypes.TagNumber(31)

	// Create participants with existing tags
	participant1 := roundtypes.Participant{
		UserID:    user1ID,
		TagNumber: &tag1,
		Response:  roundtypes.ResponseAccept,
	}
	participant2 := roundtypes.Participant{
		UserID:    user2ID,
		TagNumber: &tag2,
		Response:  roundtypes.ResponseAccept,
	}
	participant3 := roundtypes.Participant{
		UserID:    user3ID,
		TagNumber: &tag3,
		Response:  roundtypes.ResponseAccept,
	}

	// Setup test rounds with participants
	round1 := roundtypes.Round{
		ID:             round1ID,
		EventMessageID: "msg1", // Add EventMessageID
		Participants:   []roundtypes.Participant{participant1, participant2},
	}

	round2 := roundtypes.Round{
		ID:             round2ID,
		EventMessageID: "msg2", // Add EventMessageID
		Participants:   []roundtypes.Participant{participant2, participant3},
	}

	// Create slice of pointers to rounds
	upcomingRounds := []*roundtypes.Round{&round1, &round2}

	tests := []struct {
		name        string
		setup       func(*FakeRepo)
		req         *roundtypes.UpdateScheduledRoundsWithNewTagsRequest
		expectError bool
		verify      func(t *testing.T, res UpdateScheduledRoundsWithNewTagsResult, err error, fake *FakeRepo)
	}{
		{
			name: "successful update with valid tags",
			setup: func(f *FakeRepo) {
				f.GetUpcomingRoundsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) ([]*roundtypes.Round, error) {
					return upcomingRounds, nil
				}
				f.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					return nil
				}
			},
			req: &roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					user1ID: newTag1,
					user2ID: newTag2,
				},
			},
			verify: func(t *testing.T, res UpdateScheduledRoundsWithNewTagsResult, err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				payload := *res.Success
				// Should have 2 rounds (both rounds have participants that need updates)
				if len(payload.Updates) != 2 {
					t.Errorf("expected 2 updated rounds, got %d", len(payload.Updates))
				}
				// Total participants updated should be 3 (user1 once, user2 twice across 2 rounds)
				count := 0
				for _, u := range payload.Updates {
					for _, p := range u.Participants {
						if p.UserID == user1ID && p.TagNumber != nil && *p.TagNumber == newTag1 {
							count++
						}
						if p.UserID == user2ID && p.TagNumber != nil && *p.TagNumber == newTag2 {
							count++
						}
					}
				}
				if count != 3 {
					t.Errorf("expected 3 participant updates, got %d", count)
				}
			},
		},
		{
			name: "error fetching rounds",
			setup: func(f *FakeRepo) {
				f.GetUpcomingRoundsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) ([]*roundtypes.Round, error) {
					return nil, errors.New("database error")
				}
			},
			req: &roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					user1ID: newTag1,
				},
			},
			verify: func(t *testing.T, res UpdateScheduledRoundsWithNewTagsResult, err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "failed to get upcoming rounds: database error" {
					t.Errorf("expected error 'failed to get upcoming rounds: database error', got %v", res.Failure)
				}
			},
		},
		{
			name: "error updating rounds",
			setup: func(f *FakeRepo) {
				f.GetUpcomingRoundsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) ([]*roundtypes.Round, error) {
					return upcomingRounds, nil
				}
				f.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					return errors.New("update failed")
				}
			},
			req: &roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
				GuildID: sharedtypes.GuildID("guild-123"),
				ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					user1ID: newTag1,
				},
			},
			verify: func(t *testing.T, res UpdateScheduledRoundsWithNewTagsResult, err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil || (*res.Failure).Error() != "database update failed: update failed" {
					t.Errorf("expected error 'database update failed: update failed', got %v", res.Failure)
				}
			},
		},
		{
			name: "no updates needed - empty changedTags",
			req: &roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
				GuildID:     sharedtypes.GuildID("guild-123"),
				ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{},
			},
			verify: func(t *testing.T, res UpdateScheduledRoundsWithNewTagsResult, err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				payload := *res.Success
				if len(payload.Updates) != 0 {
					t.Errorf("expected 0 updates, got %d", len(payload.Updates))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeRepo()
			if tt.setup != nil {
				tt.setup(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: &FakeRoundValidator{},
				eventBus:       &FakeEventBus{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.UpdateScheduledRoundsWithNewTags(ctx, tt.req)

			if tt.expectError && err == nil {
				t.Errorf("expected an error, but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("expected no error, but got: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, result, err, fakeRepo)
			}
		})
	}
}
