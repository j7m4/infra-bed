package kafka

import (
	"context"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ProducerPlugin[T any] interface {
	ProduceMessageListener(ctx context.Context, engine ProducerEngine[T], message *k.Message) error
	Payloads(ctx context.Context) (<-chan T, error)
	GetInitialDelayDuration() time.Duration
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}

type ConsumerPlugin[T any] interface {
	ConsumeMessageHandler(ctx context.Context, engine ConsumerEngine[T], message *k.Message) error
	GetInitialDelayDuration() time.Duration
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}
