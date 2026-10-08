package job_store

import (
	"context"
	"errors"
	apperror "evsched/errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// DefaultUniqueOpts Single, static unique options definition for all managed jobs
var DefaultUniqueOpts = river.UniqueOpts{
	ByArgs: true,
	ByState: []rivertype.JobState{
		rivertype.JobStateScheduled,
		rivertype.JobStateAvailable,
		rivertype.JobStateRunning,
		rivertype.JobStatePending,
	},
}

type riverScheduler struct {
	client *river.Client[pgx.Tx]
	dbPool *pgxpool.Pool
}

func NewRiverScheduler(db *pgxpool.Pool, workers *river.Workers) (*riverScheduler, error) {

	riverClient, err := river.NewClient(riverpgxv5.New(db), &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		MaxAttempts:     3,
		SoftStopTimeout: 10 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &riverScheduler{
		client: riverClient,
		dbPool: db,
	}, nil
}

func (s *riverScheduler) Start(ctx context.Context) error {
	return s.client.Start(ctx)
}

func (s *riverScheduler) Stop(ctx context.Context) error {
	return s.client.Stop(ctx)
}

func (s *riverScheduler) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return s.client.Insert(ctx, args, opts)
}

func (s *riverScheduler) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return s.client.InsertTx(ctx, tx, args, opts)
}

func (s *riverScheduler) CancelJob(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return s.client.JobCancel(ctx, id)
}

func (s *riverScheduler) CancelJobTx(ctx context.Context, tx pgx.Tx, id int64) (*rivertype.JobRow, error) {
	return s.client.JobCancelTx(ctx, tx, id)
}

// ResumeJob will update current job state to scheduled
func (s *riverScheduler) ResumeJob(ctx context.Context, jobID int64) error {
	// Update state back to scheduled,
	// clear finalized_at
	updateQuery := `
		UPDATE river_job
		SET 
			state = 'scheduled',
			finalized_at = NULL
		WHERE id = $1 
		  AND state = 'cancelled'
		  AND scheduled_at > NOW(); 
	`

	if _, err := s.dbPool.Exec(ctx, updateQuery, jobID); err != nil {
		return fmt.Errorf("failed to resume job %d: %w", jobID, err)
	}

	return nil
}

// ResumeJobTx will update current job state to scheduled with current provided transaction
func (s *riverScheduler) ResumeJobTx(ctx context.Context, tx pgx.Tx, jobID int64) error {
	// Update state back to scheduled,
	// clear finalized_at
	updateQuery := `
		UPDATE river_job
		SET 
			state = 'scheduled',
			finalized_at = NULL
		WHERE id = $1 
		  AND state = 'cancelled'
		  AND scheduled_at > NOW(); 
	`

	if _, err := tx.Exec(ctx, updateQuery, jobID); err != nil {
		return fmt.Errorf("failed to resume job %d: %w", jobID, err)
	}

	return nil
}

// ResumeJobByAddNew will create new job with origin job data.
//
//	Return ResumeJobResult
func (s *riverScheduler) ResumeJobByAddNew(ctx context.Context, tx pgx.Tx, jobID int64, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {

	row, err := s.client.InsertTx(ctx, tx, args, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to re-insert oldJob: %w", err)
	}

	// delete old oldJob
	_, err = s.client.JobDeleteTx(ctx, tx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old job %d: %w", jobID, err)
	}

	return row, nil

}

func (s *riverScheduler) JobDelete(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return s.client.JobDelete(ctx, id)
}

func (s *riverScheduler) JobDeleteTx(ctx context.Context, tx pgx.Tx, id int64) (*rivertype.JobRow, error) {
	return s.client.JobDeleteTx(ctx, tx, id)
}

func (s *riverScheduler) GetJob(ctx context.Context, id int64) (*rivertype.JobRow, error) {
	return s.client.JobGet(ctx, id)
}

func (s *riverScheduler) GetScheduledTime(ctx context.Context, jobID int64) (time.Time, error) {
	query := `
		SELECT scheduled_at
		FROM river_job 
		WHERE id = $1
	`

	var scheduledAt time.Time
	err := s.dbPool.QueryRow(ctx, query, jobID).Scan(&scheduledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, apperror.NotFound(fmt.Sprintf("job %d", jobID))
		}
		return time.Time{}, err
	}

	return scheduledAt, nil
}
