// integration_tests/containers/postgres_container.go
package containers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupPostgresContainer starts a Postgres testcontainer and returns the container and connection string.
func SetupPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	dbName := "testdb"
	user := "testuser"
	password := "testpass"
	imageName := "postgres:16-alpine"

	// Programmatically get the absolute path to the init script
	initScriptPath, err := getAbsoluteInitScriptPath()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get init script path: %w", err)
	}

	log.Printf("Attempting to use init script from absolute path: %s", initScriptPath)

	pgContainer, err := postgres.Run(ctx,
		imageName,
		postgres.WithDatabase(dbName),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		postgres.WithInitScripts(initScriptPath),

		// --- Wait strategy using ForSQL ---
		testcontainers.WithWaitStrategy(
			wait.ForSQL("5432/tcp", "pgx",
				func(host string, port nat.Port) string {
					return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
						user,
						password,
						host,
						port.Port(),
						dbName,
					)
				},
			).WithStartupTimeout(45*time.Second),
		),
	)
	if err != nil {
		// Terminate the container if it was started but failed the wait strategy
		if pgContainer != nil {
			pgContainer.Terminate(ctx)
		}
		return nil, "", fmt.Errorf("failed to start postgres container: %w", err)
	}

	log.Println("Postgres container started and ready.")

	connStr, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		pgContainer.Terminate(ctx) // Terminate if getting conn string fails
		return nil, "", fmt.Errorf("failed to get postgres connection string: %w", err)
	}

	// --- Modify the connection string to ensure sslmode=disable  ---
	parsedURL, err := url.Parse(connStr)
	if err != nil {
		pgContainer.Terminate(ctx)
		return nil, "", fmt.Errorf("failed to parse connection string: %w", err)
	}

	query := parsedURL.Query()
	query.Set("sslmode", "disable")
	parsedURL.RawQuery = query.Encode()

	modifiedConnStr := parsedURL.String()

	log.Printf("Postgres connection string: %s", modifiedConnStr)

	return pgContainer, modifiedConnStr, nil
}

// getAbsoluteInitScriptPath determines the absolute path to the init-postgres.sh script
// relative to the location of this source file (postgres_container.go).
func getAbsoluteInitScriptPath() (string, error) {
	_, callerFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("could not get caller file path")
	}
	callerDir := filepath.Dir(callerFile)

	absolutePath := filepath.Join(callerDir, "..", "testutils", "init-postgres.sh")

	cleanedPath := filepath.Clean(absolutePath)

	return cleanedPath, nil
}
