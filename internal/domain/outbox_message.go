package domain

import "time"

// OutboxMessage represents a domain event that must be reliably
// delivered to an external system. It is persisted atomically with
// the aggregate change that produced it.
type OutboxMessage struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

func NewOutboxMessage(id, aggregateType, aggregateID, eventType string, payload []byte) *OutboxMessage {
	return &OutboxMessage{
		ID:            id,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     time.Now(),
	}
}
