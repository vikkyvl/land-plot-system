package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

//go:embed schema.sql
var schema string

type PlotRepository struct {
	pool *pgxpool.Pool
}

func NewPlotRepository(pool *pgxpool.Pool) *PlotRepository {
	return &PlotRepository{pool: pool}
}

func (repository *PlotRepository) Migrate(requestContext context.Context) error {
	if _, err := repository.pool.Exec(requestContext, schema); err != nil {
		return fmt.Errorf(wrapApplyPlotSchema, err)
	}

	return nil
}

func (repository *PlotRepository) InTransaction(
	requestContext context.Context,
	work func(store service.PlotStore) error,
) error {
	transaction, err := repository.pool.Begin(requestContext)
	if err != nil {
		return fmt.Errorf(wrapBeginPlotTransaction, err)
	}
	defer func() { _ = transaction.Rollback(requestContext) }()

	if err := work(&plotStore{transaction: transaction}); err != nil {
		return err
	}

	if err := transaction.Commit(requestContext); err != nil {
		return fmt.Errorf(wrapCommitPlotTransaction, err)
	}

	return nil
}

func (repository *PlotRepository) ExpiredSagaIDs(requestContext context.Context, limit int) ([]string, error) {
	rows, err := repository.pool.Query(requestContext, `
		SELECT id FROM plot_sagas
		WHERE status = $1 AND deadline_at < now()
		ORDER BY deadline_at
		LIMIT $2`,
		domain.SagaStarted, limit,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapQueryExpiredSagas, err)
	}

	sagaIDs, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf(wrapQueryExpiredSagas, err)
	}

	return sagaIDs, nil
}

func (repository *PlotRepository) GetByID(requestContext context.Context, id string) (domain.Plot, error) {
	row := repository.pool.QueryRow(requestContext, `SELECT `+plotColumns+` FROM plots WHERE id = $1`, id)

	plot, err := scanPlot(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plot{}, domain.ErrPlotNotFound
	}

	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapQueryPlotByID, err)
	}

	return plot, nil
}

func (repository *PlotRepository) ListByOwner(requestContext context.Context, ownerID string) ([]domain.Plot, error) {
	rows, err := repository.pool.Query(requestContext, `
		SELECT `+plotColumns+` FROM plots
		WHERE owner_id = $1 AND status <> $2
		ORDER BY created_at, id`,
		ownerID, domain.StatusCancelled,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapQueryPlotsByOwner, err)
	}
	defer rows.Close()

	plots := make([]domain.Plot, 0)

	for rows.Next() {
		plot, err := scanPlot(rows)
		if err != nil {
			return nil, fmt.Errorf(wrapScanOwnerPlot, err)
		}

		plots = append(plots, plot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(wrapIterateOwnerPlots, err)
	}

	return plots, nil
}
