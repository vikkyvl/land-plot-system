package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
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

func (repository *PlotRepository) Create(requestContext context.Context, plot domain.Plot) error {
	points, err := encodePoints(plot.Points)
	if err != nil {
		return err
	}

	_, err = repository.pool.Exec(requestContext, `
		INSERT INTO plots (`+plotColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		plot.ID, plot.OwnerID, plot.Name, points, plot.AreaM2, plot.PerimeterM,
		plot.Version, plot.CreatedAt, plot.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(wrapInsertPlot, err)
	}

	return nil
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
	rows, err := repository.pool.Query(requestContext,
		`SELECT `+plotColumns+` FROM plots WHERE owner_id = $1 ORDER BY created_at, id`, ownerID)
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

func (repository *PlotRepository) Update(
	requestContext context.Context,
	plot domain.Plot,
	expectedVersion int64,
) error {
	points, err := encodePoints(plot.Points)
	if err != nil {
		return err
	}

	command, err := repository.pool.Exec(requestContext, `
		UPDATE plots
		SET name = $1, points = $2, area_m2 = $3, perimeter_m = $4, version = $5, updated_at = $6
		WHERE id = $7 AND owner_id = $8 AND version = $9`,
		plot.Name, points, plot.AreaM2, plot.PerimeterM, plot.Version, plot.UpdatedAt,
		plot.ID, plot.OwnerID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf(wrapUpdatePlotRow, err)
	}

	if command.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}

	return nil
}
