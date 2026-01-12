package roundhandlers

import (
	"context"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleScorecardUploaded(t *testing.T) {
	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testFileName := "test-scorecard.csv"
	testFileContent := []byte("test content")

	testPayload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
		FileName: testFileName,
		FileData: testFileContent,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ScorecardUploadedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScorecardUploaded",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardUploadedPayloadV1{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							FileName: testFileName,
							FileData: testFileContent,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScorecardParseRequestedV1,
		},
		{
			name: "Handle CreateImportJob error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
		{
			name: "Handle CreateImportJob failure result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "import failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			results, err := h.HandleScorecardUploaded(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardUploaded() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScorecardUploaded() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScorecardUploaded() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScorecardUploaded() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardURLRequested(t *testing.T) {
	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUDiscURL := "https://udisc.com/scorecard/12345"

	testPayload := &roundevents.ScorecardURLRequestedPayloadV1{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
		UDiscURL: testUDiscURL,
	}

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name            string
		mockSetup       func(*roundmocks.MockService)
		payload         *roundevents.ScorecardURLRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Successfully handle ScorecardURLRequested",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardURLRequestedPayloadV1{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							UDiscURL: testUDiscURL,
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScorecardParseRequestedV1,
		},
		{
			name: "Handle HandleScorecardURLRequested error",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
		{
			name: "Handle HandleScorecardURLRequested failure result",
			mockSetup: func(mockRoundService *roundmocks.MockService) {
				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "url parse failed",
						},
					},
					nil,
				)
			},
			payload:         testPayload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			tt.mockSetup(mockRoundService)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			results, err := h.HandleScorecardURLRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardURLRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScorecardURLRequested() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleScorecardURLRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}
			if tt.wantResultLen > 0 && results[0].Topic != tt.wantResultTopic {
				t.Errorf("HandleScorecardURLRequested() result topic = %v, want %v", results[0].Topic, tt.wantResultTopic)
			}
		})
	}
}
