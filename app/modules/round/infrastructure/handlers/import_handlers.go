package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScorecardUploaded transforms a scorecard upload into an import job request.
func (h *RoundHandlers) HandleScorecardUploaded(
	ctx context.Context,
	payload *roundevents.ScorecardUploadedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.CreateImportJob(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.ScorecardParseRequestedV1,
		roundevents.ImportFailedV1,
	), nil
}

// HandleScorecardURLRequested transforms a URL request into a parse request.
func (h *RoundHandlers) HandleScorecardURLRequested(
	ctx context.Context,
	payload *roundevents.ScorecardURLRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.HandleScorecardURLRequested(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.ScorecardParseRequestedV1,
		roundevents.ImportFailedV1,
	), nil
}

// HandleParseScorecardRequest handles the actual parsing of file data.
func (h *RoundHandlers) HandleParseScorecardRequest(
	ctx context.Context,
	payload *roundevents.ScorecardUploadedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Minimal diagnostic log specifically for data-heavy operations.
	h.logger.DebugContext(ctx, "parsing scorecard file data", attr.Int("size", len(payload.FileData)))

	result, err := h.service.ParseScorecard(ctx, *payload, payload.FileData)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		topic := roundevents.ImportFailedV1
		// Check for specific parse failure payload type
		if _, ok := result.Failure.(roundevents.ScorecardParseFailedPayloadV1); ok {
			topic = roundevents.ScorecardParseFailedV1
		}
		return []handlerwrapper.Result{
			{Topic: topic, Payload: result.Failure},
		}, nil
	}

	return []handlerwrapper.Result{
		{Topic: roundevents.ScorecardParsedForUserV1, Payload: result.Success},
	}, nil
}

// HandleUserMatchConfirmedForIngest transforms matched user data into a final ingestion request.
func (h *RoundHandlers) HandleUserMatchConfirmedForIngest(
	ctx context.Context,
	payload *userevents.UDiscMatchConfirmedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload.ParsedScores == nil {
		return nil, sharedtypes.ValidationError{Message: "no parsed scorecard data in match confirmed payload"}
	}

	// Type assert the ParsedScores interface field from the shared user events.
	parsed, ok := payload.ParsedScores.(*roundevents.ParsedScorecardPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "invalid parsed scorecard payload type"}
	}

	result, err := h.service.IngestParsedScorecard(ctx, *parsed)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.ImportCompletedV1,
		roundevents.ImportFailedV1,
	), nil
}

// HandleImportCompleted transforms a completed import into a score submission event.
func (h *RoundHandlers) HandleImportCompleted(
	ctx context.Context,
	payload *roundevents.ImportCompletedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if len(payload.Scores) == 0 {
		return nil, nil
	}

	res, err := h.service.ApplyImportedScores(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if res.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: roundevents.ImportFailedV1, Payload: res.Failure},
		}, nil
	}

	appliedPayload, ok := res.Success.(*roundevents.ImportScoresAppliedPayloadV1)
	if !ok {
		return nil, sharedtypes.ValidationError{Message: "unexpected success payload type"}
	}

	// Bridges the Import flow into the standard Round Finalization flow.
	return []handlerwrapper.Result{
		{
			Topic: roundevents.RoundAllScoresSubmittedV1,
			Payload: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        appliedPayload.GuildID,
				RoundID:        appliedPayload.RoundID,
				EventMessageID: appliedPayload.EventMessageID,
				RoundData: roundtypes.Round{
					ID:             appliedPayload.RoundID,
					EventMessageID: appliedPayload.EventMessageID,
					Participants:   appliedPayload.Participants,
				},
				Participants: appliedPayload.Participants,
			},
		},
	}, nil
}
