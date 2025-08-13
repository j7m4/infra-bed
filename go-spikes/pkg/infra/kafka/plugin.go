package kafka

import (
	"context"
)

type Plugin[T any] interface {
	ConsumeHandler(context.Context, []byte) error
	PublishListener(context.Context)
	GeneratePayloads(ctx context.Context) (<-chan T, error)
}
