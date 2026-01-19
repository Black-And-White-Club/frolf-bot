package containers

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	dbName     = "testdb"
	dbUser     = "testuser"
	dbPassword = "testpass"
)

// SetupPostgresContainer starts a PostgreSQL testcontainer with resource limits
// and returns the container instance and the database connection URL.
func SetupPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	log.Println("Starting PostgreSQL container with resource limits...")

	// Get the path to the init script
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile))) // Go up 3 levels from containers dir
	initScriptPath := filepath.Join(projectRoot, "integration_tests", "testutils", "init-postgres.sh")

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.WithInitScripts(initScriptPath),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(60*time.Second),
		),
		// Add resource limits using HostConfigModifier
		testcontainers.WithConfigModifier(func(config *container.Config) {
			// Set PostgreSQL specific environment variables for better resource management
			if config.Env == nil {
				config.Env = make([]string, 0)
			}
			config.Env = append(config.Env,
				"POSTGRES_SHARED_BUFFERS=64MB",
				"POSTGRES_EFFECTIVE_CACHE_SIZE=128MB",
				"POSTGRES_WORK_MEM=4MB",
				"POSTGRES_MAINTENANCE_WORK_MEM=16MB",
			)
		}),
		// Use command args to set max_connections for integration tests
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Cmd = []string{
				"postgres",
				"-c", "max_connections=300", // Increased from default 100 to handle many test connections
			}
		}),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Resources = container.Resources{
				Memory:   512 * 1024 * 1024, // 512MB limit
				NanoCPUs: 1000000000,        // 1.0 CPU (100% of one core)
			}
			// Use tmpfs for data directory to speed up I/O
			hostConfig.Tmpfs = map[string]string{
				"/var/lib/postgresql/data": "rw",
			}
		}),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminateErr := pgContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate PostgreSQL container: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get PostgreSQL connection string: %w", err)
	}
	log.Printf("PostgreSQL container started with resource limits and init script. URL: %s", pgURL)

	// Optionally stream container logs to stdout for debugging hangs. Enable by
	// setting the environment variable STREAM_TESTCONTAINER_LOGS=1
	if os.Getenv("STREAM_TESTCONTAINER_LOGS") == "1" {
		go func() {
			rc, err := pgContainer.Logs(ctx)
			if err != nil {
				log.Printf("Failed to attach to Postgres container logs: %v", err)
				return
			}
			defer rc.Close()
			if _, err := io.Copy(os.Stdout, rc); err != nil {
				log.Printf("Error streaming Postgres container logs: %v", err)
			}
		}()
	}
	return pgContainer, pgURL, nil
}

// SetupPostgresContainerWithoutInitScript provides a container without the init script
// Use this if you don't need the UUID extension or other custom setup
func SetupPostgresContainerWithoutInitScript(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	log.Println("Starting PostgreSQL container without init script...")

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(60*time.Second),
		),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Resources = container.Resources{
				Memory:   512 * 1024 * 1024, // 512MB limit
				NanoCPUs: 1000000000,        // 1.0 CPU (100% of one core)
			}
			// Use tmpfs for data directory to speed up I/O
			hostConfig.Tmpfs = map[string]string{
				"/var/lib/postgresql/data": "rw",
			}
		}),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminateErr := pgContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate PostgreSQL container: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get PostgreSQL connection string: %w", err)
	}

	log.Printf("PostgreSQL container started without init script. URL: %s", pgURL)
	return pgContainer, pgURL, nil
}
