package store

import (
	"context"
	"evsched/db"
	"evsched/db/sqlc"

	"github.com/google/uuid"
)

type PendingOutboxEvent struct {
	ID             uuid.UUID
	EventType      string
	Destination    string
	Payload        []byte
	IdempotencyKey string
	RetryCount     int32
}

type OutboxEventStore struct {
	querier *sqlc.Queries
}

func NewOutboxEventStore(db sqlc.DBTX) *OutboxEventStore {
	return &OutboxEventStore{
		querier: sqlc.New(db),
	}
}

func (s *OutboxEventStore) WithTx(tx sqlc.DBTX) *OutboxEventStore {
	return &OutboxEventStore{querier: sqlc.New(tx)}
}

func (s *OutboxEventStore) InsertOutboxEvent(ctx context.Context, eventType string, destination string, payload []byte,
	idempotencyKey string, maxRetries int) error {
	return s.querier.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
		EventType:      eventType,
		Destination:    destination,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		MaxRetries:     int32(maxRetries),
	})
}

func (s *OutboxEventStore) FetchPendingOutboxEvents(ctx context.Context, batchSize int32) ([]*PendingOutboxEvent, error) {

	rows, err := s.querier.FetchPendingOutboxEvents(ctx, batchSize)
	if err != nil {
		return nil, err
	}

	events := make([]*PendingOutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = &PendingOutboxEvent{
			ID:             row.ID,
			EventType:      row.EventType,
			Destination:    row.Destination,
			Payload:        row.Payload,
			IdempotencyKey: row.IdempotencyKey,
			RetryCount:     row.RetryCount,
		}
	}

	return events, nil
}

func (s *OutboxEventStore) MarkOutboxEventFailed(ctx context.Context, ID uuid.UUID, errMsg string) error {
	return s.querier.MarkOutboxEventFailed(ctx, sqlc.MarkOutboxEventFailedParams{
		ID:        ID,
		LastError: db.StringToPgText(errMsg),
	})
}

func (s *OutboxEventStore) MarkOutboxEventSucceeded(ctx context.Context, ID uuid.UUID) error {
	return s.querier.MarkOutboxEventSucceeded(ctx, ID)

}
