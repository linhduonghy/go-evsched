package http

import (
	"evsched/job"
	"evsched/metric"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(
	logger *slog.Logger,
	metrics *metric.Metrics,
	db *pgxpool.Pool,
	jobService job.JobService,
) http.Handler {

	// handler
	h := &Handler{
		Metric: metrics.NewHttpHandler(),
		Health: NewHealthHandle(db),
		Job:    NewJobHandler(jobService),
	}

	r := chi.NewRouter()

	r.Use(Recover(logger))
	r.Use(otelhttp.NewMiddleware("http-server"))
	r.Use(Context(logger))
	r.Use(Metric(metrics))

	r.Route("/health", func(r chi.Router) {
		r.Get("/live", h.Health.Live)
		r.Get("/ready", h.Health.Ready)
	})

	// metric
	r.Handle("/metrics", h.Metric)

	// app route
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/jobs", func(r chi.Router) {
			r.Post("/", h.Job.AddJob)
			r.Post("/pause/{jobID}", h.Job.PauseJob)
			r.Post("/resume/{jobID}", h.Job.ResumeJob)
			r.Delete("/{jobID}", h.Job.DeleteJob)
			r.Get("/{jobID}", h.Job.GetJob)
			r.Get("/{jobID}/scheduled_time", h.Job.GetJobScheduledTime)
		})
	})

	return r
}
