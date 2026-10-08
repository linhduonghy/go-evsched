package store

import (
	"context"
	"evsched/db"
	"evsched/db/sqlc"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateJobArgs struct {
	JobType      string
	ScheduleType db.JobScheduleType
	ScheduledAt  time.Time
	CronExpr     string
	SourceID     string
	Payload      []byte
	Enabled      bool
}

type CreateJobResult struct {
	JobID       uuid.UUID
	ScheduledAt time.Time
	NextRunAt   time.Time
}

type Job struct {
	ID                   uuid.UUID          `json:"id"`
	State                db.JobState        `json:"state"`
	JobType              string             `json:"job_type"`
	ScheduleType         db.JobScheduleType `json:"schedule_type"`
	ActiveSchedulerJobID *int64             `json:"active_scheduler_job_id"`
	ScheduledAt          time.Time          `json:"scheduled_at"`
	CronExpr             string             `json:"cron_expr"`
	SourceID             string             `json:"source_id"`
	Payload              []byte             `json:"payload"`
	Enabled              bool               `json:"enabled"`
	LastRunAt            time.Time          `json:"last_run_at"`
	NextRunAt            time.Time          `json:"next_run_at"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

type JobClaim struct {
	ID           uuid.UUID
	JobType      string
	ScheduleType db.JobScheduleType
	State        db.JobState
	Cron         string
	ScheduledAt  time.Time
	Payload      []byte
}

type JobStore struct {
	querier sqlc.Querier
}

func NewJobStore(db sqlc.DBTX) *JobStore {
	return &JobStore{
		querier: sqlc.New(db),
	}
}

func (s *JobStore) WithTx(tx sqlc.DBTX) *JobStore {
	return &JobStore{querier: sqlc.New(tx)}
}

func (s *JobStore) CreateJob(ctx context.Context, arg CreateJobArgs) (CreateJobResult, error) {
	job, err := s.querier.CreateJob(ctx, sqlc.CreateJobParams{
		JobType:      arg.JobType,
		ScheduleType: arg.ScheduleType,
		ScheduledAt:  db.TimeToPgTimestamptz(arg.ScheduledAt),
		CronExpr:     db.StringToPgText(arg.CronExpr),
		SourceID:     db.StringToPgText(arg.SourceID),
		Payload:      arg.Payload,
		Enabled:      arg.Enabled,
	})
	if err != nil {
		return CreateJobResult{}, fmt.Errorf("fail to create job: %w", err)
	}

	return CreateJobResult{
		JobID:       job.ID,
		ScheduledAt: db.PgTimestamptzToTime(job.ScheduledAt),
		NextRunAt:   db.PgTimestamptzToTime(job.NextRunAt),
	}, nil

}

func (s *JobStore) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {

	job, err := s.querier.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &Job{
		ID:                   job.ID,
		State:                job.State,
		JobType:              job.JobType,
		ScheduleType:         job.ScheduleType,
		ActiveSchedulerJobID: db.PgInt8ToInt64(job.ActiveSchedulerJobID),
		ScheduledAt:          db.PgTimestamptzToTime(job.ScheduledAt),
		CronExpr:             db.PgTextToString(job.CronExpr),
		SourceID:             db.PgTextToString(job.SourceID),
		Payload:              job.Payload,
		Enabled:              job.Enabled,
		LastRunAt:            db.PgTimestamptzToTime(job.LastRunAt),
		NextRunAt:            db.PgTimestamptzToTime(job.NextRunAt),
		CreatedAt:            db.PgTimestamptzToTime(job.CreatedAt),
		UpdatedAt:            db.PgTimestamptzToTime(job.UpdatedAt),
	}, nil
}

func (s *JobStore) GetNextRunTime(ctx context.Context, jobID uuid.UUID) (time.Time, error) {
	nextRunTime, err := s.querier.GetNextRunTime(ctx, jobID)
	if err != nil {
		return time.Time{}, err
	}
	return nextRunTime.Time, nil
}

func (s *JobStore) UpdateJobStateWithConstrainStates(ctx context.Context, jobID uuid.UUID, oldStates []db.JobState, newState db.JobState) (int64, error) {
	riverJobID, err := s.querier.UpdateJobStateWithConstrainStates(ctx, sqlc.UpdateJobStateWithConstrainStatesParams{
		NewState: newState,
		ID:       jobID,
		States:   oldStates,
	})
	if err != nil {
		return 0, err
	}

	if riverJobID.Valid {
		return riverJobID.Int64, nil
	}

	return 0, fmt.Errorf("invalid river job with jobID: %s", jobID)
}

func (s *JobStore) UpdateJobState(ctx context.Context, jobID uuid.UUID, state db.JobState) error {
	return s.querier.UpdateJobState(ctx, sqlc.UpdateJobStateParams{
		ID:    jobID,
		State: state,
	})

}

func (s *JobStore) UpdateActiveSchedulerJobID(ctx context.Context, jobID uuid.UUID, schedulerJobID int64) error {
	return s.querier.UpdateActiveSchedulerJobID(ctx, sqlc.UpdateActiveSchedulerJobIDParams{
		ID: jobID,
		ActiveSchedulerJobID: pgtype.Int8{
			Int64: schedulerJobID,
			Valid: true,
		},
	})
}

func (s *JobStore) DeleteJob(ctx context.Context, jobID uuid.UUID) (int64, error) {

	riverJobID, err := s.querier.DeleteJobState(ctx, jobID)
	if err != nil {
		return 0, err
	}

	if riverJobID.Valid {
		return riverJobID.Int64, nil
	}

	return 0, fmt.Errorf("invalid river job with jobID: %s", jobID)
}

func (s *JobStore) RescheduleCronJobAfterExecution(ctx context.Context, ID uuid.UUID, nextRunAt time.Time, activeSchedulerJobID int64) error {
	return s.querier.RescheduleCronJob(ctx, sqlc.RescheduleCronJobParams{
		ID:                   ID,
		NextRunAt:            db.TimeToPgTimestamptz(nextRunAt),
		ActiveSchedulerJobID: db.Int64ToPgInt8(activeSchedulerJobID),
	})
}

func (s *JobStore) ClaimJobForExecution(ctx context.Context, ID uuid.UUID, activeSchedulerJobID int64) (*JobClaim, error) {
	job, err := s.querier.ClaimJobForExecution(ctx, sqlc.ClaimJobForExecutionParams{
		ID:                   ID,
		ActiveSchedulerJobID: db.Int64ToPgInt8(activeSchedulerJobID),
	})
	if err != nil {
		return nil, err
	}
	return &JobClaim{
		ID:           job.ID,
		JobType:      job.JobType,
		ScheduleType: job.ScheduleType,
		State:        job.State,
		Cron:         db.PgTextToString(job.CronExpr),
		ScheduledAt:  db.PgTimestamptzToTime(job.ScheduledAt),
		Payload:      job.Payload,
	}, nil
}

func (s *JobStore) CompleteOneTimeJob(ctx context.Context, ID uuid.UUID, state db.JobState) error {
	return s.querier.CompleteOneTimeJob(ctx, sqlc.CompleteOneTimeJobParams{
		ID:    ID,
		State: state,
	})
}

func (s *JobStore) MarkJobFailed(ctx context.Context, ID uuid.UUID, err string) error {
	return s.querier.MarkJobFailed(ctx, sqlc.MarkJobFailedParams{
		LastError: db.StringToPgText(err),
		ID:        ID,
	})
}
