package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const reportColumns = `id, owner_id, plot_id, plot_name, plot_version, area_m2,
	area_ares, area_hectares, perimeter_m, created_at, accuracy, terrain, climate, boundary`

const selectReports = `SELECT report.id, report.owner_id, report.plot_id, report.plot_name, report.plot_version,
	report.area_m2, report.area_ares, report.area_hectares, report.perimeter_m, report.created_at,
	report.accuracy, report.terrain, report.climate, report.boundary, report.plot_deleted_at,
	latest.id, latest.created_at
	FROM reports report
	LEFT JOIN LATERAL (
		SELECT newest.id, newest.created_at FROM reports newest
		WHERE newest.owner_id = report.owner_id AND newest.plot_id = report.plot_id
		ORDER BY newest.created_at DESC, newest.id DESC
		LIMIT 1
	) latest ON true`

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

func (repository *ReportRepository) Create(requestContext context.Context, report domain.Report) error {
	snapshot, err := encodeSnapshot(report)
	if err != nil {
		return err
	}

	_, err = repository.pool.Exec(requestContext, `
		INSERT INTO reports (`+reportColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		report.ID, report.OwnerID, report.PlotID, report.PlotName, report.PlotVersion,
		report.AreaM2, report.AreaAres, report.AreaHectares, report.PerimeterM, report.CreatedAt,
		snapshot.accuracy, snapshot.terrain, snapshot.climate, snapshot.boundary,
	)
	if err != nil {
		return fmt.Errorf(wrapInsertReport, err)
	}

	return nil
}

func (repository *ReportRepository) GetByID(requestContext context.Context, id string) (domain.Report, error) {
	row := repository.pool.QueryRow(requestContext, selectReports+` WHERE report.id = $1`, id)

	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Report{}, domain.ErrReportNotFound
	}

	if err != nil {
		return domain.Report{}, fmt.Errorf(wrapQueryReportByID, err)
	}

	return report, nil
}

func (repository *ReportRepository) List(
	requestContext context.Context,
	filter domain.ReportFilter,
) ([]domain.Report, error) {
	var plotID *string
	if filter.PlotID != "" {
		plotID = &filter.PlotID
	}

	rows, err := repository.pool.Query(requestContext, `
		`+selectReports+`
		WHERE report.owner_id = $1 AND ($2::uuid IS NULL OR report.plot_id = $2::uuid)
		ORDER BY report.created_at DESC, report.id DESC
		LIMIT $3`,
		filter.OwnerID, plotID, filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapQueryReports, err)
	}

	reports, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Report, error) {
		return scanReport(row)
	})
	if err != nil {
		return nil, fmt.Errorf(wrapQueryReports, err)
	}

	return reports, nil
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanReport(row rowScanner) (domain.Report, error) {
	var (
		report          domain.Report
		snapshot        snapshotColumns
		latestID        *string
		latestCreatedAt *time.Time
	)

	err := row.Scan(
		&report.ID, &report.OwnerID, &report.PlotID, &report.PlotName, &report.PlotVersion,
		&report.AreaM2, &report.AreaAres, &report.AreaHectares, &report.PerimeterM, &report.CreatedAt,
		&snapshot.accuracy, &snapshot.terrain, &snapshot.climate, &snapshot.boundary, &report.PlotDeletedAt,
		&latestID, &latestCreatedAt,
	)
	if err != nil {
		return domain.Report{}, err
	}

	if latestID != nil && latestCreatedAt != nil {
		report.Latest = &domain.LatestReport{ID: *latestID, CreatedAt: *latestCreatedAt}
	}

	return snapshot.decodeInto(report)
}
