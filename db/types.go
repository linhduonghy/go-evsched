package db

type JobScheduleType string

const (
	OneTimeJob JobScheduleType = "one_time"
	CronJob    JobScheduleType = "cron"
)

func (s JobScheduleType) String() string {
	return string(s)
}

type JobState string

const (
	JobStateScheduled JobState = "scheduled"
	JobStatePending   JobState = "pending"
	JobStateRunning   JobState = "running"
	JobStateCancelled JobState = "cancelled"
	JobStateDiscarded JobState = "discarded"
	JobStatePaused    JobState = "paused"
	JobStateFailed    JobState = "failed"
	JobStateSucceeded JobState = "succeeded"
)

func (j JobState) String() string {
	return string(j)
}
