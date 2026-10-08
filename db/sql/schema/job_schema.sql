
DO $$
BEGIN
CREATE TYPE job_schedule_type AS ENUM ('one_time', 'cron');
CREATE TYPE job_state AS ENUM ('scheduled', 'pending', 'running', 'cancelled', 	'discarded', 'paused', 'failed', 'succeeded');
EXCEPTION
    WHEN duplicate_object THEN NULL; -- Do nothing if it already exists
END
$$;


CREATE TABLE IF NOT EXISTS jobs
(
    id                      UUID PRIMARY KEY           DEFAULT gen_random_uuid(),
    job_type                VARCHAR(128)      NOT NULL,
    schedule_type           job_schedule_type NOT NULL, -- 'one_time' OR 'cron'
    enabled                 BOOLEAN           NOT NULL DEFAULT true,
    -- Schedule Options
    scheduled_at            TIMESTAMPTZ,                -- Populated if schedule_type = 'one_time'
    cron_expr               VARCHAR(256),               -- Populated if schedule_type = 'cron'
    source_id               VARCHAR(64),
    payload                 JSONB             NOT NULL DEFAULT '{}',
    state                   job_state         NOT NULL DEFAULT 'scheduled',
    active_scheduler_job_id BIGINT NULL,
    last_error              text NULL,
    -- Execution Auditing
    last_run_at             TIMESTAMPTZ,
    next_run_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ       NOT NULL DEFAULT now()
);