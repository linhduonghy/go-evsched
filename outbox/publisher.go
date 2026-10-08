package outbox

import (
	"context"
	"errors"
	"evsched/db/store"
	"evsched/logging"
	"evsched/message_queue"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, destination string, payload []byte, idempotencyKey string) error
}

type eventPublisher struct {
	rbClient *message_queue.Client

	// add other client
}

func NewEventPublisher(rbClient *message_queue.Client) EventPublisher {
	return &eventPublisher{rbClient: rbClient}
}

func (p *eventPublisher) Publish(ctx context.Context, eventType string, destination string, payload []byte, idempotencyKey string) error {
	if eventType == RabbitMQ.String() {
		exchange, routingKey := ParseRabbitMQDestination(destination)
		return p.rbClient.Publish(ctx, exchange, routingKey, payload, idempotencyKey)
	} else {
		logging.FromContext(ctx).WarnContext(ctx, fmt.Sprintf("eventType %s isn't supported", eventType))
		return nil
	}
}

type OutboxPublisher struct {
	logger           *slog.Logger
	publisher        EventPublisher
	outboxEventStore *store.OutboxEventStore
	batchSize        int32
	stopCh           chan struct{}
	doneCh           chan struct{}
}

func NewPublisher(logger *slog.Logger, eventPublisher EventPublisher, outboxEventStore *store.OutboxEventStore) *OutboxPublisher {
	return &OutboxPublisher{
		logger:           logger,
		publisher:        eventPublisher,
		outboxEventStore: outboxEventStore,
		batchSize:        50,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
}

func (p *OutboxPublisher) Start(ctx context.Context, pollInterval time.Duration) {
	defer close(p.doneCh)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			batchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := p.processOutboxEvent(batchCtx); err != nil {
				p.logger.Error("error processing outbox events", "err", err)
			}
			cancel()
		}

	}
}

// Stop signals the publisher loop to exit and blocks until it is completely stopped.
func (p *OutboxPublisher) Stop(shutdownCtx context.Context) error {
	close(p.stopCh)

	select {
	case <-p.doneCh:
	case <-shutdownCtx.Done():
		return fmt.Errorf("outbox publisher stop timed out: %w", shutdownCtx.Err())
	}

	// FLUSH REMAINING EVENTS: Run one final batch
	if err := p.processOutboxEvent(shutdownCtx); err != nil {
		p.logger.Error("failed to flush outbox events during shutdown", "err", err)
		return err
	}

	return nil
}

func (p *OutboxPublisher) processOutboxEvent(ctx context.Context) error {

	events, err := p.outboxEventStore.FetchPendingOutboxEvents(ctx, p.batchSize)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("fail to fetch pending outbox events - %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	for _, event := range events {

		err = p.publisher.Publish(ctx, event.EventType, event.Destination, event.Payload, event.IdempotencyKey)
		if err != nil {
			p.logger.ErrorContext(ctx, "failed to publish outbox event", "outbox_id", event.ID, "err", err)

			// mark retry or failed if attempt is exhausted
			_ = p.outboxEventStore.MarkOutboxEventFailed(ctx, event.ID, err.Error())

			continue
		}

		// Mark succeeded
		if err = p.outboxEventStore.MarkOutboxEventSucceeded(ctx, event.ID); err != nil {
			p.logger.ErrorContext(ctx, "failed to mark outbox event succeeded", "outbox_id", event.ID, "err", err)
		}
	}

	return nil
}
