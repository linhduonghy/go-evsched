-- name: CreateJob :one
INSERT INTO jobs (job_type, schedule_type, scheduled_at, cron_expr, source_id, payload, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetJob :one
SELECT *
from jobs
WHERE id = $1;

-- name: GetNextRunTime :one
SELECT next_run_at
FROM jobs
where id = $1;

-- name: ClaimJobForExecution :one
UPDATE jobs
SET state = 'running',
    updated_at = now()
WHERE id = $1
  AND active_scheduler_job_id = $2
  AND state IN ('scheduled', 'pending', 'running', 'failed')
    RETURNING id, job_type, state, schedule_type, scheduled_at, cron_expr, active_scheduler_job_id, payload;

-- name: UpdateJobStateWithConstrainStates :one
UPDATE jobs
SET state      = @new_state,
    updated_at = now()
WHERE id = @id
  AND state = ANY (@states::job_state[]) RETURNING active_scheduler_job_id;

-- name: UpdateJobState :exec
UPDATE jobs
SET state      = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpdateActiveSchedulerJobID :exec
UPDATE jobs
SET active_scheduler_job_id = $2
WHERE id = $1;

-- name: CompleteOneTimeJob :exec
UPDATE jobs
SET state = $2,
    last_run_at = now(),
    updated_at = now()
WHERE id = $1 AND state = 'running';

-- name: RescheduleCronJob :exec
UPDATE jobs
SET state = 'scheduled',
    next_run_at = $2,
    active_scheduler_job_id = $3,
    last_run_at = now(),
    updated_at = now()
WHERE id = $1 AND state = 'running';

-- name: MarkJobFailed :exec
UPDATE jobs
SET state = 'failed',
    last_error = @last_error,
    last_run_at = now(),
    updated_at = now()
WHERE id = @id;

-- name: DeleteJobState :one
DELETE
FROM jobs
WHERE id = @id RETURNING active_scheduler_job_id;

