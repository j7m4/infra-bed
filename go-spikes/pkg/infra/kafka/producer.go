package kafka

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

type Producer[T any] struct {
	producer     *k.Producer
	config       cfg.KafkaConfig
	deliveryChan chan k.Event
}

func RunProducer[T any](ctx context.Context, cfg cfg.KafkaConfig, plugin Plugin[T]) {
	var err error
	var producer *Producer[T]
	var payloads <-chan T
	var count int
	var log = logger.WithContext(ctx)

	if producer, err = NewProducer[T](cfg); err != nil {
		log.Error().Err(err).Msg("Failed to create Kafka producer")
		return
	}
	defer producer.Close()
	if payloads, err = plugin.GeneratePayloads(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to generate Kafka payloads")
		return
	}
	if count, err = producer.ProduceBatch(ctx, payloads); err != nil {
		log.Error().Err(err).Msg("Failed to produce Kafka messages")
		return
	}
	log.Info().Int("count", count).Msg("Finished producing Kafka messages")
}

func NewProducer[T any](cfg cfg.KafkaConfig) (*Producer[T], error) {
	configMap := &k.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Brokers, ","),
		"client.id":         fmt.Sprintf("%s-producer", cfg.ConsumerGroup),
		"acks":              cfg.ProducerConfig.Acks,
		"retries":           10,
		"linger.ms":         10,
		"compression.type":  cfg.ProducerConfig.CompressionType,
	}
	logger.Get().Info().Any("config", cfg).Msg("created consumer")
	producer, err := k.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer[T]{
		producer:     producer,
		config:       cfg,
		deliveryChan: make(chan k.Event, 1000),
	}, nil
}

func (p *Producer[T]) ProducePayload(ctx context.Context, payload T) error {
	log := logger.Ctx(ctx)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	key := sha256.New().Sum(data)
	msg := &k.Message{
		TopicPartition: k.TopicPartition{
			Topic:     &p.config.Topic,
			Partition: k.PartitionAny,
		},
		Key:   key,
		Value: data,
	}

	err = p.producer.Produce(msg, p.deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	select {
	case e := <-p.deliveryChan:
		m := e.(*k.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
		log.Debug().
			Int32("partition", m.TopicPartition.Partition).
			Int64("offset", int64(m.TopicPartition.Offset)).
			Msg("Message delivered")
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (p *Producer[T]) ProducePayloadAsync(payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	key := sha256.New().Sum(data)
	msg := &k.Message{
		TopicPartition: k.TopicPartition{
			Topic:     &p.config.Topic,
			Partition: k.PartitionAny,
		},
		Key:   key,
		Value: data,
	}

	return p.producer.Produce(msg, nil)
}

func (p *Producer[T]) ProduceBatch(ctx context.Context, payloadChan <-chan T) (int, error) {
	log := logger.Ctx(ctx)

	go p.handleDeliveryReports(ctx)

	count := 0
	for payload := range payloadChan {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		default:
			if err := p.ProducePayloadAsync(payload); err != nil {
				log.Error().Err(err).Msg("Failed to produce payload")
				continue
			}
			count++
			if count%1000 == 0 {
				log.Info().Int("count", count).Msg("Produced payloads")
			}
		}
	}

	p.producer.Flush(15 * 1000)
	log.Info().Int("total", count).Msg("Finished producing payloads")
	return count, nil
}

func (p *Producer[T]) handleDeliveryReports(ctx context.Context) {
	log := logger.Ctx(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-p.producer.Events():
			switch ev := e.(type) {
			case *k.Message:
				if ev.TopicPartition.Error != nil {
					log.Error().
						Err(ev.TopicPartition.Error).
						Str("key", string(ev.Key)).
						Msg("Delivery failed")
				}
			case k.Error:
				log.Error().
					Err(ev).
					Int("code", int(ev.Code())).
					Msg("Kafka error")
			}
		}
	}
}

func (p *Producer[T]) Close() {
	p.producer.Close()
}
