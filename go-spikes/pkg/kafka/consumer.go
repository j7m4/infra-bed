package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-infra-spikes/go-spikes/pkg/logger"
)

type Consumer struct {
	consumer *k.Consumer
	config   *ConnectionConfig
}

type PayloadHandler func(ctx context.Context, payload *Payload) error

func NewConsumer(cfg *ConnectionConfig) (*Consumer, error) {
	if cfg == nil {
		cfg = DefaultConnectionConfig()
	}

	configMap := DefaultConsumerConfigMap(cfg)

	consumer, err := k.NewConsumer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		config:   cfg,
	}, nil
}

func (c *Consumer) Subscribe() error {
	err := c.consumer.SubscribeTopics([]string{c.config.Topic}, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.config.Topic, err)
	}
	return nil
}

func (c *Consumer) Consume(ctx context.Context, handler PayloadHandler) (int, error) {
	log := logger.Ctx(ctx)
	log.Info().
		Str("topic", c.config.Topic).
		Str("group", c.config.ConsumerGroup).
		Msg("Starting consumer")

	count := 0
	var lastMessageTime time.Time
	settings := DefaultConsumeTestSettings()
	testTimeout := time.After(settings.TestTimeout)

	if err := c.Subscribe(); err != nil {
		return count, err
	}

	for {
		if !lastMessageTime.IsZero() {
			if time.Since(lastMessageTime) > settings.SilenceTimeout {
				log.Info().
					Int("consume_count", count).
					Dur("silence_timeout", settings.SilenceTimeout).
					Msg("Message no longer received, ending consume")
				return count, nil
			}
		}
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

			var payload Payload
			if err := json.Unmarshal(msg.Value, &payload); err != nil {
				log.Error().
					Err(err).
					Str("key", string(msg.Key)).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Failed to unmarshal payload")
				continue
			}
			lastMessageTime = time.Now()
			if handler != nil {
				if err := handler(ctx, &payload); err != nil {
					log.Error().
						Err(err).
						Str("entity_id", payload.EntityID).
						Msg("Handler error")
					continue
				}
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

func (c *Consumer) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.config.Topic, false, 5000)
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
