package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-infra-spikes/go-spikes/pkg/logger"
)

type Consumer struct {
	consumer *k.Consumer
	config   *KafkaConfig
}

type PayloadHandler func(ctx context.Context, payload *Payload) error

func NewConsumer(cfg *KafkaConfig) (*Consumer, error) {
	if cfg == nil {
		cfg = DefaultKafkaConfig()
	}

	configMap := k.ConfigMap{
		"bootstrap.servers":       strings.Join(cfg.Brokers, ","),
		"group.id":                cfg.ConsumerGroup,
		"client.id":               fmt.Sprintf("%s-client", cfg.ConsumerGroup),
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      true,
		"auto.commit.interval.ms": 5000,
		"session.timeout.ms":      10000,
		"max.poll.interval.ms":    300000,
	}

	consumer, err := k.NewConsumer(&configMap)
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

func (c *Consumer) Consume(ctx context.Context, handler PayloadHandler) error {
	log := logger.Ctx(ctx)
	log.Info().
		Str("topic", c.config.Topic).
		Str("group", c.config.ConsumerGroup).
		Msg("Starting consumer")

	if err := c.Subscribe(); err != nil {
		return err
	}

	messagesConsumed := 0
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Int("total_messages", messagesConsumed).Msg("Consumer shutting down")
			return ctx.Err()
		case <-ticker.C:
			log.Info().Int("messages_consumed", messagesConsumed).Msg("Consumer status")
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

			if handler != nil {
				if err := handler(ctx, &payload); err != nil {
					log.Error().
						Err(err).
						Str("entity_id", payload.EntityID).
						Msg("Handler error")
					continue
				}
			}

			messagesConsumed++
			if messagesConsumed%1000 == 0 {
				log.Info().
					Int("count", messagesConsumed).
					Int32("partition", msg.TopicPartition.Partition).
					Int64("offset", int64(msg.TopicPartition.Offset)).
					Msg("Consumed messages")
			}
		}
	}
}

func (c *Consumer) ConsumeN(ctx context.Context, n int, handler PayloadHandler) ([]*Payload, error) {
	log := logger.Ctx(ctx)

	if err := c.Subscribe(); err != nil {
		return nil, err
	}

	payloads := make([]*Payload, 0, n)
	timeout := time.After(30 * time.Second)

	for len(payloads) < n {
		select {
		case <-ctx.Done():
			return payloads, ctx.Err()
		case <-timeout:
			log.Warn().
				Int("expected", n).
				Int("received", len(payloads)).
				Msg("Timeout waiting for messages")
			return payloads, nil
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
				log.Error().Err(err).Msg("Failed to unmarshal payload")
				continue
			}

			if handler != nil {
				if err := handler(ctx, &payload); err != nil {
					log.Error().Err(err).Msg("Handler error")
					continue
				}
			}

			payloads = append(payloads, &payload)
		}
	}

	return payloads, nil
}

func (c *Consumer) GetMetadata() (*k.Metadata, error) {
	return c.consumer.GetMetadata(&c.config.Topic, false, 5000)
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
