package job_store

import (
	"context"
	"errors"
	"evsched/config"
	"evsched/db/store"
	apperror "evsched/errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel/trace"
)

type Scheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	AddJob(ctx context.Context, tx pgx.Tx, managedJobID uuid.UUID, scheduledAt time.Time) (AddJobResult, error)
	PauseJob(ctx context.Context, tx pgx.Tx, jobID int64) error
	ResumeJob(ctx context.Context, tx pgx.Tx, jobID int64) error
	DeleteJob(ctx context.Context, tx pgx.Tx, jobID int64) error
	GetJob(ctx context.Context, jobID int64) (*Job, error)
	GetNextRunTime(ctx context.Context, jobID int64) (time.Time, error)
}

type scheduler struct {
	store      *store.Store
	riverSched *riverScheduler
}

func NewScheduler(logger *slog.Logger, trace trace.Tracer, jobCfg config.JobStoreConfig, store *store.Store, jobExecutor JobExecutor) (Scheduler, error) {

	worker := NewWorker(jobCfg, jobExecutor)
	// wrap worker in trace context
	traceWorker := NewTraceWorker(logger, trace, worker)

	workers := river.NewWorkers()
	river.AddWorker(workers, traceWorker)

	riverSched, err := NewRiverScheduler(store.DBPool(), workers)

	if err != nil {
		return nil, fmt.Errorf("failed to create river scheduler: %w", err)
	}

	return &scheduler{store: store, riverSched: riverSched}, nil
}

func (s *scheduler) Start(ctx context.Context) error {
	return s.riverSched.Start(ctx)
}

func (s *scheduler) Stop(ctx context.Context) error {
	return s.riverSched.Stop(ctx)
}

func (s *scheduler) AddJob(ctx context.Context, tx pgx.Tx, managedJobID uuid.UUID, scheduledAt time.Time) (AddJobResult, error) {
	// insert river jobs
	riverJob, err := s.riverSched.InsertTx(ctx, tx,
		GenericJobArgs{
			ManagedJobID: managedJobID,
		},
		&river.InsertOpts{ScheduledAt: scheduledAt, UniqueOpts: DefaultUniqueOpts},
	)
	if err != nil {
		return AddJobResult{}, err
	}

	return AddJobResult{
		JobID:       riverJob.Job.ID,
		ScheduledAt: riverJob.Job.ScheduledAt,
	}, nil
}

func (s *scheduler) PauseJob(ctx context.Context, tx pgx.Tx, jobID int64) error {
	if _, err := s.riverSched.CancelJobTx(ctx, tx, jobID); err != nil {
		return fmt.Errorf("fail to cancel river job: %w", err)
	}
	return nil
}

func (s *scheduler) ResumeJob(ctx context.Context, tx pgx.Tx, jobID int64) error {
	// cancel river job
	if err := s.riverSched.ResumeJobTx(ctx, tx, jobID); err != nil {
		return fmt.Errorf("fail to resume river job: %w", err)
	}
	return nil
}

func (s *scheduler) DeleteJob(ctx context.Context, tx pgx.Tx, jobID int64) error {

	_, err := s.riverSched.JobDeleteTx(ctx, tx, jobID)
	if err != nil {
		if errors.Is(err, rivertype.ErrNotFound) {
			return nil
		}
		if errors.Is(err, rivertype.ErrJobRunning) {
			return apperror.Conflict("job is running")
		}
		return err
	}

	return err

}

func (s *scheduler) GetJob(ctx context.Context, jobID int64) (*Job, error) {

	row, err := s.riverSched.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(fmt.Sprintf("job %d", jobID))
		}
		return nil, fmt.Errorf("fail to get job id %d: %w", jobID, err)
	}

	return &Job{
		ID:          row.ID,
		ScheduledAt: row.ScheduledAt,
		EncodedArgs: row.EncodedArgs,
		FinalizedAt: row.FinalizedAt,
		State:       string(row.State),
	}, nil
}

func (s *scheduler) GetNextRunTime(ctx context.Context, ID int64) (time.Time, error) {
	return s.riverSched.GetScheduledTime(ctx, ID)

}
