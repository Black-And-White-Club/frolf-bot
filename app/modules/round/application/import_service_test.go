package roundservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	sharedeventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	rounddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

type stubUserLookup struct {
	usernames    map[string]sharedtypes.DiscordID
	displayNames map[string]sharedtypes.DiscordID
}

func newStubUserLookup() *stubUserLookup {
	return &stubUserLookup{
		usernames:    make(map[string]sharedtypes.DiscordID),
		displayNames: make(map[string]sharedtypes.DiscordID),
	}
}

func (s *stubUserLookup) FindByNormalizedUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, normalizedUsername string) (*UserIdentity, error) {
	if id, ok := s.usernames[normalizedUsername]; ok {
		return &UserIdentity{UserID: id}, nil
	}
	return nil, nil
}

func (s *stubUserLookup) FindByNormalizedUDiscDisplayName(ctx context.Context, guildID sharedtypes.GuildID, normalizedDisplayName string) (*UserIdentity, error) {
	if id, ok := s.displayNames[normalizedDisplayName]; ok {
		return &UserIdentity{UserID: id}, nil
	}
	return nil, nil
}

func newTestRoundService(db rounddb.RoundDB, eventBus sharedeventbus.EventBus, lookup UserLookup) *RoundService {
	return NewRoundService(
		db,
		nil,
		eventBus,
		lookup,
		&roundmetrics.NoOpMetrics{},
		loggerfrolfbot.NoOpLogger,
		noop.NewTracerProvider().Tracer("test"),
		roundutil.NewRoundValidator(),
	)
}

func TestRoundService_CreateImportJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddbmocks.NewMockRoundDB(ctrl)
	ctx := context.Background()
	basePayload := roundevents.ScorecardUploadedPayload{
		GuildID:   sharedtypes.GuildID("guild-1"),
		RoundID:   sharedtypes.RoundID(uuid.New()),
		ImportID:  "import-1",
		UserID:    sharedtypes.DiscordID("111"),
		ChannelID: "chan-1",
		FileName:  "scores.csv",
	}

	service := newTestRoundService(mockDB, nil, nil)

	t.Run("fetch round error", func(t *testing.T) {
		payload := basePayload
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(nil, fmt.Errorf("boom"))

		result, err := service.CreateImportJob(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		require.Contains(t, result.Failure.(*roundevents.ImportFailedPayload).Error, "failed to fetch round")
	})

	t.Run("import conflict", func(t *testing.T) {
		payload := basePayload
		existing := &roundtypes.Round{ImportID: "other-id", ImportStatus: string(rounddb.ImportStatusPending)}
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(existing, nil)

		result, err := service.CreateImportJob(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		require.Contains(t, result.Failure.(*roundevents.ImportFailedPayload).Error, "another import is already in progress")
	})

	t.Run("success with file data", func(t *testing.T) {
		payload := basePayload
		payload.FileData = []byte("names,1\nPlayer,4")
		payload.Notes = "notes"
		roundEntity := &roundtypes.Round{}

		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(roundEntity, nil)
		mockDB.EXPECT().UpdateRound(gomock.Any(), payload.GuildID, payload.RoundID, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, updated *roundtypes.Round) (*roundtypes.Round, error) {
				require.Equal(t, payload.ImportID, updated.ImportID)
				require.Equal(t, string(rounddb.ImportStatusPending), string(updated.ImportStatus))
				require.Equal(t, string(rounddb.ImportTypeCSV), string(updated.ImportType))
				require.Equal(t, payload.FileName, updated.FileName)
				return updated, nil
			},
		)

		result, err := service.CreateImportJob(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		success := result.Success.(*roundevents.ScorecardUploadedPayload)
		require.Equal(t, payload.ImportID, success.ImportID)
	})
}

func TestRoundService_HandleScorecardURLRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddbmocks.NewMockRoundDB(ctrl)
	ctx := context.Background()
	payload := roundevents.ScorecardURLRequestedPayload{
		GuildID:  sharedtypes.GuildID("guild-1"),
		RoundID:  sharedtypes.RoundID(uuid.New()),
		ImportID: "import-url",
		UDiscURL: "https://udisc.com/score",
		Notes:    "note",
	}
	service := newTestRoundService(mockDB, nil, nil)

	t.Run("round missing", func(t *testing.T) {
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(nil, nil)

		result, err := service.HandleScorecardURLRequested(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		require.Contains(t, result.Failure.(*roundevents.ImportFailedPayload).Error, "round not found")
	})

	t.Run("db update error", func(t *testing.T) {
		roundEntity := &roundtypes.Round{}
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(roundEntity, nil)
		mockDB.EXPECT().UpdateRound(gomock.Any(), payload.GuildID, payload.RoundID, gomock.Any()).Return(nil, fmt.Errorf("save fail"))

		result, err := service.HandleScorecardURLRequested(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		require.Contains(t, result.Failure.(*roundevents.ImportFailedPayload).Error, "failed to persist UDisc URL")
	})

	t.Run("success updates round", func(t *testing.T) {
		roundEntity := &roundtypes.Round{}
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(roundEntity, nil)
		mockDB.EXPECT().UpdateRound(gomock.Any(), payload.GuildID, payload.RoundID, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, updated *roundtypes.Round) (*roundtypes.Round, error) {
				require.Equal(t, payload.UDiscURL, updated.UDiscURL)
				require.Equal(t, string(rounddb.ImportTypeURL), string(updated.ImportType))
				return updated, nil
			},
		)

		result, err := service.HandleScorecardURLRequested(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		success := result.Success.(*roundevents.ScorecardURLRequestedPayload)
		require.Equal(t, payload.UDiscURL, success.UDiscURL)
	})
}

func TestRoundService_ParseScorecard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddbmocks.NewMockRoundDB(ctrl)
	ctx := context.Background()
	basePayload := roundevents.ScorecardUploadedPayload{
		GuildID:   sharedtypes.GuildID("guild-1"),
		RoundID:   sharedtypes.RoundID(uuid.New()),
		ImportID:  "import-1",
		UserID:    sharedtypes.DiscordID("111"),
		ChannelID: "chan-1",
	}
	service := newTestRoundService(mockDB, nil, nil)

	t.Run("unsupported extension", func(t *testing.T) {
		payload := basePayload
		payload.FileName = "scores.txt"
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "").Return(nil)

		result, err := service.ParseScorecard(ctx, payload, []byte{})
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		failure := result.Failure.(*roundevents.ImportFailedPayload)
		require.Contains(t, failure.Error, "unsupported file format")
	})

	t.Run("parse csv file", func(t *testing.T) {
		payload := basePayload
		payload.FileName = "scores.csv"
		sample := "Name,1,2,3,Total\nPar,3,3,3,9\nPlayer One,3,4,3,10\n"

		gomock.InOrder(
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "").Return(nil),
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsed", "", "").Return(nil),
		)

		result, err := service.ParseScorecard(ctx, payload, []byte(sample))
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		parsedPayload := result.Success.(*roundevents.ParsedScorecardPayload)
		require.Equal(t, payload.ImportID, parsedPayload.ImportID)
		require.NotNil(t, parsedPayload.ParsedData)
		require.Len(t, parsedPayload.ParsedData.PlayerScores, 1)
		require.Len(t, parsedPayload.ParsedData.ParScores, 4)
	})

	t.Run("parse xlsx file", func(t *testing.T) {
		payload := basePayload
		payload.FileName = "scores.xlsx"
		xlsxData := buildXLSXBytes(t, [][]string{
			{"Name", "1", "2", "3"},
			{"Par", "3", "3", "3"},
			{"Player One", "3", "4", "3"},
		})

		gomock.InOrder(
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "").Return(nil),
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsed", "", "").Return(nil),
		)

		result, err := service.ParseScorecard(ctx, payload, xlsxData)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		parsedPayload := result.Success.(*roundevents.ParsedScorecardPayload)
		require.Equal(t, payload.ImportID, parsedPayload.ImportID)
		require.NotNil(t, parsedPayload.ParsedData)
		require.Len(t, parsedPayload.ParsedData.PlayerScores, 1)
		require.Len(t, parsedPayload.ParsedData.ParScores, 3)
	})

	t.Run("parse failure updates status", func(t *testing.T) {
		payload := basePayload
		payload.FileName = "scores.xlsx"

		gomock.InOrder(
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "").Return(nil),
			mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "failed", gomock.Any(), "PARSE_ERROR").Return(nil),
		)

		result, err := service.ParseScorecard(ctx, payload, []byte("not-a-real-xlsx"))
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
		failure := result.Failure.(*roundevents.ImportFailedPayload)
		require.Equal(t, "PARSE_ERROR", failure.ErrorCode)
		require.Contains(t, failure.Error, "failed to parse scorecard")
	})
}

func TestRoundService_IngestParsedScorecard(t *testing.T) {
	t.Run("parsed data missing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDB := rounddbmocks.NewMockRoundDB(ctrl)
		ctx := context.Background()
		payload := roundevents.ParsedScorecardPayload{
			GuildID:  sharedtypes.GuildID("guild-1"),
			RoundID:  sharedtypes.RoundID(uuid.New()),
			ImportID: "import-1",
		}
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "").Return(nil)

		service := newTestRoundService(mockDB, nil, nil)
		result, err := service.IngestParsedScorecard(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
	})

	t.Run("round missing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDB := rounddbmocks.NewMockRoundDB(ctrl)
		ctx := context.Background()
		payload := roundevents.ParsedScorecardPayload{
			GuildID:  sharedtypes.GuildID("guild-1"),
			RoundID:  sharedtypes.RoundID(uuid.New()),
			ImportID: "import-2",
			ParsedData: &roundtypes.ParsedScorecard{
				PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Player"}},
			},
		}
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "").Return(nil)
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(nil, nil)
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "failed", "failed to fetch round", "ROUND_NOT_FOUND").Return(nil)

		service := newTestRoundService(mockDB, nil, nil)
		result, err := service.IngestParsedScorecard(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
	})

	t.Run("no matches", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDB := rounddbmocks.NewMockRoundDB(ctrl)
		ctx := context.Background()
		payload := roundevents.ParsedScorecardPayload{
			GuildID:  sharedtypes.GuildID("guild-1"),
			RoundID:  sharedtypes.RoundID(uuid.New()),
			ImportID: "import-3",
			ParsedData: &roundtypes.ParsedScorecard{
				PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Unknown"}},
			},
		}
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "").Return(nil)
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(&roundtypes.Round{}, nil)
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "failed", "no players matched (1 total in scorecard)", "NO_MATCHES").Return(nil)

		service := newTestRoundService(mockDB, nil, nil)
		result, err := service.IngestParsedScorecard(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Failure)
	})

	t.Run("existing participant", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDB := rounddbmocks.NewMockRoundDB(ctrl)
		mockEventBus := mocks.NewMockEventBus(ctrl)
		ctx := context.Background()
		player := roundtypes.PlayerScoreRow{PlayerName: "Matched One", HoleScores: []int{3, 3}, Total: 6}
		payload := roundevents.ParsedScorecardPayload{
			GuildID:   sharedtypes.GuildID("guild-1"),
			RoundID:   sharedtypes.RoundID(uuid.New()),
			ImportID:  "import-4",
			UserID:    sharedtypes.DiscordID("importer"),
			ChannelID: "chan-1",
			ParsedData: &roundtypes.ParsedScorecard{
				PlayerScores: []roundtypes.PlayerScoreRow{player},
			},
		}
		matchedID := sharedtypes.DiscordID("matched-1")
		tag := sharedtypes.TagNumber(8)
		round := &roundtypes.Round{
			Participants: []roundtypes.Participant{{UserID: matchedID, TagNumber: &tag, Response: roundtypes.ResponseAccept}},
		}

		lookup := newStubUserLookup()
		lookup.usernames[normalizeName(player.PlayerName)] = matchedID

		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "").Return(nil)
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(round, nil)
		mockEventBus.EXPECT().Publish(roundevents.ImportCompletedTopic, gomock.Any()).DoAndReturn(func(topic string, msgs ...*message.Message) error {
			require.Len(t, msgs, 1)
			var completed roundevents.ImportCompletedPayload
			require.NoError(t, json.Unmarshal(msgs[0].Payload, &completed))
			require.Equal(t, 1, completed.MatchedPlayers)
			require.Equal(t, 0, completed.PlayersAutoAdded)
			return nil
		})
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "processing", "", "").Return(nil)

		service := newTestRoundService(mockDB, mockEventBus, lookup)
		result, err := service.IngestParsedScorecard(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		success := result.Success.(*scoreevents.ProcessRoundScoresRequestPayload)
		require.Len(t, success.Scores, 1)
		require.False(t, success.Overwrite)
		require.Equal(t, sharedtypes.Score(6), success.Scores[0].Score)
		require.NotNil(t, success.Scores[0].TagNumber)
	})

	t.Run("auto add participant", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDB := rounddbmocks.NewMockRoundDB(ctrl)
		mockEventBus := mocks.NewMockEventBus(ctrl)
		ctx := context.Background()
		player := roundtypes.PlayerScoreRow{PlayerName: "Auto Add", HoleScores: []int{2, 2}, Total: 4}
		payload := roundevents.ParsedScorecardPayload{
			GuildID:   sharedtypes.GuildID("guild-1"),
			RoundID:   sharedtypes.RoundID(uuid.New()),
			ImportID:  "import-5",
			UserID:    sharedtypes.DiscordID("importer"),
			ChannelID: "chan-2",
			ParsedData: &roundtypes.ParsedScorecard{
				PlayerScores: []roundtypes.PlayerScoreRow{player},
			},
		}
		matchedID := sharedtypes.DiscordID("matched-2")

		lookup := newStubUserLookup()
		lookup.usernames[normalizeName(player.PlayerName)] = matchedID

		newParticipantMatcher := gomock.AssignableToTypeOf(roundtypes.Participant{})
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "").Return(nil)
		mockDB.EXPECT().GetRound(gomock.Any(), payload.GuildID, payload.RoundID).Return(&roundtypes.Round{}, nil)
		mockDB.EXPECT().UpdateParticipant(gomock.Any(), payload.GuildID, payload.RoundID, newParticipantMatcher).Return(nil, nil)
		mockEventBus.EXPECT().Publish(roundevents.RoundParticipantAutoAddedTopic, gomock.Any()).DoAndReturn(func(topic string, msgs ...*message.Message) error {
			require.Len(t, msgs, 1)
			var added roundevents.RoundParticipantAutoAddedPayload
			require.NoError(t, json.Unmarshal(msgs[0].Payload, &added))
			require.Equal(t, matchedID, added.AddedUser)
			return nil
		})
		mockEventBus.EXPECT().Publish(roundevents.ImportCompletedTopic, gomock.Any()).DoAndReturn(func(topic string, msgs ...*message.Message) error {
			require.Len(t, msgs, 1)
			var completed roundevents.ImportCompletedPayload
			require.NoError(t, json.Unmarshal(msgs[0].Payload, &completed))
			require.Equal(t, 1, completed.PlayersAutoAdded)
			return nil
		})
		mockDB.EXPECT().UpdateImportStatus(gomock.Any(), payload.GuildID, payload.RoundID, payload.ImportID, "processing", "", "").Return(nil)

		service := newTestRoundService(mockDB, mockEventBus, lookup)
		result, err := service.IngestParsedScorecard(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		success := result.Success.(*scoreevents.ProcessRoundScoresRequestPayload)
		require.Len(t, success.Scores, 1)
		require.False(t, success.Overwrite)
		require.Equal(t, sharedtypes.Score(4), success.Scores[0].Score)
		require.Nil(t, success.Scores[0].TagNumber)
	})
}

func buildXLSXBytes(t *testing.T, rows [][]string) []byte {
	t.Helper()

	f := excelize.NewFile()
	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	for idx, row := range rows {
		axis, err := excelize.CoordinatesToCellName(1, idx+1)
		require.NoError(t, err)
		cells := make([]interface{}, len(row))
		for i, val := range row {
			cells[i] = val
		}
		require.NoError(t, f.SetSheetRow(sheet, axis, &cells))
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	require.NoError(t, f.Close())
	return buf.Bytes()
}
