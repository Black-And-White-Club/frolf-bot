package roundhandlers

import (
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestHandleImportCompleted_FanOutMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-1"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	payload := &roundevents.ImportCompletedPayload{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{{UserID: sharedtypes.DiscordID("u1"), Score: 5}},
	}

	msg := message.NewMessage("m1", nil)

	// Unmarshal will populate payload
	mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(m *message.Message, out interface{}) error {
			*out.(*roundevents.ImportCompletedPayload) = *payload
			return nil
		},
	)

	// Service returns applied snapshot
	finalParticipants := []roundtypes.Participant{{UserID: sharedtypes.DiscordID("u1"), Score: func() *sharedtypes.Score { s := sharedtypes.Score(5); return &s }()}}
	mockService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
		roundservice.RoundOperationResult{Success: &roundevents.ImportScoresAppliedPayload{GuildID: guildID, RoundID: roundID, ImportID: importID, Participants: finalParticipants, EventMessageID: ""}}, nil,
	)

	// Expect two CreateResultMessage calls (discord + backend)
	mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundParticipantScoreUpdated).Times(2).
		DoAndReturn(func(original *message.Message, payload any, topic string) (*message.Message, error) {
			return message.NewMessage("out", nil), nil
		})

	h := NewRoundHandlers(mockService, logger, tracer, mockHelpers, metrics)
	out, err := h.HandleImportCompleted(msg)
	require.NoError(t, err)
	require.Len(t, out, 2)
}

func TestHandleImportCompleted_ServiceFailureProducesImportFailedMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-err"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	payload := &roundevents.ImportCompletedPayload{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{{UserID: sharedtypes.DiscordID("u1"), Score: 5}},
	}

	msg := message.NewMessage("m1", nil)

	mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(m *message.Message, out interface{}) error {
			*out.(*roundevents.ImportCompletedPayload) = *payload
			return nil
		},
	)

	mockService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
		roundservice.RoundOperationResult{Failure: &roundevents.ImportFailedPayload{GuildID: guildID, RoundID: roundID, ImportID: importID, Error: "all failed"}}, nil,
	)

	mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(message.NewMessage("fail", nil), nil)

	h := NewRoundHandlers(mockService, logger, tracer, mockHelpers, metrics)
	out, err := h.HandleImportCompleted(msg)
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestHandleImportCompleted_NoScoresReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-none"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	payload := &roundevents.ImportCompletedPayload{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{},
	}

	msg := message.NewMessage("m1", nil)

	mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(m *message.Message, out interface{}) error {
			*out.(*roundevents.ImportCompletedPayload) = *payload
			return nil
		},
	)

	// Service should NOT be called when there are no scores
	mockService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Times(0)

	h := NewRoundHandlers(mockService, logger, tracer, mockHelpers, metrics)
	out, err := h.HandleImportCompleted(msg)
	require.NoError(t, err)
	require.Nil(t, out)
}

func TestHandleImportCompleted_UnexpectedSuccessTypeReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-badtype"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	payload := &roundevents.ImportCompletedPayload{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{{UserID: sharedtypes.DiscordID("u1"), Score: 1}},
	}

	msg := message.NewMessage("m1", nil)

	mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(m *message.Message, out interface{}) error {
			*out.(*roundevents.ImportCompletedPayload) = *payload
			return nil
		},
	)

	// Return a success payload of the wrong concrete type (string) to trigger type assertion error
	mockService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
		roundservice.RoundOperationResult{Success: "not-the-right-type"}, nil,
	)

	h := NewRoundHandlers(mockService, logger, tracer, mockHelpers, metrics)
	_, err := h.HandleImportCompleted(msg)
	require.Error(t, err)
}

func TestHandleImportCompleted_CreateResultMessageBackendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	importID := "imp-2"
	guildID := sharedtypes.GuildID("g-1")
	roundID := sharedtypes.RoundID(uuid.New())

	payload := &roundevents.ImportCompletedPayload{
		ImportID: importID,
		GuildID:  guildID,
		RoundID:  roundID,
		Scores:   []sharedtypes.ScoreInfo{{UserID: sharedtypes.DiscordID("u1"), Score: 3}},
	}

	msg := message.NewMessage("m1", nil)

	mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(m *message.Message, out interface{}) error {
			*out.(*roundevents.ImportCompletedPayload) = *payload
			return nil
		},
	)

	finalParticipants := []roundtypes.Participant{{UserID: sharedtypes.DiscordID("u1"), Score: func() *sharedtypes.Score { s := sharedtypes.Score(3); return &s }()}}
	mockService.EXPECT().ApplyImportedScores(gomock.Any(), gomock.Any()).Return(
		roundservice.RoundOperationResult{Success: &roundevents.ImportScoresAppliedPayload{GuildID: guildID, RoundID: roundID, ImportID: importID, Participants: finalParticipants, EventMessageID: ""}}, nil,
	)

	// First CreateResultMessage (discord) returns ok
	mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundParticipantScoreUpdated).Return(message.NewMessage("ok", nil), nil)
	// Second CreateResultMessage (backend) returns an error
	mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundParticipantScoreUpdated).Return(nil, assert.AnError).Times(1)

	h := NewRoundHandlers(mockService, logger, tracer, mockHelpers, metrics)
	_, err := h.HandleImportCompleted(msg)
	require.Error(t, err)
}
