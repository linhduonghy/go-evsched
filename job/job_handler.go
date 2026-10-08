package job

import (
	"context"
	"evsched/config"
	"evsched/db/store"
	"evsched/logging"
	"evsched/outbox"

	"github.com/jackc/pgx/v5"
)

type Handler interface {
	Execute(ctx context.Context, jobType JobType, data []byte, idempotencyKey string) error
}

func InitHandlers(overrideHandlers map[JobType]Handler) map[JobType]Handler {
	handlers := make(map[JobType]Handler, len(AllJobTypes))
	for _, jt := range AllJobTypes {
		handlers[jt] = &defaultHandler{}
	}

	for k, v := range overrideHandlers {
		handlers[k] = v
	}

	return handlers
}

type defaultHandler struct {
}

func (p *defaultHandler) Execute(ctx context.Context, jobType JobType, data []byte, idempotencyKey string) error {
	logging.FromContext(ctx).Warn("job handler is not provided, switch to default job handler", "job_type", jobType, "data", data)
	return nil
}

type workflowHandler struct {
	outboxEventStore    *store.OutboxEventStore
	workflowRabbitMQCfg config.Publisher
}

func NewWorkflowHandler(outboxEventStore *store.OutboxEventStore, cfg config.Publisher) Handler {
	return &workflowHandler{outboxEventStore: outboxEventStore, workflowRabbitMQCfg: cfg}
}

func (w *workflowHandler) WithTx(tx pgx.Tx) Handler {
	return &workflowHandler{
		outboxEventStore:    store.NewOutboxEventStore(tx),
		workflowRabbitMQCfg: w.workflowRabbitMQCfg,
	}
}

func (w *workflowHandler) Execute(ctx context.Context, jobType JobType, data []byte, idempotencyKey string) error {
	dest := outbox.FormatRabbitMQDestination(w.workflowRabbitMQCfg.Exchange, w.workflowRabbitMQCfg.RoutingKey)
	return w.outboxEventStore.InsertOutboxEvent(ctx, string(outbox.RabbitMQ), dest, data, idempotencyKey, 5)
}
