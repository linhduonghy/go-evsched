package scheduler

import (
	"context"
	"evsched/config"
	"evsched/db/store"
	"evsched/scheduler/job_store"
	"evsched/scheduler/memory"
	"fmt"
	"log/slog"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Manager coordinates the lifecycle of both durable and Memory schedulers
type Manager struct {
	JobStore job_store.Scheduler
	Memory   memory.Scheduler
}

func NewManager(logger *slog.Logger, traceProvider *sdktrace.TracerProvider, jobCfg config.JobConfig, memSched memory.Scheduler, store *store.Store, jobExecutor job_store.JobExecutor) (*Manager, error) {
	jobStore, err := job_store.NewScheduler(logger, traceProvider.Tracer("evschedgo/scheduler"), jobCfg.JobStore, store, jobExecutor)
	if err != nil {
		return nil, fmt.Errorf("error creating job store: %w", err)
	}

	return &Manager{
		JobStore: jobStore,
		Memory:   memSched,
	}, nil
}

// Start boots both background job runners
func (m *Manager) Start(ctx context.Context) error {
	if m.JobStore != nil {
		if err := m.JobStore.Start(ctx); err != nil {
			return fmt.Errorf("failed to Start durable scheduler: %w", err)
		}
	}

	if m.Memory != nil {
		m.Memory.Start()
	}

	return nil
}

// Stop safely shuts down both schedulers (useful for graceful shutdown)
func (m *Manager) Stop(ctx context.Context) error {
	var durableErr, memoryErr error

	if m.JobStore != nil {
		durableErr = m.JobStore.Stop(ctx)
	}

	if m.Memory != nil {
		memoryErr = m.Memory.Shutdown(ctx)
	}

	if durableErr != nil || memoryErr != nil {
		return fmt.Errorf("shutdown errors: durable=%v, Memory=%v", durableErr, memoryErr)
	}

	return nil
}
