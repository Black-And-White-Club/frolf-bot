package roundhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

const defaultPaginationSnapshotTTL = 30 * 24 * time.Hour

func (h *RoundHandlers) HandlePaginationSnapshotUpsertRequested(
	ctx context.Context,
	payload *roundevents.PaginationSnapshotUpsertRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("pagination snapshot upsert payload is nil")
	}

	messageID := strings.TrimSpace(payload.MessageID)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		requestID = messageID
	}

	if h.paginationSnapshotStore == nil {
		return upsertResult(requestID, messageID, false, "pagination snapshot store is not configured"), nil
	}
	if messageID == "" {
		return upsertResult(requestID, messageID, false, "message id is empty"), nil
	}
	if len(payload.Snapshot) == 0 {
		return upsertResult(requestID, messageID, false, "snapshot is empty"), nil
	}

	ttl := defaultPaginationSnapshotTTL
	if payload.TTLSeconds > 0 {
		ttl = time.Duration(payload.TTLSeconds) * time.Second
	}

	expiresAt := time.Now().UTC().Add(ttl)
	if err := h.paginationSnapshotStore.Upsert(ctx, messageID, payload.Snapshot, expiresAt); err != nil {
		h.logger.WarnContext(ctx, "Failed to persist pagination snapshot",
			attr.String("message_id", messageID),
			attr.Error(err),
		)
		return upsertResult(requestID, messageID, false, err.Error()), nil
	}

	return upsertResult(requestID, messageID, true, ""), nil
}

func (h *RoundHandlers) HandlePaginationSnapshotGetRequested(
	ctx context.Context,
	payload *roundevents.PaginationSnapshotGetRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("pagination snapshot get payload is nil")
	}

	messageID := strings.TrimSpace(payload.MessageID)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		requestID = messageID
	}

	if h.paginationSnapshotStore == nil {
		return getResult(requestID, messageID, false, nil, "pagination snapshot store is not configured"), nil
	}
	if messageID == "" {
		return getResult(requestID, messageID, false, nil, "message id is empty"), nil
	}

	snapshot, found, err := h.paginationSnapshotStore.Get(ctx, messageID)
	if err != nil {
		h.logger.WarnContext(ctx, "Failed to retrieve pagination snapshot",
			attr.String("message_id", messageID),
			attr.Error(err),
		)
		return getResult(requestID, messageID, false, nil, err.Error()), nil
	}
	if !found {
		return getResult(requestID, messageID, false, nil, ""), nil
	}

	return getResult(requestID, messageID, true, snapshot, ""), nil
}

func (h *RoundHandlers) HandlePaginationSnapshotDeleteRequested(
	ctx context.Context,
	payload *roundevents.PaginationSnapshotDeleteRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("pagination snapshot delete payload is nil")
	}

	messageID := strings.TrimSpace(payload.MessageID)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		requestID = messageID
	}

	if h.paginationSnapshotStore == nil {
		return deleteResult(requestID, messageID, false, "pagination snapshot store is not configured"), nil
	}
	if messageID == "" {
		return deleteResult(requestID, messageID, false, "message id is empty"), nil
	}

	if err := h.paginationSnapshotStore.Delete(ctx, messageID); err != nil {
		h.logger.WarnContext(ctx, "Failed to delete pagination snapshot",
			attr.String("message_id", messageID),
			attr.Error(err),
		)
		return deleteResult(requestID, messageID, false, err.Error()), nil
	}

	return deleteResult(requestID, messageID, true, ""), nil
}

// HandleRoundDeleted removes any pagination snapshot linked to a deleted round message.
func (h *RoundHandlers) HandleRoundDeleted(
	ctx context.Context,
	payload *roundevents.RoundDeletedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil || h.paginationSnapshotStore == nil {
		return []handlerwrapper.Result{}, nil
	}

	messageID := strings.TrimSpace(payload.EventMessageID)
	if messageID == "" {
		return []handlerwrapper.Result{}, nil
	}

	if err := h.paginationSnapshotStore.Delete(ctx, messageID); err != nil {
		h.logger.WarnContext(ctx, "Failed to clean pagination snapshot for deleted round",
			attr.String("message_id", messageID),
			attr.Error(err),
		)
	}

	return []handlerwrapper.Result{}, nil
}

// HandleRoundCompleted removes any pagination snapshot linked to a completed round message.
func (h *RoundHandlers) HandleRoundCompleted(
	ctx context.Context,
	payload *roundevents.RoundCompletedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil || h.paginationSnapshotStore == nil {
		return []handlerwrapper.Result{}, nil
	}

	messageID := strings.TrimSpace(payload.RoundData.EventMessageID)
	if messageID == "" {
		if metadataMessageID, ok := ctx.Value("discord_message_id").(string); ok {
			messageID = strings.TrimSpace(metadataMessageID)
		}
	}

	if messageID == "" {
		return []handlerwrapper.Result{}, nil
	}

	if err := h.paginationSnapshotStore.Delete(ctx, messageID); err != nil {
		h.logger.WarnContext(ctx, "Failed to clean pagination snapshot for completed round",
			attr.String("message_id", messageID),
			attr.Error(err),
		)
	}

	return []handlerwrapper.Result{}, nil
}

func upsertResult(requestID, messageID string, success bool, errMsg string) []handlerwrapper.Result {
	return []handlerwrapper.Result{
		{
			Topic: roundevents.PaginationSnapshotUpsertResultV1,
			Payload: &roundevents.PaginationSnapshotUpsertResultPayloadV1{
				RequestID: requestID,
				MessageID: messageID,
				Success:   success,
				Error:     errMsg,
			},
		},
	}
}

func getResult(requestID, messageID string, found bool, snapshot json.RawMessage, errMsg string) []handlerwrapper.Result {
	payload := &roundevents.PaginationSnapshotGetResultPayloadV1{
		RequestID: requestID,
		MessageID: messageID,
		Found:     found,
		Error:     errMsg,
	}
	if len(snapshot) > 0 {
		payload.Snapshot = append(json.RawMessage(nil), snapshot...)
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.PaginationSnapshotGetResultV1,
			Payload: payload,
		},
	}
}

func deleteResult(requestID, messageID string, success bool, errMsg string) []handlerwrapper.Result {
	return []handlerwrapper.Result{
		{
			Topic: roundevents.PaginationSnapshotDeleteResultV1,
			Payload: &roundevents.PaginationSnapshotDeleteResultPayloadV1{
				RequestID: requestID,
				MessageID: messageID,
				Success:   success,
				Error:     errMsg,
			},
		},
	}
}

// PaginationSnapshotStore is the persistence contract used by snapshot event handlers.
type PaginationSnapshotStore interface {
	Upsert(ctx context.Context, messageID string, snapshot json.RawMessage, expiresAt time.Time) error
	Get(ctx context.Context, messageID string) (json.RawMessage, bool, error)
	Delete(ctx context.Context, messageID string) error
}

// DBPaginationSnapshotStore adapts the round repository implementation to handler expectations.
type DBPaginationSnapshotStore struct {
	repo rounddb.PaginationSnapshotStore
}

// NewDBPaginationSnapshotStore constructs a DB-backed pagination snapshot store adapter.
func NewDBPaginationSnapshotStore(repo rounddb.PaginationSnapshotStore) (*DBPaginationSnapshotStore, error) {
	if repo == nil {
		return nil, fmt.Errorf("pagination snapshot repository is nil")
	}
	return &DBPaginationSnapshotStore{repo: repo}, nil
}

func (s *DBPaginationSnapshotStore) Upsert(ctx context.Context, messageID string, snapshot json.RawMessage, expiresAt time.Time) error {
	return s.repo.Upsert(ctx, nil, messageID, snapshot, expiresAt)
}

func (s *DBPaginationSnapshotStore) Get(ctx context.Context, messageID string) (json.RawMessage, bool, error) {
	return s.repo.Get(ctx, nil, messageID)
}

func (s *DBPaginationSnapshotStore) Delete(ctx context.Context, messageID string) error {
	return s.repo.Delete(ctx, nil, messageID)
}
