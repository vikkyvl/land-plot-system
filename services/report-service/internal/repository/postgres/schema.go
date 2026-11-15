package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

func Migrate(requestContext context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(requestContext, schema); err != nil {
		return fmt.Errorf(wrapApplyReportSchema, err)
	}

	return nil
}
