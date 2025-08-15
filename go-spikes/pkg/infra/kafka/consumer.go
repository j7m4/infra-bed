package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type ConsumerEngine[T any] struct {
	consumer         *k.Consumer
	connectionConfig cfg.KafkaConfig
	plugin           ConsumerPlugin[T]
	tracer           trace.Tracer
}

func NewConsumerEngine[T any](cfg cfg.KafkaConfig, plugin ConsumerPlugin[T]) (*ConsumerEngine[T], error) {
	kafkaConfig := &k.ConfigMap{
		"bootstrap.servers":       strings.Join(cfg.Brokers, ","),
		"group.id":                cfg.ConsumerGroup,
		"client.id":               fmt.Sprintf("%s-consumer", cfg.ConsumerGroup),
		"auto.offset.reset":       cfg.ConsumerConfig.AutoOffsetReset,
		"enable.auto.commit":      false,
		"auto.commit.interval.ms": int(cfg.ConsumerConfig.AutoCommitInterval.Milliseconds()),
		"session.timeout.ms":      int(cfg.ConsumerConfig.SessionTimeout.Milliseconds()),
		"max.poll.interval.ms":    int(cfg.ConsumerConfig.MaxPollInterval.Milliseconds()),
	}
	consumer, err := k.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &ConsumerEngine[T]{
		consumer:         consumer,
		connectionConfig: cfg,
		plugin:           plugin,
		tracer:           otel.Tracer("KafkaConsumer"),
	}, nil
}

func (c *ConsumerEngine[T]) Close() {
	if err := c.consumer.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close consumer")
	} else {
		log.Info().Msg("Consumer closed successfully")
	}
}

func (c *ConsumerEngine[T]) Run(ctx context.Context) {
	var err error

	if err = c.Consume(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to consume Kafka messages")
		return
	}
}

func (c *ConsumerEngine[T]) Subscribe() error {
	err := c.consumer.SubscribeTopics([]string{c.connectionConfig.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.connectionConfig.Topic, err)
	}
	return nil
}

func (c *ConsumerEngine[T]) Consume(ctx context.Context) error {

	log := logger.Ctx(ctx)
	log.Info().
		Str("topic", c.connectionConfig.Topic).
		Str("group", c.connectionConfig.ConsumerGroup).
		Msg("Starting consumer")

	count := 0
	batchSize := 10000
	batchConsumeMsg := fmt.Sprintf("kafka.consume.batch: %d", batchSize)
	settings := DefaultConsumeTestSettings()
	runTimer := infra.StartRunTimer(ctx, c.plugin)
	intervalTicker := infra.StartIntervalTicker(ctx, c.plugin)

	if err := c.Subscribe(); err != nil {
		return err
	}

	batchCtx, batchSpan := c.tracer.Start(ctx, batchConsumeMsg)
	log = logger.Ctx(batchCtx)
	defer batchSpan.End()

	for {
		select {
		case <-intervalTicker.C:
			log.Trace().Msg("Consumer tick")
		}
		select {
		case <-ctx.Done():
			log.Info().
				Int("consume_count", count).
				Msg("consume context done")
			return ctx.Err()
		case <-runTimer:
			log.Info().
				Int("consume_count", count).
				Dur("runTimer", settings.TestTimeout).
				Msg("consume runTimer reached")
			return nil
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(k.Error).Code() == k.ErrTimedOut {
					continue
				}
				log.Error().Err(err).Msg("Error reading message")
				continue
			}
			if msg == nil {
				log.Warn().
					Msg("Received nil message, skipping")
				continue
			}
			if err := c.plugin.ConsumeHandler(batchCtx, msg.Value); err != nil {
				log.Error().
					Err(err).
					Str("key", string(msg.Key)).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Failed to unmarshal payload")
				continue
			}
			count++
			if count%batchSize == 0 {
				// close out old and create new span
				batchSpan.End()
				batchCtx, batchSpan = c.tracer.Start(ctx, batchConsumeMsg)
				log = logger.Ctx(batchCtx)
				log.Info().
					Int("count", count).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg(batchConsumeMsg)
			}
		}
	}
}

func (c *ConsumerEngine[T]) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.connectionConfig.Topic, false, 5000)
}
