//go:build integration

package postgres

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDatabaseURLKey = "TEST_DATABASE_URL"

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv(testDatabaseURLKey)
	if databaseURL == "" {
		t.Skip(testDatabaseURLKey + " is not set")
	}

	schemaName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	runSchemaStatement(t, databaseURL, "CREATE SCHEMA "+schemaName)

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}

	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		runSchemaStatement(t, databaseURL, "DROP SCHEMA "+schemaName+" CASCADE")
	})

	if err := NewPlotRepository(pool).Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	return pool
}

func runSchemaStatement(t *testing.T, databaseURL, statement string) {
	t.Helper()

	connection, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = connection.Close(context.Background()) }()

	if _, err := connection.Exec(context.Background(), statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}
