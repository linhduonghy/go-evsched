package request

import (
	apperror "evsched/errors"
	"time"
)

type CreateJobRequest struct {
	JobType     string         `json:"job_type"`
	JobData     map[string]any `json:"job_data"`
	IsCron      bool           `json:"is_cron"`
	Cron        string         `json:"cron"`
	ScheduledAt time.Time      `json:"scheduled_at"`

	// SourceID represents for request job ID
	// can find Job by SourceID
	SourceID string `json:"source_id"`
}

func (r *CreateJobRequest) Validate() error {

	if r.JobType == "" {
		return apperror.InvalidInput("job_type is required")
	}

	if r.JobData == nil {
		return apperror.InvalidInput("job_data is required")
	}

	if r.ScheduledAt.IsZero() || time.Now().After(r.ScheduledAt) {
		return apperror.InvalidInput("scheduled_at is required and must be in the future")
	}

	if r.SourceID == "" {
		return apperror.InvalidInput("source_id is required")
	}

	return nil

}
