CREATE TABLE IF NOT EXISTS events (
    global_seq  BIGSERIAL   PRIMARY KEY,
    stream_id   TEXT        NOT NULL,
    version     INTEGER     NOT NULL,
    event_type  TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    UNIQUE (stream_id, version)
);
