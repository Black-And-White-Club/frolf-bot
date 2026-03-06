package roundhandlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

type fakePaginationSnapshotStore struct {
	upsertFn func(ctx context.Context, messageID string, snapshot json.RawMessage, expiresAt time.Time) error
	getFn    func(ctx context.Context, messageID string) (json.RawMessage, bool, error)
	deleteFn func(ctx context.Context, messageID string) error
}

func (f *fakePaginationSnapshotStore) Upsert(ctx context.Context, messageID string, snapshot json.RawMessage, expiresAt time.Time) error {
	if f.upsertFn != nil {
		return f.upsertFn(ctx, messageID, snapshot, expiresAt)
	}
	return nil
}

func (f *fakePaginationSnapshotStore) Get(ctx context.Context, messageID string) (json.RawMessage, bool, error) {
	if f.getFn != nil {
		return f.getFn(ctx, messageID)
	}
	return nil, false, nil
}

func (f *fakePaginationSnapshotStore) Delete(ctx context.Context, messageID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, messageID)
	}
	return nil
}

func TestHandlePaginationSnapshotUpsertRequested(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			h := &RoundHandlers{
				logger: loggerfrolfbot.NoOpLogger,
				paginationSnapshotStore: &fakePaginationSnapshotStore{
					upsertFn: func(ctx context.Context, messageID string, snapshot json.RawMessage, expiresAt time.Time) error {
						if messageID != "msg-1" {
							t.Fatalf("messageID = %q, want msg-1", messageID)
						}
						if len(snapshot) == 0 {
							t.Fatal("snapshot was empty")
						}
						if expiresAt.Before(time.Now().UTC()) {
							t.Fatal("expiresAt should be in the future")
						}
						return nil
					},
				},
			}

			results, err := h.HandlePaginationSnapshotUpsertRequested(context.Background(), &roundevents.PaginationSnapshotUpsertRequestedPayloadV1{
				RequestID: "req-1",
				MessageID: "msg-1",
				Snapshot:  json.RawMessage(`{"kind":"lines"}`),
			})
			if err != nil {
				t.Fatalf("HandlePaginationSnapshotUpsertRequested() error = %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("len(results) = %d, want 1", len(results))
			}
			if results[0].Topic != roundevents.PaginationSnapshotUpsertResultV1 {
				t.Fatalf("topic = %q, want %q", results[0].Topic, roundevents.PaginationSnapshotUpsertResultV1)
			}

			payload, ok := results[0].Payload.(*roundevents.PaginationSnapshotUpsertResultPayloadV1)
			if !ok {
				t.Fatalf("payload type = %T", results[0].Payload)
			}
			if !payload.Success {
				t.Fatalf("payload.Success = false, want true; error=%q", payload.Error)
			}
		})
	}
}

func TestHandlePaginationSnapshotGetRequested(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			h := &RoundHandlers{
				logger: loggerfrolfbot.NoOpLogger,
				paginationSnapshotStore: &fakePaginationSnapshotStore{
					getFn: func(ctx context.Context, messageID string) (json.RawMessage, bool, error) {
						if messageID != "msg-2" {
							t.Fatalf("messageID = %q, want msg-2", messageID)
						}
						return json.RawMessage(`{"kind":"fields"}`), true, nil
					},
				},
			}

			results, err := h.HandlePaginationSnapshotGetRequested(context.Background(), &roundevents.PaginationSnapshotGetRequestedPayloadV1{
				RequestID: "req-2",
				MessageID: "msg-2",
			})
			if err != nil {
				t.Fatalf("HandlePaginationSnapshotGetRequested() error = %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("len(results) = %d, want 1", len(results))
			}
			if results[0].Topic != roundevents.PaginationSnapshotGetResultV1 {
				t.Fatalf("topic = %q, want %q", results[0].Topic, roundevents.PaginationSnapshotGetResultV1)
			}

			payload, ok := results[0].Payload.(*roundevents.PaginationSnapshotGetResultPayloadV1)
			if !ok {
				t.Fatalf("payload type = %T", results[0].Payload)
			}
			if !payload.Found {
				t.Fatal("payload.Found = false, want true")
			}
			if string(payload.Snapshot) != `{"kind":"fields"}` {
				t.Fatalf("payload.Snapshot = %s", string(payload.Snapshot))
			}
		})
	}
}

func TestHandlePaginationSnapshotDeleteRequested(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			h := &RoundHandlers{
				logger: loggerfrolfbot.NoOpLogger,
				paginationSnapshotStore: &fakePaginationSnapshotStore{
					deleteFn: func(ctx context.Context, messageID string) error {
						if messageID != "msg-3" {
							t.Fatalf("messageID = %q, want msg-3", messageID)
						}
						return nil
					},
				},
			}

			results, err := h.HandlePaginationSnapshotDeleteRequested(context.Background(), &roundevents.PaginationSnapshotDeleteRequestedPayloadV1{
				RequestID: "req-3",
				MessageID: "msg-3",
			})
			if err != nil {
				t.Fatalf("HandlePaginationSnapshotDeleteRequested() error = %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("len(results) = %d, want 1", len(results))
			}
			if results[0].Topic != roundevents.PaginationSnapshotDeleteResultV1 {
				t.Fatalf("topic = %q, want %q", results[0].Topic, roundevents.PaginationSnapshotDeleteResultV1)
			}

			payload, ok := results[0].Payload.(*roundevents.PaginationSnapshotDeleteResultPayloadV1)
			if !ok {
				t.Fatalf("payload type = %T", results[0].Payload)
			}
			if !payload.Success {
				t.Fatalf("payload.Success = false, want true; error=%q", payload.Error)
			}
		})
	}
}

func TestRoundLifecycleSnapshotCleanup(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			deleted := make([]string, 0, 2)
			h := &RoundHandlers{
				logger: loggerfrolfbot.NoOpLogger,
				paginationSnapshotStore: &fakePaginationSnapshotStore{
					deleteFn: func(ctx context.Context, messageID string) error {
						deleted = append(deleted, messageID)
						return nil
					},
				},
			}

			_, err := h.HandleRoundDeleted(context.Background(), &roundevents.RoundDeletedPayloadV1{EventMessageID: "msg-del"})
			if err != nil {
				t.Fatalf("HandleRoundDeleted() error = %v", err)
			}

			_, err = h.HandleRoundCompleted(context.Background(), &roundevents.RoundCompletedPayloadV1{RoundData: roundtypes.Round{EventMessageID: "msg-complete"}})
			if err != nil {
				t.Fatalf("HandleRoundCompleted() error = %v", err)
			}

			if len(deleted) != 2 {
				t.Fatalf("deleted len = %d, want 2", len(deleted))
			}
			if deleted[0] != "msg-del" || deleted[1] != "msg-complete" {
				t.Fatalf("deleted = %#v", deleted)
			}
		})
	}
}
