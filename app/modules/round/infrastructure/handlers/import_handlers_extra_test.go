package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// =============================================================================
// TEST: HandleScorecardURLRequested
// =============================================================================

func TestRoundHandlers_HandleScorecardURLRequested(t *testing.T) {
	ctx := context.Background()
	payload := &roundevents.ScorecardURLRequestedPayloadV1{
		UDiscURL: "https://udisc.com/export",
	}

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScorecardURLRequestedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Success - URL Accepted",
			fakeSetup: func(fake *FakeService) {
				fake.ScorecardURLRequestedFunc = func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
					return results.SuccessResult[roundtypes.CreateImportJobResult, error](roundtypes.CreateImportJobResult{
						Job: &roundtypes.ImportCreateJobInput{},
					}), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScorecardParseRequestedV1,
		},
		{
			name: "Failure - Invalid URL format",
			fakeSetup: func(fake *FakeService) {
				fake.ScorecardURLRequestedFunc = func(ctx context.Context, req *roundtypes.ImportCreateJobInput) (roundservice.CreateImportJobResult, error) {
					return results.FailureResult[roundtypes.CreateImportJobResult, error](errors.New("INVALID_URL")), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}
			logger := loggerfrolfbot.NoOpLogger
			h := &RoundHandlers{
				service: fakeService,
				logger:  logger,
			}

			res, err := h.HandleScorecardURLRequested(ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleScorecardURLRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && res[0].Topic != tt.wantResultTopic {
				t.Errorf("expected topic %s, got %s", tt.wantResultTopic, res[0].Topic)
			}
		})
	}
}

// =============================================================================
// TEST: HandleScorecardParsedForNormalization
// =============================================================================

func TestRoundHandlers_HandleScorecardParsedForNormalization(t *testing.T) {
	ctx := context.Background()
	payload := &roundevents.ParsedScorecardPayloadV1{
		ImportID: "imp-1",
		ParsedData: &roundtypes.ParsedScorecard{
			PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Tester"}},
		},
	}

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ParsedScorecardPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Success - Data Normalized",
			fakeSetup: func(fake *FakeService) {
				fake.NormalizeParsedScorecardFunc = func(ctx context.Context, p *roundtypes.ParsedScorecard, m roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error) {
					// Verify metadata was mapped correctly from payload
					if m.ImportID != "imp-1" {
						return results.OperationResult[*roundtypes.NormalizedScorecard, error]{}, errors.New("metadata mismatch")
					}
					return results.SuccessResult[*roundtypes.NormalizedScorecard, error](&roundtypes.NormalizedScorecard{}), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ScorecardNormalizedV1,
		},
		{
			name: "Failure - Normalization Logic Error",
			fakeSetup: func(fake *FakeService) {
				fake.NormalizeParsedScorecardFunc = func(ctx context.Context, p *roundtypes.ParsedScorecard, m roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error) {
					return results.FailureResult[*roundtypes.NormalizedScorecard, error](errors.New("normalization failed")), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}
			logger := loggerfrolfbot.NoOpLogger
			h := &RoundHandlers{
				service: fakeService,
				logger:  logger,
			}

			res, err := h.HandleScorecardParsedForNormalization(ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleScorecardParsedForNormalization() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && res[0].Topic != tt.wantResultTopic {
				t.Errorf("expected topic %s, got %s", tt.wantResultTopic, res[0].Topic)
			}
		})
	}
}

// =============================================================================
// TEST: HandleScorecardNormalized
// =============================================================================

func TestRoundHandlers_HandleScorecardNormalized(t *testing.T) {
	ctx := context.Background()
	payload := &roundevents.ScorecardNormalizedPayloadV1{ImportID: "imp-1"}

	tests := []struct {
		name            string
		fakeSetup       func(*FakeService)
		payload         *roundevents.ScorecardNormalizedPayloadV1
		wantErr         bool
		wantResultLen   int
		wantResultTopic string
		expectedErrMsg  string
	}{
		{
			name: "Success - Ingestion Complete",
			fakeSetup: func(fake *FakeService) {
				fake.IngestNormalizedScorecardFunc = func(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
					return results.SuccessResult[*roundtypes.IngestScorecardResult, error](&roundtypes.IngestScorecardResult{}), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportCompletedV1,
		},
		{
			name: "Failure - Name matching failed",
			fakeSetup: func(fake *FakeService) {
				fake.IngestNormalizedScorecardFunc = func(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
					return results.FailureResult[*roundtypes.IngestScorecardResult, error](errors.New("MATCH_ERROR")), nil
				}
			},
			payload:         payload,
			wantErr:         false,
			wantResultLen:   1,
			wantResultTopic: roundevents.ImportFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}
			logger := loggerfrolfbot.NoOpLogger
			h := &RoundHandlers{
				service: fakeService,
				logger:  logger,
			}

			res, err := h.HandleScorecardNormalized(ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleScorecardNormalized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && res[0].Topic != tt.wantResultTopic {
				t.Errorf("expected topic %s, got %s", tt.wantResultTopic, res[0].Topic)
			}
		})
	}
}
