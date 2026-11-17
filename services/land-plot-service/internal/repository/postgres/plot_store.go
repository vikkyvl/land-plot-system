package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

type plotStore struct {
	transaction pgx.Tx
}

func (store *plotStore) CreatePlot(requestContext context.Context, plot domain.Plot) error {
	points, err := encodePoints(plot.Points)
	if err != nil {
		return err
	}

	_, err = store.transaction.Exec(requestContext, `
		INSERT INTO plots (`+plotColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		plot.ID, plot.OwnerID, plot.Name, points, plot.AreaM2, plot.PerimeterM,
		plot.Version, plot.Status, plot.CreatedAt, plot.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(wrapInsertPlot, err)
	}

	return nil
}

func (store *plotStore) UpdatePlot(requestContext context.Context, plot domain.Plot, expectedVersion int64) error {
	points, err := encodePoints(plot.Points)
	if err != nil {
		return err
	}

	command, err := store.transaction.Exec(requestContext, `
		UPDATE plots
		SET name = $1, points = $2, area_m2 = $3, perimeter_m = $4, version = $5, status = $6, updated_at = $7
		WHERE id = $8 AND owner_id = $9 AND version = $10`,
		plot.Name, points, plot.AreaM2, plot.PerimeterM, plot.Version, plot.Status, plot.UpdatedAt,
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

func (store *plotStore) LockPlot(requestContext context.Context, id string) (domain.Plot, error) {
	row := store.transaction.QueryRow(requestContext,
		`SELECT `+plotColumns+` FROM plots WHERE id = $1 FOR UPDATE`, id)

	plot, err := scanPlot(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plot{}, domain.ErrPlotNotFound
	}

	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapLockPlot, err)
	}

	return plot, nil
}

func (store *plotStore) StartSaga(requestContext context.Context, saga domain.Saga, timeout time.Duration) error {
	previousState, err := encodePreviousState(saga.PreviousState)
	if err != nil {
		return err
	}

	_, err = store.transaction.Exec(requestContext, `
		INSERT INTO plot_sagas (id, plot_id, kind, status, previous_state, new_version, deadline_at)
		VALUES ($1, $2, $3, $4, $5, $6, now() + make_interval(secs => $7))`,
		saga.ID, saga.PlotID, saga.Kind, saga.Status, previousState, saga.NewVersion, timeout.Seconds(),
	)
	if err != nil {
		return fmt.Errorf(wrapInsertSaga, err)
	}

	return nil
}

func (store *plotStore) FinishSaga(
	requestContext context.Context,
	finish service.SagaFinish,
) (domain.Saga, bool, error) {
	row := store.transaction.QueryRow(requestContext, `
		UPDATE plot_sagas
		SET status = $2, finished_at = now(), reason = NULLIF($3, '')
		WHERE id = $1 AND status = $4
		RETURNING `+sagaColumns,
		finish.SagaID, finish.Status, finish.Reason, domain.SagaStarted,
	)

	saga, err := scanSaga(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Saga{}, false, nil
	}

	if err != nil {
		return domain.Saga{}, false, fmt.Errorf(wrapFinishSaga, err)
	}

	return saga, true, nil
}

func (store *plotStore) AppendEvent(requestContext context.Context, event domain.PlotEvent) error {
	payload, err := encodePlotEvent(event)
	if err != nil {
		return err
	}

	_, err = store.transaction.Exec(requestContext, `
		INSERT INTO outbox_events (id, routing_key, payload) VALUES ($1, $2, $3)`,
		event.ID, string(event.Type), payload,
	)
	if err != nil {
		return fmt.Errorf(wrapInsertOutboxEvent, err)
	}

	return nil
}
