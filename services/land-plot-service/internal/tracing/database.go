package tracing

import (
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TraceDatabase(poolConfig *pgxpool.Config) {
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
}
