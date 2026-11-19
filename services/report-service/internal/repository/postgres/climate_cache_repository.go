package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

type ClimateCacheRepository struct {
	pool *pgxpool.Pool
}

func NewClimateCacheRepository(pool *pgxpool.Pool) *ClimateCacheRepository {
	return &ClimateCacheRepository{pool: pool}
}

func (repository *ClimateCacheRepository) Get(
	requestContext context.Context,
	key service.ClimateCacheKey,
	notBefore time.Time,
) (domain.Climate, bool, error) {
	var profile []byte

	err := repository.pool.QueryRow(requestContext, `
		SELECT profile FROM climate_cache
		WHERE cell_latitude_tenths = $1 AND cell_longitude_tenths = $2 AND first_year = $3 AND last_year = $4
			AND fetched_at >= $5`,
		key.Cell.LatitudeTenths, key.Cell.LongitudeTenths, key.Period.FirstYear, key.Period.LastYear, notBefore,
	).Scan(&profile)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Climate{}, false, nil
	}

	if err != nil {
		return domain.Climate{}, false, fmt.Errorf(wrapQueryClimateCache, err)
	}

	var record climateRecord
	if err := json.Unmarshal(profile, &record); err != nil {
		return domain.Climate{}, false, fmt.Errorf(wrapDecodeClimate, err)
	}

	return record.climate(), true, nil
}

func (repository *ClimateCacheRepository) Put(
	requestContext context.Context,
	key service.ClimateCacheKey,
	climate domain.Climate,
	fetchedAt time.Time,
) error {
	profile, err := json.Marshal(toClimateRecord(climate))
	if err != nil {
		return fmt.Errorf(wrapEncodeClimate, err)
	}

	_, err = repository.pool.Exec(
		requestContext,
		`
		INSERT INTO climate_cache
			(cell_latitude_tenths, cell_longitude_tenths, first_year, last_year, profile, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cell_latitude_tenths, cell_longitude_tenths, first_year, last_year)
		DO UPDATE SET profile = EXCLUDED.profile, fetched_at = EXCLUDED.fetched_at`,
		key.Cell.LatitudeTenths,
		key.Cell.LongitudeTenths,
		key.Period.FirstYear,
		key.Period.LastYear,
		profile,
		fetchedAt,
	)
	if err != nil {
		return fmt.Errorf(wrapUpsertClimateCache, err)
	}

	return nil
}
