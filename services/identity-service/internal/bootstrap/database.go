package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/tracing"
)

type migrator interface {
	Migrate(requestContext context.Context) error
}

func openDatabase(databaseURL string, timeout time.Duration) (*pgxpool.Pool, error) {
	timeoutContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf(wrapOpenDatabasePool, err)
	}

	tracing.TraceDatabase(poolConfig)

	pool, err := pgxpool.NewWithConfig(timeoutContext, poolConfig)
	if err != nil {
		return nil, fmt.Errorf(wrapOpenDatabasePool, err)
	}

	if err := pool.Ping(timeoutContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf(wrapPingDatabase, err)
	}

	return pool, nil
}

func migrate(repository migrator, timeout time.Duration) error {
	timeoutContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := repository.Migrate(timeoutContext); err != nil {
		return fmt.Errorf(wrapMigrateDatabase, err)
	}

	return nil
}
