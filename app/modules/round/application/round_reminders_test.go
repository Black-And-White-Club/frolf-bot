package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	testReminderRoundID    = sharedtypes.RoundID(uuid.New())
	testReminderRoundTitle = "Test Round"
	testReminderLocation   = "Test Location"
	testReminderStartTime  = time.Now()
	testReminderType       = "Test Reminder Type"
)

var (
	testParticipant1 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user1"),
		TagNumber: nil,
		Response:  roundtypes.ResponseAccept,
	}
	testParticipant2 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user2"),
		TagNumber: nil,
		Response:  roundtypes.ResponseTentative,
	}
	testParticipant3 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user3"),
		TagNumber: nil,
		Response:  roundtypes.ResponseDecline,
	}
)

func TestRoundService_ProcessRoundReminder(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	testDiscordMessageID := "12345"

	tests := []struct {
		name           string
		setup          func(*FakeRepo, *FakeGuildConfigProvider)
		req            *roundtypes.ProcessRoundReminderRequest
		expectedResult results.OperationResult[roundtypes.ProcessRoundReminderResult, error]
		expectedError  error
		verify         func(*testing.T, results.OperationResult[roundtypes.ProcessRoundReminderResult, error])
	}{
		{
			name: "successful processing with participants",
			setup: func(f *FakeRepo, gcp *FakeGuildConfigProvider) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{testParticipant1, testParticipant2}, nil
				}
			},
			req: &roundtypes.ProcessRoundReminderRequest{
				RoundID:          testReminderRoundID,
				GuildID:          sharedtypes.GuildID("guild-123"),
				RoundTitle:       testReminderRoundTitle,
				StartTime:        testReminderStartTime.String(),
				Location:         testReminderLocation,
				ReminderType:     testReminderType,
				EventMessageID:   testDiscordMessageID,
				DiscordChannelID: "channel-123",
			},
			expectedResult: results.OperationResult[roundtypes.ProcessRoundReminderResult, error]{
				Success: &roundtypes.ProcessRoundReminderResult{
					RoundID:          testReminderRoundID,
					GuildID:          sharedtypes.GuildID("guild-123"),
					RoundTitle:       testReminderRoundTitle,
					StartTime:        testReminderStartTime.String(),
					Location:         testReminderLocation,
					UserIDs:          []sharedtypes.DiscordID{testParticipant1.UserID, testParticipant2.UserID},
					ReminderType:     testReminderType,
					EventMessageID:   testDiscordMessageID,
					DiscordChannelID: "channel-123",
				},
			},
		},
		{
			name: "successful processing with enrichment",
			setup: func(f *FakeRepo, gcp *FakeGuildConfigProvider) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{testParticipant1}, nil
				}
				gcp.GetConfigFunc = func(ctx context.Context, g sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return &guildtypes.GuildConfig{
						EventChannelID: "enriched-channel",
					}, nil
				}
			},
			req: &roundtypes.ProcessRoundReminderRequest{
				RoundID:        testReminderRoundID,
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundTitle:     testReminderRoundTitle,
				StartTime:      testReminderStartTime.String(),
				Location:       testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: results.OperationResult[roundtypes.ProcessRoundReminderResult, error]{
				Success: &roundtypes.ProcessRoundReminderResult{
					RoundID:          testReminderRoundID,
					GuildID:          sharedtypes.GuildID("guild-123"),
					RoundTitle:       testReminderRoundTitle,
					StartTime:        testReminderStartTime.String(),
					Location:         testReminderLocation,
					UserIDs:          []sharedtypes.DiscordID{testParticipant1.UserID},
					ReminderType:     testReminderType,
					EventMessageID:   testDiscordMessageID,
					DiscordChannelID: "enriched-channel",
				},
			},
		},
		{
			name: "successful processing with no matching participants",
			setup: func(f *FakeRepo, gcp *FakeGuildConfigProvider) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{testParticipant3}, nil
				}
			},
			req: &roundtypes.ProcessRoundReminderRequest{
				RoundID:          testReminderRoundID,
				GuildID:          sharedtypes.GuildID("guild-123"),
				RoundTitle:       testReminderRoundTitle,
				StartTime:        testReminderStartTime.String(),
				Location:         testReminderLocation,
				ReminderType:     testReminderType,
				EventMessageID:   testDiscordMessageID,
				DiscordChannelID: "channel-123",
			},
			expectedResult: results.OperationResult[roundtypes.ProcessRoundReminderResult, error]{
				Success: &roundtypes.ProcessRoundReminderResult{
					RoundID:          testReminderRoundID,
					GuildID:          sharedtypes.GuildID("guild-123"),
					RoundTitle:       testReminderRoundTitle,
					StartTime:        testReminderStartTime.String(),
					Location:         testReminderLocation,
					UserIDs:          nil,
					ReminderType:     testReminderType,
					EventMessageID:   testDiscordMessageID,
					DiscordChannelID: "channel-123",
				},
			},
		},
		{
			name: "error retrieving participants",
			setup: func(f *FakeRepo, gcp *FakeGuildConfigProvider) {
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return nil, errors.New("database error")
				}
			},
			req: &roundtypes.ProcessRoundReminderRequest{
				RoundID:    testReminderRoundID,
				GuildID:    sharedtypes.GuildID("guild-123"),
				RoundTitle: testReminderRoundTitle,
			},
			expectedResult: results.OperationResult[roundtypes.ProcessRoundReminderResult, error]{
				Failure: ptr(errors.New("database error")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeRepo{}
			fakeGCP := &FakeGuildConfigProvider{}
			if tt.setup != nil {
				tt.setup(fakeRepo, fakeGCP)
			}

			s := &RoundService{
				repo:                fakeRepo,
				logger:              logger,
				metrics:             mockMetrics,
				tracer:              tracer,
				roundValidator:      &FakeRoundValidator{},
				eventBus:            &FakeEventBus{},
				parserFactory:       &StubFactory{},
				guildConfigProvider: fakeGCP,
			}

			result, err := s.ProcessRoundReminder(ctx, tt.req)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got nil")
				} else {
					if result.Success.RoundID != tt.expectedResult.Success.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", tt.expectedResult.Success.RoundID, result.Success.RoundID)
					}
					if len(result.Success.UserIDs) != len(tt.expectedResult.Success.UserIDs) {
						t.Errorf("expected %d user IDs, got %d", len(tt.expectedResult.Success.UserIDs), len(result.Success.UserIDs))
					}
					if result.Success.DiscordChannelID != tt.expectedResult.Success.DiscordChannelID {
						t.Errorf("expected DiscordChannelID: %q, got: %q", tt.expectedResult.Success.DiscordChannelID, result.Success.DiscordChannelID)
					}
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil")
				} else {
					if (*result.Failure).Error() != (*tt.expectedResult.Failure).Error() {
						t.Errorf("expected Error: %v, got: %v", (*tt.expectedResult.Failure).Error(), (*result.Failure).Error())
					}
				}
			}

			if tt.verify != nil {
				tt.verify(t, result)
			}
		})
	}
}
