package kafka

import (
	"context"
	"time"
)

/*
type Plugin[T any] interface {
	ConsumeHandler(context.Context, []byte) error
	ProduceListener(context.Context)
	Payloads(ctx context.Context) (<-chan T, error)
}
*/

type ProducerPlugin[T any] interface {
	ProduceListener(context.Context)
	Payloads(ctx context.Context) (<-chan T, error)
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}

type ConsumerPlugin[T any] interface {
	ConsumeHandler(context.Context, []byte) error
	GetRunDuration() time.Duration
	GetIntervalDuration() time.Duration
}
