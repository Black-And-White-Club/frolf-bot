package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
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
		name         string
		setupService func(s *FakeService)
		expectTopic  string
	}{
		{
			name: "Success - URL Accepted",
			setupService: func(s *FakeService) {
				s.ScorecardURLRequestedFn = func(ctx context.Context, p roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{Success: &roundevents.ScorecardUploadedPayloadV1{}}, nil
				}
			},
			expectTopic: string(roundevents.ScorecardParseRequestedV1),
		},
		{
			name: "Failure - Invalid URL format",
			setupService: func(s *FakeService) {
				s.ScorecardURLRequestedFn = func(ctx context.Context, p roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{ErrorCode: "INVALID_URL"},
					}, nil
				}
			},
			expectTopic: string(roundevents.ImportFailedV1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewFakeService()
			tt.setupService(svc)
			h := &RoundHandlers{service: svc}

			res, err := h.HandleScorecardURLRequested(ctx, payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res[0].Topic != tt.expectTopic {
				t.Errorf("expected topic %s, got %s", tt.expectTopic, res[0].Topic)
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
		name         string
		setupService func(s *FakeService)
		expectTopic  string
	}{
		{
			name: "Success - Data Normalized",
			setupService: func(s *FakeService) {
				s.NormalizeParsedScorecardFn = func(ctx context.Context, p *roundtypes.ParsedScorecard, m roundtypes.Metadata) (results.OperationResult, error) {
					// Verify metadata was mapped correctly from payload
					if m.ImportID != "imp-1" {
						return results.OperationResult{}, errors.New("metadata mismatch")
					}
					return results.OperationResult{Success: &roundevents.ScorecardNormalizedPayloadV1{}}, nil
				}
			},
			expectTopic: string(roundevents.ScorecardNormalizedV1),
		},
		{
			name: "Failure - Normalization Logic Error",
			setupService: func(s *FakeService) {
				s.NormalizeParsedScorecardFn = func(ctx context.Context, p *roundtypes.ParsedScorecard, m roundtypes.Metadata) (results.OperationResult, error) {
					return results.OperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{Error: "normalization failed"},
					}, nil
				}
			},
			expectTopic: string(roundevents.ImportFailedV1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewFakeService()
			tt.setupService(svc)
			h := &RoundHandlers{service: svc}

			res, err := h.HandleScorecardParsedForNormalization(ctx, payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res[0].Topic != tt.expectTopic {
				t.Errorf("expected topic %s, got %s", tt.expectTopic, res[0].Topic)
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
		name         string
		setupService func(s *FakeService)
		expectTopic  string
	}{
		{
			name: "Success - Ingestion Complete",
			setupService: func(s *FakeService) {
				s.IngestNormalizedScorecardFn = func(ctx context.Context, p roundevents.ScorecardNormalizedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{Success: &roundevents.ImportCompletedPayloadV1{}}, nil
				}
			},
			expectTopic: string(roundevents.ImportCompletedV1),
		},
		{
			name: "Failure - Name matching failed",
			setupService: func(s *FakeService) {
				s.IngestNormalizedScorecardFn = func(ctx context.Context, p roundevents.ScorecardNormalizedPayloadV1) (results.OperationResult, error) {
					return results.OperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{ErrorCode: "MATCH_ERROR"},
					}, nil
				}
			},
			expectTopic: string(roundevents.ImportFailedV1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewFakeService()
			tt.setupService(svc)
			h := &RoundHandlers{service: svc}

			res, err := h.HandleScorecardNormalized(ctx, payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res[0].Topic != tt.expectTopic {
				t.Errorf("expected topic %s, got %s", tt.expectTopic, res[0].Topic)
			}
		})
	}
}
