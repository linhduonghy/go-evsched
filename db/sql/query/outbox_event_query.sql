-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (event_type, destination, payload, idempotency_key, max_retries)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT (idempotency_key) DO NOTHING;

-- name: FetchPendingOutboxEvents :many
-- Uses FOR UPDATE SKIP LOCKED so multiple publisher workers can scale horizontally without race conditions
UPDATE outbox_events
SET status     = 'processing',
    updated_at = now()
WHERE id IN
    (
        SELECT id FROM outbox_events
            WHERE status IN ('pending', 'failed')
                AND retry_count < max_retries
            ORDER BY created_at ASC
            LIMIT $1
            FOR UPDATE SKIP LOCKED
    )
RETURNING id, event_type, destination, payload, idempotency_key, retry_count;

-- name: MarkOutboxEventSucceeded :exec
UPDATE outbox_events
SET status = 'succeed',
    last_attempt_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET retry_count = retry_count + 1,
    status =
        CASE
            WHEN retry_count + 1 >= max_retries THEN 'failed'
            ELSE 'pending'
        END,
    last_error = $2,
    last_attempt_at = now(),
    updated_at = now()
WHERE id = $1;