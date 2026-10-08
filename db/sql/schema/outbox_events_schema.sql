CREATE TABLE IF NOT EXISTS outbox_events
(
    id              UUID PRIMARY KEY             DEFAULT gen_random_uuid(),
    event_type      VARCHAR(64)         NOT NULL,
    destination     VARCHAR(255)        NOT NULL,
    payload         JSONB               NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    status          VARCHAR(50)         NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'succeed', 'failed'
    retry_count     INT                 NOT NULL DEFAULT 0,
    max_retries     INT                 NOT NULL DEFAULT 5,
    last_error      TEXT,
    last_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT now()
);

-- polling index for worker batch fetching
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending_polling
    ON outbox_events (created_at ASC)
    WHERE status IN ('pending', 'failed') AND retry_count < max_retries;

-- cleanup index for pruning old succeeded events
CREATE INDEX IF NOT EXISTS idx_outbox_events_cleanup
    ON outbox_events (last_attempt_at ASC)
    WHERE status = 'succeed';