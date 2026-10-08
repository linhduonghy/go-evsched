package job

import (
	"time"
)

type JobType string

const (
	DefaultJob      JobType = "default_job"
	ExecuteWorkflow JobType = "workflow"
)

func (jt JobType) String() string {
	return string(jt)
}

var AllJobTypes = []JobType{
	DefaultJob,
	ExecuteWorkflow,
}

type AddJobResult struct {
	JobID       string
	ScheduledAt time.Time
	NextRunAt   time.Time
}

type ResumeJobResult struct {
	JobID     string
	NextRunAt time.Time
}

type JobDetail struct {
	ID           string    `json:"id"`
	JobType      string    `json:"job_type"`
	ScheduleType string    `json:"schedule_type"`
	ScheduledAt  time.Time `json:"scheduled_at"`
	CronExpr     string    `json:"cron_expr"`
	SourceID     string    `json:"source_id"`
	Payload      []byte    `json:"payload"`
	LastRunAt    time.Time `json:"last_run_at"`
	NextRunAt    time.Time `json:"next_run_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
