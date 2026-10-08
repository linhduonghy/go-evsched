package response

import "time"

type JobResponse struct {
	JobID       string    `json:"job_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}
