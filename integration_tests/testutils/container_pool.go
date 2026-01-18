package testutils

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/containers"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// ContainerPool manages shared container instances across test runs
type ContainerPool struct {
	mu            sync.Mutex
	pgContainer   *postgres.PostgresContainer
	natsContainer *nats.NATSContainer
	pgConnStr     string
	natsURL       string
	initialized   bool
	refCount      int
}

var globalPool = &ContainerPool{}

// Acquire returns existing containers or starts new ones
// Increments refCount on each call
func (p *ContainerPool) Acquire(ctx context.Context) (*postgres.PostgresContainer, testcontainers.Container, string, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		p.refCount++
		var natsGeneric testcontainers.Container
		if p.natsContainer != nil {
			natsGeneric = p.natsContainer.Container
		}
		return p.pgContainer, natsGeneric, p.pgConnStr, p.natsURL, nil
	}

	// First call - start containers
	pgContainer, pgConnStr, err := containers.SetupPostgresContainer(ctx)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to setup postgres container: %w", err)
	}

	natsContainer, natsURL, err := containers.SetupNatsContainer(ctx)
	if err != nil {
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		return nil, nil, "", "", fmt.Errorf("failed to setup nats container: %w", err)
	}

	p.pgContainer = pgContainer
	p.natsContainer = natsContainer
	p.pgConnStr = pgConnStr
	p.natsURL = natsURL
	p.initialized = true
	p.refCount = 1

	var natsGeneric testcontainers.Container
	if natsContainer != nil {
		natsGeneric = natsContainer.Container
	}

	return pgContainer, natsGeneric, pgConnStr, natsURL, nil
}

// Release decrements the reference count
func (p *ContainerPool) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.refCount > 0 {
		p.refCount--
	}
}

// Shutdown terminates containers when refCount reaches 0
func (p *ContainerPool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil
	}

	// Always shutdown regardless of refCount in cleanup
	var errs []error

	if p.natsContainer != nil {
		if err := p.natsContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate nats container: %w", err))
		}
		p.natsContainer = nil
	}

	if p.pgContainer != nil {
		if err := p.pgContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate postgres container: %w", err))
		}
		p.pgContainer = nil
	}

	p.initialized = false
	p.refCount = 0
	p.pgConnStr = ""
	p.natsURL = ""

	if len(errs) > 0 {
		return fmt.Errorf("container shutdown errors: %v", errs)
	}

	return nil
}

// ShutdownContainerPool is a public function to shutdown the global container pool
func ShutdownContainerPool(ctx context.Context) error {
	return globalPool.Shutdown(ctx)
}
