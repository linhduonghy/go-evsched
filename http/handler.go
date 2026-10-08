package http

import (
	"evsched/metric"
)

type Handler struct {
	Health *HealthHandler
	Metric *metric.HttpHandler
	Job    *JobHandler
}
