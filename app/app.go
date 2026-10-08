package app

import (
	"context"
	"errors"
	"evsched/config"
	"evsched/db"
	"evsched/db/store"
	apphttp "evsched/http"
	"evsched/job"
	"evsched/logging"
	"evsched/message_queue"
	"evsched/metric"
	"evsched/observability"
	"evsched/outbox"
	"evsched/scheduler"
	"evsched/scheduler/job_store"
	"evsched/scheduler/memory"
	"evsched/utils"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config          *config.Config
	logger          *slog.Logger
	server          *http.Server
	db              *pgxpool.Pool
	rabbitmqClient  *message_queue.Client
	obs             *observability.Observability
	schedManager    *scheduler.Manager
	outboxPublisher *outbox.OutboxPublisher
}

type BusinessDependencies struct {
	store       *store.Store
	JobService  *job.JobBusiness
	jobExecutor job_store.JobExecutor
}

func New(ctx context.Context, defaultConfigPath, configPath string, initDB *bool) (a *App, err error) {

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.Load(defaultConfigPath, configPath)
	if err != nil {
		return nil, fmt.Errorf("errors load config: %w", err)
	}

	// obs
	obs, err := observability.New(initCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("init observability: %w", err)
	}

	// logger
	logger := logging.New(cfg, obs.LogProvider)

	a = &App{
		config: cfg,
		obs:    obs,
		logger: logger,
	}

	defer func() {
		if err != nil && a != nil {
			if shutdownErr := a.shutdown(); shutdownErr != nil {
				logger.Error(
					"startup cleanup failed",
					"error",
					shutdownErr,
				)
			}
		}
	}()

	// http client
	httpClient := utils.NewHttpClient(cfg.Http.Timeout)

	// db
	a.db, err = db.NewPostgres(initCtx, cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	// init river schema
	if *initDB {
		if err = db.InitDB(ctx, a.logger, a.db); err != nil {
			return nil, err
		}
	}

	// rabbitmq
	a.rabbitmqClient, err = message_queue.NewClient(ctx, cfg.RabbitMQ.URL(), a.logger)
	if err != nil {
		return nil, fmt.Errorf("init rabbitmq client: %w", err)
	}

	// db store

	// metrics
	metrics := metric.NewMetrics()

	// memory scheduler
	var memSched memory.Scheduler
	memSched, err = memory.NewScheduler(cfg.TZ)
	if err != nil {
		return nil, fmt.Errorf("init memory scheduler: %w", err)
	}

	// handler
	businessDep, err := a.setupBusinessDependencies(ctx, cfg, a.db, a.logger, metrics, a.obs, memSched, httpClient, a.rabbitmqClient)
	if err != nil {
		return nil, fmt.Errorf("init dependencies: %w", err)
	}

	// scheduler
	a.schedManager, err = scheduler.NewManager(a.logger, a.obs.TracerProvider, a.config.Job, memSched, businessDep.store, businessDep.jobExecutor)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	err = postSetupDependencies(businessDep, a.schedManager)
	if err != nil {
		return nil, fmt.Errorf("post setup dependencies: %w", err)
	}

	// router
	r := apphttp.NewRouter(logger, metrics, a.db, businessDep.JobService)
	addr := fmt.Sprintf("%s:%d", "", cfg.Server.Port)
	a.server = newHttpServer(addr, r)

	return a, nil
}

func postSetupDependencies(dep *BusinessDependencies, manager *scheduler.Manager) error {
	dep.JobService.SetScheduler(manager.JobStore)
	return nil
}

func (a *App) setupBusinessDependencies(ctx context.Context, cfg *config.Config, db *pgxpool.Pool, logger *slog.Logger,
	m *metric.Metrics, obs *observability.Observability,
	memSched memory.Scheduler, httpClient *utils.HttpClient, rabbitClient *message_queue.Client) (*BusinessDependencies, error) {

	st := store.NewStore(db)
	jobStore := store.NewJobStore(db)
	outboxEventStore := store.NewOutboxEventStore(db)

	workflowJobPublisherCfg, ok := cfg.Job.JobStore.Publisher[string(job.ExecuteWorkflow)]
	if !ok {
		return nil, fmt.Errorf("no workflow job publisher")
	}
	workflowJobHandler := job.NewWorkflowHandler(outboxEventStore, workflowJobPublisherCfg)

	jobService, err := job.NewJobService(logger, st, jobStore, map[job.JobType]job.Handler{job.ExecuteWorkflow: workflowJobHandler})
	if err != nil {
		return nil, fmt.Errorf("init job service: %w", err)
	}

	// outbox event publisher
	a.outboxPublisher = outbox.NewPublisher(logger, outbox.NewEventPublisher(rabbitClient), outboxEventStore)

	return &BusinessDependencies{
		store:       st,
		JobService:  jobService,
		jobExecutor: jobService,
	}, nil
}

func newHttpServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (a *App) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	// Start Outbox publisher
	go func() {
		a.logger.Info("Starting outbox publisher")
		a.outboxPublisher.Start(ctx, a.config.OutBox.PollInterval)
	}()

	// Start Scheduler
	a.logger.Info("Starting scheduler")
	if err := a.schedManager.Start(ctx); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	// Start HTTP server
	go func() {
		a.logger.Info("Starting HTTP server", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server runtime error: %w", err)
		}
	}()

	// Wait for cancel signal
	select {
	case err := <-errCh:
		a.logger.Error("server error encountered, shutting down", "error", err)
		shutdownErr := a.shutdown()
		return errors.Join(err, shutdownErr)

	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
		return a.shutdown()
	}

}

func (a *App) shutdown() error {

	a.logger.Info("initiating graceful application shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var errs []error

	if a.server != nil {
		a.logger.Info("stopping HTTP server...")
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown http server: %w", err))
		}
	}

	if a.schedManager != nil {
		a.logger.Info("stopping scheduler...")
		if err := a.schedManager.Stop(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("stop scheduler: %w", err))
		}
	}

	if a.outboxPublisher != nil {
		a.logger.Info("stopping outbox publisher...")
		if err := a.outboxPublisher.Stop(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("stop outbox publisher: %w", err))
		}
	}

	if a.rabbitmqClient != nil {
		a.logger.Info("closing rabbitmq client...")
		if err := a.rabbitmqClient.Close(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("close rabbitmq client: %w", err))
		}
	}

	if a.db != nil {
		a.logger.Info("closing database pool...")
		a.db.Close()
	}

	if a.obs != nil {
		a.logger.Info("shutting down observability...")
		if err := a.obs.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown observability: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	a.logger.Info("application stopped gracefully")
	return nil
}
