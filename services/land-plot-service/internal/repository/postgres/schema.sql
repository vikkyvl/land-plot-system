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

-- Plots stored before the saga existed were already delivered, so they start as confirmed.
ALTER TABLE plots ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'CONFIRMED'
    CHECK (status IN ('PENDING', 'CONFIRMED', 'CANCELLED'));

CREATE TABLE IF NOT EXISTS plot_sagas (
    id UUID PRIMARY KEY,
    plot_id UUID NOT NULL REFERENCES plots (id),
    kind TEXT NOT NULL CHECK (kind IN ('CREATE', 'UPDATE')),
    status TEXT NOT NULL CHECK (status IN ('STARTED', 'COMPLETED', 'COMPENSATED')),
    previous_state JSONB NULL,
    new_version BIGINT NOT NULL,
    deadline_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ NULL,
    reason TEXT NULL
);

CREATE INDEX IF NOT EXISTS plot_sagas_started_idx
    ON plot_sagas (deadline_at) WHERE status = 'STARTED';

CREATE UNIQUE INDEX IF NOT EXISTS plot_sagas_one_started_per_plot_idx
    ON plot_sagas (plot_id) WHERE status = 'STARTED';

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
