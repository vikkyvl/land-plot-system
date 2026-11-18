package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const reportColumns = `id, owner_id, plot_id, plot_name, plot_version, area_m2,
	area_ares, area_hectares, perimeter_m, created_at`

const storedReportColumns = reportColumns + `, plot_deleted_at`

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

func (repository *ReportRepository) Create(requestContext context.Context, report domain.Report) error {
	_, err := repository.pool.Exec(requestContext, `
		INSERT INTO reports (`+reportColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		report.ID, report.OwnerID, report.PlotID, report.PlotName, report.PlotVersion,
		report.AreaM2, report.AreaAres, report.AreaHectares, report.PerimeterM, report.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf(wrapInsertReport, err)
	}

	return nil
}

func (repository *ReportRepository) GetByID(requestContext context.Context, id string) (domain.Report, error) {
	row := repository.pool.QueryRow(requestContext, `SELECT `+storedReportColumns+` FROM reports WHERE id = $1`, id)

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
		SELECT `+storedReportColumns+` FROM reports
		WHERE owner_id = $1 AND ($2::uuid IS NULL OR plot_id = $2::uuid)
		ORDER BY created_at DESC, id DESC
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
	var report domain.Report

	err := row.Scan(
		&report.ID, &report.OwnerID, &report.PlotID, &report.PlotName, &report.PlotVersion,
		&report.AreaM2, &report.AreaAres, &report.AreaHectares, &report.PerimeterM, &report.CreatedAt,
		&report.PlotDeletedAt,
	)

	return report, err
}
