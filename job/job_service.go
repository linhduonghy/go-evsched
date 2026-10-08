package job

import (
	"context"
	"encoding/json"
	"errors"
	"evsched/db"
	"evsched/db/store"
	"evsched/domain"
	"evsched/logging"
	"evsched/scheduler/job_store"
	"evsched/utils"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type JobService interface {
	AddJob(ctx context.Context, jobType string, sourceID string, data map[string]any, isCron bool, cron string, scheduledAt time.Time) (AddJobResult, error)
	Pause(ctx context.Context, jobID string) error
	Resume(ctx context.Context, jobID string) error
	Delete(ctx context.Context, jobID string) error
	Get(ctx context.Context, jobID string) (*JobDetail, error)
	GetNextRunTime(ctx context.Context, jobID string) (time.Time, error)
}

type JobBusiness struct {
	logger   *slog.Logger
	store    *store.Store
	jobStore *store.JobStore
	sched    job_store.Scheduler
	// handlers contains job handler for specific job type
	handlers map[JobType]Handler
}

func NewJobService(logger *slog.Logger, st *store.Store, jobStore *store.JobStore, overrideHandlers map[JobType]Handler) (*JobBusiness, error) {
	handlers := InitHandlers(overrideHandlers)
	return &JobBusiness{logger: logger, store: st, jobStore: jobStore, handlers: handlers}, nil
}

func (s *JobBusiness) SetScheduler(sched job_store.Scheduler) {
	s.sched = sched
}

func (s *JobBusiness) AddJob(ctx context.Context, jobType string, sourceID string, data map[string]any, isCron bool, cron string, scheduledAt time.Time) (AddJobResult, error) {

	if !isCron {
		return s.addOneTimeJob(ctx,
			domain.JobData{
				JobType:  jobType,
				SourceID: sourceID,
				RawData:  data,
			}, scheduledAt)
	}

	return s.addCronJob(ctx,
		domain.JobData{
			JobType:  jobType,
			SourceID: sourceID,
			RawData:  data,
		}, cron)

}

func (s *JobBusiness) addOneTimeJob(ctx context.Context, data domain.JobData, scheduledAt time.Time) (AddJobResult, error) {
	return s.addJob(ctx, data, db.OneTimeJob, scheduledAt, "")
}

func (s *JobBusiness) addCronJob(ctx context.Context, data domain.JobData, cron string) (AddJobResult, error) {

	nextRunTime, err := getNextRunTimeCron(time.Now(), cron)
	if err != nil {
		return AddJobResult{}, err
	}

	return s.addJob(ctx, data, db.CronJob, nextRunTime, cron)
}

// getNextRunTimeCron calculate the next run of cron expression
func getNextRunTimeCron(from time.Time, cronEx string) (time.Time, error) {
	parser, err := utils.CronParser.Parse(cronEx)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron expression: %w", err)
	}
	nextRunTime := parser.Next(from)
	if nextRunTime.Before(from) {
		return time.Time{}, fmt.Errorf("cron expression is not valid")
	}
	return nextRunTime, nil
}

// addJob add job for both onetime and cron job
// scheduledAt is the next run time if cron job
func (s *JobBusiness) addJob(ctx context.Context, data domain.JobData, schedType db.JobScheduleType, scheduledAt time.Time, cronEx string) (AddJobResult, error) {

	scheduledAt = scheduledAt.UTC()

	var job store.CreateJobResult
	err := s.store.WithTx(ctx, func(tx pgx.Tx) error {
		var createErr error

		// create managed jobs
		payload, errPayload := json.Marshal(data.RawData)
		if errPayload != nil {
			return fmt.Errorf("job payload marshalling failed: %w", errPayload)
		}

		jobTx := s.jobStore.WithTx(tx)
		job, createErr = jobTx.CreateJob(ctx, store.CreateJobArgs{
			JobType:      data.JobType,
			ScheduleType: schedType,
			ScheduledAt:  scheduledAt,
			CronExpr:     cronEx,
			SourceID:     data.SourceID,
			Payload:      payload,
			Enabled:      true,
		})
		if createErr != nil {
			return fmt.Errorf("create managed job: %w", createErr)
		}

		// insert river jobs
		riverJob, insertRiverErr := s.sched.AddJob(ctx, tx, job.JobID, job.ScheduledAt)
		if insertRiverErr != nil {
			return fmt.Errorf("insert river job: %w", insertRiverErr)
		}

		// update active river job id
		if err1 := jobTx.UpdateActiveSchedulerJobID(ctx, job.JobID, riverJob.JobID); err1 != nil {
			return fmt.Errorf("update active scheduler job id: %w", err1)
		}

		return nil
	})

	if err != nil {
		return AddJobResult{}, fmt.Errorf("failed to add job: %w", err)
	}

	return AddJobResult{
		JobID:       job.JobID.String(),
		ScheduledAt: job.ScheduledAt,
		NextRunAt:   job.NextRunAt,
	}, nil
}

func (s *JobBusiness) Pause(ctx context.Context, jobID string) error {
	ID, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("fail to parse job id: %w", err)
	}

	return s.store.WithTx(ctx, func(tx pgx.Tx) error {
		// update job state
		riverJobID, errTx := s.jobStore.WithTx(tx).UpdateJobStateWithConstrainStates(ctx, ID, []db.JobState{db.JobStatePending, db.JobStateScheduled}, db.JobStatePaused)
		if errTx != nil {
			return fmt.Errorf("fail to update job state with jobID = %s: %w", jobID, errTx)
		}

		// cancel river job
		if errTx = s.sched.PauseJob(ctx, tx, riverJobID); errTx != nil {
			return fmt.Errorf("fail to pause scheduler job: %w", err)
		}

		return nil
	})
}

func (s *JobBusiness) Resume(ctx context.Context, jobID string) error {
	ID, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("fail to parse job id: %w", err)
	}

	return s.store.WithTx(ctx, func(tx pgx.Tx) error {

		riverJobID, errTx := s.jobStore.WithTx(tx).UpdateJobStateWithConstrainStates(ctx, ID, []db.JobState{db.JobStatePaused}, db.JobStateRunning)
		if errTx != nil {
			return fmt.Errorf("fail to update job state with jobID = %s: %w", jobID, errTx)
		}

		// cancel river job
		if errTx = s.sched.ResumeJob(ctx, tx, riverJobID); errTx != nil {
			return fmt.Errorf("fail to resume scheduler job with jobID = %s: %w", jobID, errTx)
		}

		return nil
	})
}

func (s *JobBusiness) Delete(ctx context.Context, jobID string) error {
	ID, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("fail to parse job id: %w", err)
	}

	return s.store.WithTx(ctx, func(tx pgx.Tx) error {

		riverJobID, errTx := s.jobStore.WithTx(tx).DeleteJob(ctx, ID)
		if errTx != nil {
			return fmt.Errorf("fail to delete job with jobID = %s: %w", jobID, errTx)
		}

		if errTx = s.sched.DeleteJob(ctx, tx, riverJobID); errTx != nil {
			return fmt.Errorf("fail to delete scheduler job with jobID = %s: %w", jobID, errTx)
		}

		return err
	})
}

func (s *JobBusiness) Get(ctx context.Context, jobID string) (*JobDetail, error) {
	ID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("fail to parse job id: %w", err)
	}

	job, err := s.jobStore.GetJob(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("fail to get job id=%s: %w", jobID, err)
	}

	return &JobDetail{
		ID:           job.ID.String(),
		JobType:      job.JobType,
		ScheduleType: job.ScheduleType.String(),
		ScheduledAt:  job.ScheduledAt,
		CronExpr:     job.CronExpr,
		SourceID:     job.SourceID,
		Payload:      job.Payload,
		LastRunAt:    job.LastRunAt,
		NextRunAt:    job.NextRunAt,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
	}, nil
}

func (s *JobBusiness) GetNextRunTime(ctx context.Context, jobID string) (time.Time, error) {
	ID, err := uuid.Parse(jobID)
	if err != nil {
		return time.Time{}, fmt.Errorf("fail to parse job id: %w", err)
	}
	return s.jobStore.GetNextRunTime(ctx, ID)
}

func (s *JobBusiness) ExecuteJob(ctx context.Context, jobID uuid.UUID, schedulerJobID int64) error {

	logger := logging.FromContext(ctx)

	idempotencyKey := fmt.Sprintf("%s:%d", jobID, schedulerJobID)

	return s.store.WithTx(ctx, func(tx pgx.Tx) error {
		jobStoreTx := s.jobStore.WithTx(tx)
		job, err := jobStoreTx.ClaimJobForExecution(ctx, jobID, schedulerJobID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Job was deleted from DB
				// Job was already 'completed' or 'cancelled'
				// active_scheduler_job_id mismatched (stale or rescheduled execution)
				logger.WarnContext(ctx, "job execution skipped: job not found, already finished, or active_scheduler_job_id mismatched")
				return nil
			}

			logger.ErrorContext(ctx, "failed to claim job for execution", "err", err)
			return fmt.Errorf("claim job failed: %w", err)
		}

		err = s.executeJobInternal(ctx, tx, job, idempotencyKey)
		if err != nil {
			logger.ErrorContext(ctx, "execute job handler failed", "err", err)

			tx.Rollback(ctx)

			_ = s.jobStore.MarkJobFailed(ctx, jobID, err.Error())

			return fmt.Errorf("execute internal job failed: %w", err)
		}

		return s.completedJob(ctx, tx, jobStoreTx, job)
	})

}

// executeJobInternal execute job correspond with job type handlers
func (s *JobBusiness) executeJobInternal(ctx context.Context, tx pgx.Tx, jobClaim *store.JobClaim, idempotencyKey string) error {
	logger := logging.FromContext(ctx)

	jobType := JobType(jobClaim.JobType)

	handler, ok := s.handlers[jobType]
	if !ok {
		logger.WarnContext(ctx, fmt.Sprintf("unsupport jobClaim type %s", jobClaim.JobType))
		return nil
	}

	if jobType == ExecuteWorkflow { // pass tx to workflow handler
		if wfHandler, ok := handler.(*workflowHandler); ok {
			handler = wfHandler.WithTx(tx)
		}
	}

	err := handler.Execute(ctx, jobType, jobClaim.Payload, idempotencyKey)
	if err != nil {
		return err
	}
	return nil
}

func (s *JobBusiness) completedJob(ctx context.Context, tx pgx.Tx, jobTx *store.JobStore, job *store.JobClaim) error {

	if job.ScheduleType == db.OneTimeJob {
		return jobTx.CompleteOneTimeJob(ctx, job.ID, db.JobStateSucceeded)
	} else {
		// with cron job, calculate next run time
		nextRunAt, errCron := getNextRunTimeCron(time.Now(), job.Cron)
		if errCron != nil {
			return errCron
		}

		if !nextRunAt.IsZero() {
			schedJob, errCron := s.sched.AddJob(ctx, tx, job.ID, nextRunAt)
			if errCron != nil {
				return fmt.Errorf("add scheduler job failed: %w", errCron)
			}

			if errCron = jobTx.RescheduleCronJobAfterExecution(ctx, job.ID, nextRunAt, schedJob.JobID); errCron != nil {
				return fmt.Errorf("reschedule cron job after execution failed: %w", errCron)
			}

			return nil

		} else {
			logging.FromContext(ctx).ErrorContext(ctx, "get next time for cron job failed", "cron", job.Cron)
			return fmt.Errorf("get next time for cron job failed")
		}
	}
}

func (s *JobBusiness) markJobFailed(ctx context.Context, job *store.JobClaim) {

}

func (s *JobBusiness) MarkJobCancel(ctx context.Context, jobID uuid.UUID) error {
	logger := logging.FromContext(ctx)

	job, err := s.jobStore.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.WarnContext(ctx, "job missing from database")
			return nil
		}

		logger.ErrorContext(ctx, "failed to get job from database", "err", err)
		return fmt.Errorf("get job failed: %w", err)
	}

	// mark cancel only for onetime job
	if job.ScheduleType == db.OneTimeJob {
		err = s.jobStore.UpdateJobState(ctx, job.ID, db.JobStateCancelled)
		if err != nil {
			logger.ErrorContext(ctx, "mark job cancel failed", "err", err)
			return err
		}
	}

	return nil

}
