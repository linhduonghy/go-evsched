package job_store

import (
	"time"
)

type AddJobResult struct {
	JobID       int64
	ScheduledAt time.Time
}

type ResumeJobResult struct {
	JobID       int64
	ScheduledAt time.Time
}

type Job struct {
	ID          int64
	ScheduledAt time.Time
	EncodedArgs []byte
	FinalizedAt *time.Time
	State       string
}
