BEGIN;

-- Serializes schema initialization when several scaled instances start together.
SELECT pg_advisory_xact_lock(82002);

CREATE TABLE IF NOT EXISTS plots (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    name TEXT NOT NULL,
    points JSONB NOT NULL,
    area_m2 DOUBLE PRECISION NOT NULL,
    perimeter_m DOUBLE PRECISION NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS plots_owner_id_idx ON plots (owner_id);

COMMIT;
