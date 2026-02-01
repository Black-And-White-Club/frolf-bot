package userhandlers

import (
	"context"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleScorecardParsed(t *testing.T) {
	testImportID := "import-123"
	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("user-123")

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name       string
		payload    *roundevents.ParsedScorecardPayloadV1
		setupFake  func(*FakeUserService)
		wantLen    int
		wantTopics []string
		wantErr    bool
	}{
		{
			name: "Success - Mixed results (confirmed and unmatched)",
			payload: &roundevents.ParsedScorecardPayloadV1{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
				UserID:   testUserID,
				// Ensure these match your actual roundevents definitions
				ParsedData: &roundtypes.ParsedScorecard{
					PlayerScores: []roundtypes.PlayerScoreRow{
						{PlayerName: "Confirmed Player"},
						{PlayerName: "Unmatched Player"},
					},
				},
			},
			setupFake: func(f *FakeUserService) {
				f.MatchParsedScorecardFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (results.OperationResult[*userservice.MatchResult, error], error) {
					return results.SuccessResult[*userservice.MatchResult, error](&userservice.MatchResult{
						Mappings: []userevents.UDiscConfirmedMappingV1{
							{
								PlayerName:    "Confirmed Player",
								DiscordUserID: sharedtypes.DiscordID("mapped-id"),
							},
						},
						Unmatched: []string{"Unmatched Player"},
					}), nil
				}
			},
			wantLen:    2,
			wantTopics: []string{userevents.UDiscMatchConfirmedV1, userevents.UDiscMatchConfirmationRequiredV1},
			wantErr:    false,
		},
		{
			name: "Failure - Service Infrastructure Error",
			payload: &roundevents.ParsedScorecardPayloadV1{
				ImportID: testImportID,
				GuildID:  testGuildID,
				ParsedData: &roundtypes.ParsedScorecard{
					PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Any"}},
				},
			},
			setupFake: func(f *FakeUserService) {
				f.MatchParsedScorecardFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (results.OperationResult[*userservice.MatchResult, error], error) {
					return results.OperationResult[*userservice.MatchResult, error]{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fake)
			}

			h := NewUserHandlers(fake, logger, tracer, nil, metrics)
			res, err := h.HandleScorecardParsed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(res) != tt.wantLen {
					t.Errorf("got %d results, want %d", len(res), tt.wantLen)
				}
				for i, topic := range tt.wantTopics {
					if i < len(res) && res[i].Topic != topic {
						t.Errorf("result[%d] topic = %s, want %s", i, res[i].Topic, topic)
					}
				}
			}
		})
	}
}
