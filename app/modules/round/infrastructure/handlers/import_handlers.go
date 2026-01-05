package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleScorecardUploaded handles scorecard uploaded events.
func (h *RoundHandlers) HandleScorecardUploaded(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScorecardUploaded",
		&roundevents.ScorecardUploadedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreboardUploadedPayload := payload.(*roundevents.ScorecardUploadedPayloadV1)

			h.logger.InfoContext(ctx, "Received ScorecardUploaded event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", scoreboardUploadedPayload.ImportID),
				attr.String("guild_id", string(scoreboardUploadedPayload.GuildID)),
				attr.String("round_id", scoreboardUploadedPayload.RoundID.String()),
				attr.String("file_name", scoreboardUploadedPayload.FileName),
			)

			result, err := h.roundService.CreateImportJob(ctx, *scoreboardUploadedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScorecardUploaded event",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to handle ScorecardUploaded event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Import job creation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Publish failure event
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.ImportFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Import job created successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload", result.Success),
				)

				parseMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ScorecardParseRequestedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create parse request message: %w", err)
				}

				return []*message.Message{parseMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from CreateImportJob service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleScorecardURLRequested handles scorecard URL requested events.
func (h *RoundHandlers) HandleScorecardURLRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScorecardURLRequested",
		&roundevents.ScorecardURLRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreboardURLRequestedPayload := payload.(*roundevents.ScorecardURLRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received ScorecardURLRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", scoreboardURLRequestedPayload.ImportID),
				attr.String("guild_id", string(scoreboardURLRequestedPayload.GuildID)),
				attr.String("round_id", scoreboardURLRequestedPayload.RoundID.String()),
				attr.String("udisc_url", scoreboardURLRequestedPayload.UDiscURL),
			)

			result, err := h.roundService.HandleScorecardURLRequested(ctx, *scoreboardURLRequestedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScorecardURLRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to handle ScorecardURLRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scorecard URL request handling failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Publish failure event
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.ImportFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Scorecard URL request handled successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload", result.Success),
				)

				parseMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ScorecardParseRequestedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create parse request message: %w", err)
				}

				return []*message.Message{parseMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from HandleScorecardURLRequested service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleParseScorecardRequest handles parse scorecard requests.
func (h *RoundHandlers) HandleParseScorecardRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParseScorecardRequest",
		&roundevents.ScorecardUploadedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreboardUploadedPayload := payload.(*roundevents.ScorecardUploadedPayloadV1)

			h.logger.InfoContext(ctx, "Received ParseScorecardRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", scoreboardUploadedPayload.ImportID),
				attr.String("guild_id", string(scoreboardUploadedPayload.GuildID)),
				attr.String("round_id", scoreboardUploadedPayload.RoundID.String()),
				attr.String("file_name", scoreboardUploadedPayload.FileName),
			)

			// Get file data from payload (would be the actual file bytes)
			fileData := scoreboardUploadedPayload.FileData

			headerLen := 10
			if len(fileData) < headerLen {
				headerLen = len(fileData)
			}
			h.logger.InfoContext(ctx, "Inspecting file data",
				attr.Int("file_size", len(fileData)),
				attr.String("file_header", fmt.Sprintf("%x", fileData[:headerLen])),
			)

			result, err := h.roundService.ParseScorecard(ctx, *scoreboardUploadedPayload, fileData)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ParseScorecardRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to handle ParseScorecardRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scorecard parsing failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Determine the topic based on the failure payload type
				topic := roundevents.ImportFailedV1
				if _, ok := result.Failure.(roundevents.ScorecardParseFailedPayload); ok {
					topic = roundevents.ScorecardParseFailedV1
				}

				// Publish failure event
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					topic,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Scorecard parsed successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload", result.Success),
				)

				// Publish parsed scorecard to user module for player matching
				userMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ScorecardParsedForUserV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create parsed scorecard user message: %w", err)
				}

				return []*message.Message{userMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ParseScorecard service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleUserMatchConfirmedForIngest ingests parsed scorecards after user matching completes and publishes score processing requests.
func (h *RoundHandlers) HandleUserMatchConfirmedForIngest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleUserMatchConfirmedForIngest",
		&userevents.UDiscMatchConfirmedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			matchedPayload := payload.(*userevents.UDiscMatchConfirmedPayload)

			h.logger.InfoContext(ctx, "Received user match confirmed for score ingestion",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", matchedPayload.ImportID),
				attr.String("guild_id", string(matchedPayload.GuildID)),
				attr.String("round_id", matchedPayload.RoundID.String()),
			)

			// Extract the parsed scorecard from the payload
			// The user module should have attached it when confirming matches
			parsedScorecardRaw := matchedPayload.ParsedScores
			if parsedScorecardRaw == nil {
				h.logger.ErrorContext(ctx, "No parsed scorecard data in match confirmed payload",
					attr.CorrelationIDFromMsg(msg),
					attr.String("import_id", matchedPayload.ImportID),
				)
				return nil, fmt.Errorf("no parsed scorecard data in match confirmed payload")
			}

			// Convert interface{} to ParsedScorecardPayload
			// When JSON unmarshals into interface{}, it creates a map[string]interface{}
			// We need to re-marshal and unmarshal to get the typed struct
			parsedPayload := &roundevents.ParsedScorecardPayloadV1{}
			parsedBytes, err := json.Marshal(parsedScorecardRaw)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to marshal parsed scorecard data",
					attr.CorrelationIDFromMsg(msg),
					attr.String("import_id", matchedPayload.ImportID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to marshal parsed scorecard data: %w", err)
			}

			if err := json.Unmarshal(parsedBytes, parsedPayload); err != nil {
				h.logger.ErrorContext(ctx, "Failed to unmarshal parsed scorecard data",
					attr.CorrelationIDFromMsg(msg),
					attr.String("import_id", matchedPayload.ImportID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to unmarshal parsed scorecard data: %w", err)
			}

			result, err := h.roundService.IngestParsedScorecard(ctx, *parsedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to ingest scorecard after user matching",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to ingest scorecard after user matching: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scorecard ingestion failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.ImportFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Scorecard ingestion succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("success_payload", result.Success),
				)

				importCompletedMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ImportCompletedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create ImportCompleted message: %w", err)
				}

				return []*message.Message{importCompletedMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from IngestParsedScorecard service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleImportCompleted(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleImportCompleted",
		&roundevents.ImportCompletedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			completed := payload.(*roundevents.ImportCompletedPayloadV1)
			h.logger.InfoContext(ctx, "Received ImportCompleted event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("import_id", completed.ImportID),
				attr.String("guild_id", string(completed.GuildID)),
				attr.String("round_id", completed.RoundID.String()),
				attr.Int("matched_players", completed.MatchedPlayers),
				attr.Int("auto_added_players", completed.PlayersAutoAdded),
				attr.Int("scores_to_import", len(completed.Scores)),
			)

			// Delegate import score application to the service so it follows the exact
			// same path as manual score updates (UpdateParticipantScore -> publish ParticipantScoreUpdated).
			if len(completed.Scores) == 0 {
				h.logger.InfoContext(ctx, "Import completed with no scores to ingest",
					attr.CorrelationIDFromMsg(msg),
					attr.String("import_id", completed.ImportID),
				)
				return nil, nil
			}

			res, err := h.roundService.ApplyImportedScores(ctx, *completed)
			if err != nil {
				h.logger.ErrorContext(ctx, "ApplyImportedScores failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to apply imported scores: %w", err)
			}

			if res.Failure != nil {
				// Create and return an ImportFailed event to be published by the router
				failureMsg, errMsg := h.helpers.CreateResultMessage(msg, res.Failure, roundevents.ImportFailedV1)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Service returned the authoritative final participants snapshot.
			// Emit a single AllScoresSubmitted event using the authoritative snapshot
			// so downstream finalization runs exactly as in manual flow.
			appliedPayload, ok := res.Success.(*roundevents.ImportScoresAppliedPayloadV1)
			if !ok {
				return nil, fmt.Errorf("unexpected success payload type from ApplyImportedScores: %T", res.Success)
			}

			allSubmitted := roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        appliedPayload.GuildID,
				RoundID:        appliedPayload.RoundID,
				EventMessageID: appliedPayload.EventMessageID,
				RoundData: roundtypes.Round{
					ID:             appliedPayload.RoundID,
					EventMessageID: appliedPayload.EventMessageID,
					Participants:   appliedPayload.Participants,
				},
				Participants: appliedPayload.Participants,
			}

			allMsg, err := h.helpers.CreateResultMessage(msg, &allSubmitted, roundevents.RoundAllScoresSubmittedV1)
			if err != nil {
				return nil, fmt.Errorf("failed to create RoundAllScoresSubmitted message: %w", err)
			}

			return []*message.Message{allMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
