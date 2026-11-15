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

ALTER TABLE plot_summaries ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    position BIGSERIAL NOT NULL,
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ NULL,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL
);

ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ NULL;

DROP INDEX IF EXISTS outbox_events_unpublished_idx;

CREATE INDEX IF NOT EXISTS outbox_events_pending_idx
    ON outbox_events (position) WHERE published_at IS NULL AND failed_at IS NULL;

COMMIT;
