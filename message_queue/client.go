package message_queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type MessageQueueHeader string

const IdempotencyKey MessageQueueHeader = "Idempotency-Key"

type Client struct {
	url      string
	conn     *amqp.Connection
	ch       *amqp.Channel
	logger   *slog.Logger
	mu       sync.RWMutex
	isClosed bool
}

func NewClient(ctx context.Context, url string, logger *slog.Logger) (*Client, error) {
	client := &Client{
		url:    url,
		logger: logger,
	}

	if err := client.connect(ctx); err != nil {
		return nil, err
	}

	go client.handleReconnect(ctx)

	return client, nil
}

func (c *Client) connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}

	// Enable Publisher Confirms for delivery safety
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	c.conn = conn
	c.ch = ch
	return nil
}

func (c *Client) handleReconnect(ctx context.Context) {
	for {
		c.mu.RLock()
		if c.isClosed {
			c.mu.RUnlock()
			return
		}
		closeErr := c.conn.NotifyClose(make(chan *amqp.Error))
		c.mu.RUnlock()

		select {
		case <-ctx.Done():
			return
		case err := <-closeErr:
			if err != nil {
				c.logger.Warn("rabbitmq connection lost, attempting reconnect...", "error", err)
			}
			for {
				time.Sleep(3 * time.Second)
				if err := c.connect(ctx); err == nil {
					c.logger.Info("rabbitmq reconnected successfully")
					break
				}
			}
		}
	}
}

// Publish sends a message to an exchange
func (c *Client) Publish(ctx context.Context, exchange string, routingKey string, payload []byte, idempotencyKey string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ch == nil {
		return fmt.Errorf("rabbitmq channel is not available")
	}

	// Inject OpenTelemetry W3C trace context into AMQP headers
	headers := make(amqp.Table)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		headers[k] = v
	}
	headers[string(IdempotencyKey)] = idempotencyKey

	confirmation, err := c.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers:      headers,
			Body:         payload,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Block until RabbitMQ confirms receipt on disk/broker
	if !confirmation.Wait() {
		return fmt.Errorf("failed to receive publisher confirmation from rabbitmq")
	}

	return nil
}

func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isClosed = true

	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
