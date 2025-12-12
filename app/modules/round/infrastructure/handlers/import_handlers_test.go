package roundhandlers

import (
	"encoding/json"
	"fmt"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleScorecardUploaded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testFileName := "test-scorecard.csv"
	testFileContent := []byte("test content")

	testPayload := &roundevents.ScorecardUploadedPayload{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
		FileName: testFileName,
		FileData: testFileContent,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ScorecardUploaded",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardUploadedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							FileName: testFileName,
							FileData: testFileContent,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParseRequestTopic).Return(
					message.NewMessage("result-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("result-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle invalid payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unmarshal error"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: unmarshal error",
		},
		{
			name: "Handle CreateImportJob error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScorecardUploaded event: service error",
		},
		{
			name: "Handle CreateImportJob failure result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "import failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					message.NewMessage("failure-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("failure-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle CreateImportJob failure result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "import failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: create message error",
		},
		{
			name: "Handle CreateImportJob success result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardUploadedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							FileName: testFileName,
							FileData: testFileContent,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParseRequestTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create parse request message: create message error",
		},
		{
			name: "Handle unexpected result from CreateImportJob",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CreateImportJob(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewRoundHandlers(mockRoundService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleScorecardUploaded(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleScorecardUploaded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("RoundHandlers.HandleScorecardUploaded() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleScorecardUploaded() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardURLRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUDiscURL := "https://udisc.com/scorecard/12345"

	testPayload := &roundevents.ScorecardURLRequestedPayload{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
		UDiscURL: testUDiscURL,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ScorecardURLRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardUploadedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							UDiscURL: testUDiscURL,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParseRequestTopic).Return(
					message.NewMessage("result-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("result-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle invalid payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unmarshal error"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: unmarshal error",
		},
		{
			name: "Handle HandleScorecardURLRequested error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScorecardURLRequested event: service error",
		},
		{
			name: "Handle HandleScorecardURLRequested failure result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "import failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					message.NewMessage("failure-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("failure-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle HandleScorecardURLRequested failure result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "import failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: create message error",
		},
		{
			name: "Handle HandleScorecardURLRequested success result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScorecardUploadedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							UDiscURL: testUDiscURL,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParseRequestTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create parse request message: create message error",
		},
		{
			name: "Handle unexpected result from HandleScorecardURLRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardURLRequestedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().HandleScorecardURLRequested(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewRoundHandlers(mockRoundService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleScorecardURLRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleScorecardURLRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("RoundHandlers.HandleScorecardURLRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleScorecardURLRequested() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleParseScorecardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testFileName := "test-scorecard.csv"
	testFileContent := []byte("test content")

	testPayload := &roundevents.ScorecardUploadedPayload{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
		FileName: testFileName,
		FileData: testFileContent,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ParseScorecardRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParsedTopic).Return(
					message.NewMessage("result-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("result-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle invalid payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unmarshal error"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: unmarshal error",
		},
		{
			name: "Handle ParseScorecard error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParseScorecardRequest event: service error",
		},
		{
			name: "Handle ParseScorecard failure result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "parse failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					message.NewMessage("failure-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("failure-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle ParseScorecard failure result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "parse failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: create message error",
		},
		{
			name: "Handle ParseScorecard success result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ScorecardParsedTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: create message error",
		},
		{
			name: "Handle unexpected result from ParseScorecard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScorecardUploadedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParseScorecard(gomock.Any(), *testPayload, testFileContent).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewRoundHandlers(mockRoundService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleParseScorecardRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleParseScorecardRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("RoundHandlers.HandleParseScorecardRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleParseScorecardRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleScorecardParsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "test-import-id"
	testGuildID := sharedtypes.GuildID("test-guild-id")
	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.ParsedScorecardPayload{
		ImportID: testImportID,
		GuildID:  testGuildID,
		RoundID:  testRoundID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ScorecardParsed",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &scoreevents.ProcessRoundScoresRequestPayload{
							RoundID: testRoundID,
							GuildID: testGuildID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), scoreevents.ProcessRoundScoresRequest).Return(
					message.NewMessage("result-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("result-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle invalid payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unmarshal error"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: unmarshal error",
		},
		{
			name: "Handle IngestParsedScorecard error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScorecardParsed event: service error",
		},
		{
			name: "Handle IngestParsedScorecard failure result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "ingest failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					message.NewMessage("failure-id", nil), nil,
				)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("failure-id", nil)},
			wantErr: false,
		},
		{
			name: "Handle IngestParsedScorecard failure result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ImportFailedPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
							Error:    "ingest failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.ImportFailedTopic).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: create message error",
		},
		{
			name: "Handle IngestParsedScorecard success result but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{
						Success: &scoreevents.ProcessRoundScoresRequestPayload{
							RoundID: testRoundID,
							GuildID: testGuildID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), scoreevents.ProcessRoundScoresRequest).Return(
					nil, fmt.Errorf("create message error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create score processing message: create message error",
		},
		{
			name: "Handle unexpected result from IngestParsedScorecard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().IngestParsedScorecard(gomock.Any(), *testPayload).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewRoundHandlers(mockRoundService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleScorecardParsed(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleScorecardParsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("RoundHandlers.HandleScorecardParsed() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleScorecardParsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}
