BEGIN;

SELECT pg_advisory_xact_lock(82003);

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    plot_id UUID NOT NULL,
    plot_name TEXT NOT NULL,
    plot_version BIGINT NOT NULL,
    area_m2 DOUBLE PRECISION NOT NULL,
    area_ares DOUBLE PRECISION NOT NULL,
    area_hectares DOUBLE PRECISION NOT NULL,
    perimeter_m DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS reports_owner_id_idx ON reports (owner_id);

CREATE TABLE IF NOT EXISTS plot_summaries (
    plot_id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    name TEXT NOT NULL,
    area_m2 DOUBLE PRECISION NOT NULL CHECK (area_m2 > 0),
    version BIGINT NOT NULL CHECK (version > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS plot_summaries_owner_id_idx ON plot_summaries (owner_id);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL
);

COMMIT;
