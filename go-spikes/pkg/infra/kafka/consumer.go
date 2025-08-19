package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/infra-bed/go-spikes/pkg/config"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/rs/zerolog"
	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
)

type ConsumerEngine[T any] interface {
	Run(ctx context.Context) error
	Close()
	AcceptMessage(message *k.Message) error
	RejectMessage(message *k.Message) error
	GetMetadata() (*k.Metadata, error)
}

func NewConsumerEngine[T any](cfg cfg.KafkaConfig, plugin ConsumerPlugin[T]) (ConsumerEngine[T], error) {
	var err error
	var consumer *k.Consumer

	kafkaConfig := &k.ConfigMap{
		"bootstrap.servers":    strings.Join(cfg.Brokers, ","),
		"client.id":            cfg.ConsumerConfig.ClientId,
		"group.id":             cfg.ConsumerConfig.ConsumerGroup,
		"auto.offset.reset":    cfg.ConsumerConfig.AutoOffsetReset,
		"enable.auto.commit":   cfg.ConsumerConfig.AutoCommitEnabled,
		"session.timeout.ms":   int(cfg.ConsumerConfig.SessionTimeout.Milliseconds()),
		"max.poll.interval.ms": int(cfg.ConsumerConfig.MaxPollInterval.Milliseconds()),
	}
	if int(cfg.ConsumerConfig.AutoCommitInterval.Milliseconds()) > 0 {
		if err = kafkaConfig.SetKey("auto.commit.interval.ms", int(cfg.ConsumerConfig.AutoCommitInterval.Milliseconds())); err != nil {
			return nil, err
		}
	}
	if cfg.ConsumerConfig.IsolationLevel != "" {
		if err = kafkaConfig.SetKey("isolation.level", cfg.ConsumerConfig.IsolationLevel); err != nil {
			return nil, err
		}
	}
	consumer, err = k.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	logBatchSize := cfg.ConsumerConfig.LogBatchSize
	if logBatchSize <= 0 {
		logBatchSize = config.DefaultLogBatchSize
	}

	return &consumerEngineImpl[T]{
		consumer:         consumer,
		connectionConfig: cfg,
		plugin:           plugin,
		// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
		tracer: otel.Tracer("KafkaConsumer"),
		// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
		logBatchSize: logBatchSize,
		log:          logger.Get(),
	}, nil
}

type consumerEngineImpl[T any] struct {
	consumer         *k.Consumer
	connectionConfig cfg.KafkaConfig
	plugin           ConsumerPlugin[T]
	tracer           trace.Tracer
	logBatchSize     int
	log              *zerolog.Logger
}

func (c *consumerEngineImpl[T]) Close() {
	if err := c.consumer.Close(); err != nil {
		c.log.Error().Err(err).Msg("Failed to close consumer")
	} else {
		c.log.Info().Msg("Consumer closed successfully")
	}
}

func (c *consumerEngineImpl[T]) AcceptMessage(message *k.Message) error {
	if message == nil {
		c.log.Warn().Msg("Received nil message, cannot accept")
		return nil
	}
	// commit manually, if not auto-commit enabled
	if !c.connectionConfig.ConsumerConfig.AutoCommitEnabled {
		if _, err := c.consumer.CommitMessage(message); err != nil {
			c.log.Error().
				Err(err).
				Int32("partition", message.TopicPartition.Partition).
				Int64("offset", int64(message.TopicPartition.Offset)).
				Msg("Failed to commit message")
		}
	}
	return nil
}

func (c *consumerEngineImpl[T]) RejectMessage(message *k.Message) error {
	if message == nil {
		c.log.Warn().Msg("Received nil message, cannot reject")
		return nil
	}
	return nil
}

func (c *consumerEngineImpl[T]) Run(ctx context.Context) error {

	if c.plugin.GetInitialDelayDuration() > 0 {
		select {
		case <-infra.StartInitialDelayTimer(ctx, c.plugin):
			c.log.Trace().Msg("Consumer initialDelay")
		}
	}
	c.log.Trace().Msg("Consumer post-initialDelay")

	var msg *k.Message
	var err error

	c.log.Info().
		Str("topic", c.connectionConfig.Topic).
		Str("group", c.connectionConfig.ConsumerConfig.ConsumerGroup).
		Msg("Starting consumer")

	count := 0
	batchConsumeMsg := fmt.Sprintf("kafka.consume.batch: %d", c.logBatchSize)
	runTimer := infra.StartRunTimer(ctx, c.plugin)
	intervalTimer := infra.NewIntervalTimer(ctx, c.plugin)

	if err = c.consumer.SubscribeTopics([]string{c.connectionConfig.Topic}, nil); err != nil {
		return err
	}

	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	batchCtx, batchSpan := c.tracer.Start(ctx, batchConsumeMsg)
	batchLog := logger.Ctx(batchCtx)
	defer batchSpan.End()
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka

	for {
		intervalTimer.NextTickWait()
		select {
		case <-ctx.Done():
			if msg != nil {
				batchLog.Info().
					Int("count", count).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg(batchConsumeMsg)
			}
			batchLog.Info().Int("count", count).Msg("consume context done")
			return ctx.Err()
		case <-runTimer:
			if msg != nil {
				batchLog.Info().
					Int("count", count).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg(batchConsumeMsg)
			}
			batchLog.Info().Int("count", count).Msg("consume runTimer reached")
			return nil
		default:
			batchLog.Trace().Msg("Consumer reading message")
			msg, err = c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(k.Error).Code() == k.ErrTimedOut {
					continue
				}
				batchLog.Error().Err(err).Msg("Error reading message")
				continue
			}
			if msg == nil {
				batchLog.Warn().
					Msg("Received nil message, skipping")
				continue
			}
			if err := c.plugin.ConsumeMessageHandler(batchCtx, c, msg); err != nil {
				batchLog.Error().
					Err(err).
					Str("key", string(msg.Key)).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Failed to unmarshal payload")
				continue
			}
			count++
			if count%c.logBatchSize == 0 {
				// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
				// close out old and create new span
				batchSpan.End()
				batchCtx, batchSpan = c.tracer.Start(ctx, batchConsumeMsg)
				// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
				batchLog.Info().
					Int("count", count).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg(batchConsumeMsg)
			}
		}
	}
}

func (c *consumerEngineImpl[T]) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.connectionConfig.Topic, false, 5000)
}
