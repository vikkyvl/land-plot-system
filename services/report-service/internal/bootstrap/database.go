package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
)

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

	if err := postgres.Migrate(timeoutContext, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf(wrapMigrateDatabase, err)
	}

	return pool, nil
}
