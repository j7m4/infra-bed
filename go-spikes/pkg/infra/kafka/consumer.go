package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra/kafka/entityrepo"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/rs/zerolog/log"
)

type Consumer[T any] struct {
	consumer *k.Consumer
	config   cfg.KafkaConfig
	plugin   Plugin[T]
}

type PayloadHandler func(ctx context.Context, payload *entityrepo.Payload) error

func RunConsumer[T any](ctx context.Context, cfg cfg.KafkaConfig, plugin Plugin[T]) {
	var consumer *Consumer[T]
	var err error
	var count int

	if consumer, err = NewConsumer[T](cfg, plugin); err != nil {
		log.Error().Err(err).Msg("Failed to create Kafka consumer")
		return
	}
	defer func(consumer *Consumer[T]) {
		err := consumer.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close Kafka consumer")
		} else {
			log.Info().Msg("Kafka consumer closed successfully")
		}
	}(consumer)

	if count, err = consumer.Consume(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to consume Kafka messages")
		return
	}

	log.Info().Int("count", count).Msg("Finished consuming Kafka messages")
}

func NewConsumer[T any](cfg cfg.KafkaConfig, plugin Plugin[T]) (*Consumer[T], error) {
	consumer, err := k.NewConsumer(&k.ConfigMap{
		"bootstrap.servers":       strings.Join(cfg.Brokers, ","),
		"group.id":                cfg.ConsumerGroup,
		"client.id":               fmt.Sprintf("%s-consumer", cfg.ConsumerGroup),
		"auto.offset.reset":       cfg.ConsumerConfig.AutoOffsetReset,
		"enable.auto.commit":      false,
		"auto.commit.interval.ms": int(cfg.ConsumerConfig.AutoCommitInterval.Milliseconds()),
		"session.timeout.ms":      int(cfg.ConsumerConfig.SessionTimeout.Milliseconds()),
		"max.poll.interval.ms":    int(cfg.ConsumerConfig.MaxPollInterval.Milliseconds()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}
	logger.Get().Info().Any("config", cfg).Msg("created consumer")

	return &Consumer[T]{
		consumer: consumer,
		config:   cfg,
		plugin:   plugin,
	}, nil
}

func (c *Consumer[T]) Subscribe() error {
	err := c.consumer.SubscribeTopics([]string{c.config.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.config.Topic, err)
	}
	return nil
}

func (c *Consumer[T]) Consume(ctx context.Context) (int, error) {
	log := logger.Ctx(ctx)
	//log := logger.Get()
	log.Info().
		Str("topic", c.config.Topic).
		Str("group", c.config.ConsumerGroup).
		Msg("Starting consumer")

	count := 0
	settings := DefaultConsumeTestSettings()
	testTimeout := time.After(settings.TestTimeout)

	if err := c.Subscribe(); err != nil {
		return count, err
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Int("consume_count", count).
				Msg("consume context done")
			return count, ctx.Err()
		case <-testTimeout:
			log.Info().
				Int("consume_count", count).
				Dur("testTimeout", settings.TestTimeout).
				Msg("consume testTimeout reached")
			return count, nil
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
			if err := c.plugin.ConsumeHandler(ctx, msg.Value); err != nil {
				log.Error().
					Err(err).
					Str("key", string(msg.Key)).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Failed to unmarshal payload")
				continue
			}
			count++
			if count%1000 == 0 {
				log.Info().
					Int("count", count).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Consumed messages")
			}
		}
	}
}

func (c *Consumer[T]) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.config.Topic, false, 5000)
}

func (c *Consumer[T]) Close() error {
	return c.consumer.Close()
}
