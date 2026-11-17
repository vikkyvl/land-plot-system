package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type migrator interface {
	Migrate(requestContext context.Context) error
}

func openDatabase(databaseURL string, timeout time.Duration) (*pgxpool.Pool, error) {
	timeoutContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pool, err := pgxpool.New(timeoutContext, databaseURL)
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
