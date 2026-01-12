package userhandlers

import (
	"context"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleScorecardParsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "import-123"
	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *roundevents.ParsedScorecardPayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Match confirmation required",
			payload: &roundevents.ParsedScorecardPayloadV1{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmationRequiredPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UDiscMatchConfirmationRequiredV1,
			wantErr:   false,
		},
		{
			name: "Match confirmed",
			payload: &roundevents.ParsedScorecardPayloadV1{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmedPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UDiscMatchConfirmedV1,
			wantErr:   false,
		},
		{
			name: "Unexpected result (nil success and failure)",
			payload: &roundevents.ParsedScorecardPayloadV1{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{}, nil)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)
			results, err := h.HandleScorecardParsed(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardParsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleScorecardParsed() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if tt.wantLen > 0 && results[0].Topic != tt.wantTopic {
					t.Errorf("HandleScorecardParsed() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
