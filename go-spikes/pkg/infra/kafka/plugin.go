package kafka

import (
	"context"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ProducerPlugin[T any] interface {
	GetName() string
	ProduceMessageListener(ctx context.Context, engine ProducerJob[T], message *k.Message) error
	Payloads(ctx context.Context) (<-chan T, error)
	GetInitialDelayDuration() time.Duration
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}

type ConsumerPlugin[T any] interface {
	GetName() string
	ConsumeMessageHandler(ctx context.Context, engine ConsumerJob[T], message *k.Message) error
	GetInitialDelayDuration() time.Duration
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}
