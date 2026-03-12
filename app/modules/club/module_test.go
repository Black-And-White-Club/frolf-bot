package club

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseModuleResourcesStopsQueueWhenRouterCloseFails(t *testing.T) {
	t.Parallel()

	routerErr := errors.New("router close timeout")
	router := &stubCloser{err: routerErr}
	queue := &stubQueueService{}

	err := closeModuleResources(testLogger(), router, queue)

	require.Error(t, err)
	assert.Equal(t, 1, router.closeCalls)
	assert.Equal(t, 1, queue.stopCalls)
	assert.ErrorContains(t, err, "error closing ClubRouter")
	assert.ErrorIs(t, err, routerErr)
}

func TestCloseModuleResourcesIgnoresQueueStopError(t *testing.T) {
	t.Parallel()

	queueErr := errors.New("queue stop failed")
	router := &stubCloser{}
	queue := &stubQueueService{stopErr: queueErr}

	err := closeModuleResources(testLogger(), router, queue)

	require.NoError(t, err)
	assert.Equal(t, 1, router.closeCalls)
	assert.Equal(t, 1, queue.stopCalls)
}

type stubCloser struct {
	closeCalls int
	err        error
}

func (s *stubCloser) Close() error {
	s.closeCalls++
	return s.err
}

type stubQueueService struct {
	stopCalls int
	stopCtx   context.Context
	stopErr   error
}

func (s *stubQueueService) ScheduleOpenExpiry(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (s *stubQueueService) ScheduleAcceptedExpiry(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (s *stubQueueService) CancelChallengeJobs(context.Context, uuid.UUID) error {
	return nil
}

func (s *stubQueueService) HealthCheck(context.Context) error {
	return nil
}

func (s *stubQueueService) Start(context.Context) error {
	return nil
}

func (s *stubQueueService) Stop(ctx context.Context) error {
	s.stopCalls++
	s.stopCtx = ctx
	return s.stopErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
