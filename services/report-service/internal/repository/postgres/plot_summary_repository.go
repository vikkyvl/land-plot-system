package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type PlotSummaryRepository struct {
	pool *pgxpool.Pool
}

func NewPlotSummaryRepository(pool *pgxpool.Pool) *PlotSummaryRepository {
	return &PlotSummaryRepository{pool: pool}
}

func (repository *PlotSummaryRepository) Apply(processingContext context.Context, event domain.PlotEvent) error {
	transaction, err := repository.pool.Begin(processingContext)
	if err != nil {
		return fmt.Errorf(wrapBeginPlotEventTransaction, err)
	}
	defer func() { _ = transaction.Rollback(processingContext) }()

	firstDelivery, err := recordProcessedEvent(processingContext, transaction, event)
	if err != nil {
		return err
	}

	if firstDelivery {
		if err := upsertSummary(processingContext, transaction, event); err != nil {
			return err
		}
	}

	if err := transaction.Commit(processingContext); err != nil {
		return fmt.Errorf(wrapCommitPlotEvent, err)
	}

	return nil
}

func recordProcessedEvent(processingContext context.Context, transaction pgx.Tx, event domain.PlotEvent) (bool, error) {
	result, err := transaction.Exec(processingContext, `
		INSERT INTO processed_events (event_id, event_type, processed_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (event_id) DO NOTHING`,
		event.ID, string(event.Type),
	)
	if err != nil {
		return false, fmt.Errorf(wrapRecordProcessedEvent, err)
	}

	return result.RowsAffected() > 0, nil
}

func upsertSummary(processingContext context.Context, transaction pgx.Tx, event domain.PlotEvent) error {
	_, err := transaction.Exec(processingContext, `
		INSERT INTO plot_summaries (
			plot_id, owner_id, name, area_m2, version, occurred_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (plot_id) DO UPDATE SET
			owner_id = EXCLUDED.owner_id,
			name = EXCLUDED.name,
			area_m2 = EXCLUDED.area_m2,
			version = EXCLUDED.version,
			occurred_at = EXCLUDED.occurred_at,
			updated_at = NOW()
		WHERE EXCLUDED.version > plot_summaries.version`,
		event.PlotID, event.OwnerID, event.Name, event.AreaM2, event.Version, event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf(wrapUpsertPlotSummary, err)
	}

	return nil
}
