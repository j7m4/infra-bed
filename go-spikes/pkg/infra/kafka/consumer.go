package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/infra-bed/go-spikes/pkg/config"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/infra-bed/go-spikes/pkg/model"
	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
)

type ConsumerJob[T any] interface {
	Run(ctx context.Context)
	Close()
	AcceptMessage(ctx context.Context, message *k.Message) error
	RejectMessage(ctx context.Context, message *k.Message) error
	GetMetadata() (*k.Metadata, error)
	GetPlugin() model.Plugin
}

func NewConsumerJob[T any](cfg cfg.KafkaConfig, plugin ConsumerPlugin[T]) (model.Job, error) {
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

	return &consumerJobImpl[T]{
		consumer:         consumer,
		connectionConfig: cfg,
		plugin:           plugin,
		// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
		tracer: otel.Tracer("KafkaConsumer"),
		// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
		logBatchSize: logBatchSize,
	}, nil
}

type consumerJobImpl[T any] struct {
	consumer         *k.Consumer
	connectionConfig cfg.KafkaConfig
	plugin           ConsumerPlugin[T]
	tracer           trace.Tracer
	logBatchSize     int
}

func (c *consumerJobImpl[T]) GetPlugin() model.Plugin {
	return c.plugin
}

func (c *consumerJobImpl[T]) Close() {
	log := logger.Get()
	if err := c.consumer.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close consumer")
	} else {
		log.Info().Msg("Consumer closed successfully")
	}
}

func (c *consumerJobImpl[T]) AcceptMessage(ctx context.Context, message *k.Message) error {
	log := logger.Ctx(ctx)
	if message == nil {
		log.Warn().Msg("Received nil message, cannot accept")
		return nil
	}
	// commit manually, if not auto-commit enabled
	if !c.connectionConfig.ConsumerConfig.AutoCommitEnabled {
		if _, err := c.consumer.CommitMessage(message); err != nil {
			log.Error().
				Err(err).
				Int32("partition", message.TopicPartition.Partition).
				Int64("offset", int64(message.TopicPartition.Offset)).
				Msg("Failed to commit message")
		}
	}
	return nil
}

func (c *consumerJobImpl[T]) RejectMessage(ctx context.Context, message *k.Message) error {
	log := logger.Ctx(ctx)
	if message == nil {
		log.Warn().Msg("Received nil message, cannot reject")
		return nil
	}
	return nil
}

func (c *consumerJobImpl[T]) Run(ctx context.Context) {
	log := logger.Ctx(ctx)

	var msg *k.Message
	var err error

	log.Info().
		Str("topic", c.connectionConfig.Topic).
		Str("group", c.connectionConfig.ConsumerConfig.ConsumerGroup).
		Msg("Starting consumer")

	count := 0
	batchConsumeMsg := fmt.Sprintf("kafka.consume.batch: %d", c.logBatchSize)
	intervalTimer := model.NewIntervalTimer(ctx, c.plugin)

	if err = c.consumer.SubscribeTopics([]string{c.connectionConfig.Topic}, nil); err != nil {
		log.Error().
			Err(err).
			Str("topic", c.connectionConfig.Topic).
			Str("group", c.connectionConfig.ConsumerConfig.ConsumerGroup).
			Msg("Failed to subscribe to topic")
		return
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
			return
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

func (c *consumerJobImpl[T]) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.connectionConfig.Topic, false, 5000)
}
