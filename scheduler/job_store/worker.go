package job_store

import (
	"context"
	"evsched/config"
	"evsched/logging"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// JobExecutor defines capability Worker needs to delegate execution.
type JobExecutor interface {
	ExecuteJob(ctx context.Context, jobID uuid.UUID, schedulerJobID int64) error
	MarkJobCancel(ctx context.Context, jobID uuid.UUID) error
}

type Worker struct {
	river.WorkerDefaults[GenericJobArgs]
	misfireThreshold time.Duration
	executor         JobExecutor
}

func NewWorker(cfg config.JobStoreConfig, executor JobExecutor) *Worker {
	return &Worker{
		misfireThreshold: cfg.MisfireThreshold,
		executor:         executor,
	}
}

func (w *Worker) Work(ctx context.Context, job *river.Job[GenericJobArgs]) error {

	logger := logging.FromContext(ctx).With("job_id", job.Args.ManagedJobID, "scheduler_job_id", job.JobRow.ID)
	ctx = logging.BindContext(ctx, logger)
	logger.InfoContext(ctx, "executing river job worker")

	// check job misfired
	if w.misfireThreshold > 0 && !job.ScheduledAt.IsZero() {
		if delay := time.Since(job.ScheduledAt); delay > w.misfireThreshold {
			logger.Warn("job misfired: threshold exceeded", "scheduled_at", job.ScheduledAt,
				"delay", delay.Round(time.Millisecond), "threshold", w.misfireThreshold)

			// mark job as cancel
			err := w.executor.MarkJobCancel(ctx, job.Args.ManagedJobID)
			if err != nil {
				logger.ErrorContext(ctx, "mark job cancel fail", "err", err)
				return err
			}

			// Cancel the job so River stops processing and does not retry it
			return river.JobCancel(fmt.Errorf("misfire threshold exceeded: delayed by %s (max %s)",
				delay.Round(time.Second), w.misfireThreshold))
		}
	}

	return w.executor.ExecuteJob(ctx, job.Args.ManagedJobID, job.JobRow.ID)
}
