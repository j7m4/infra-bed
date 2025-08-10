package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-infra-spikes/go-spikes/pkg/logger"
)

type Producer struct {
	producer     *k.Producer
	config       *KafkaConfig
	deliveryChan chan k.Event
}

func NewProducer(cfg *KafkaConfig) (*Producer, error) {
	if cfg == nil {
		cfg = DefaultKafkaConfig()
	}

	configMap := k.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Brokers, ","),
		"client.id":         "payload-producer",
		"acks":              "all",
		"retries":           10,
		"linger.ms":         10,
		"compression.type":  "snappy",
	}

	producer, err := k.NewProducer(&configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{
		producer:     producer,
		config:       cfg,
		deliveryChan: make(chan k.Event, 1000),
	}, nil
}

func (p *Producer) ProducePayload(ctx context.Context, payload *Payload) error {
	log := logger.Ctx(ctx)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &k.Message{
		TopicPartition: k.TopicPartition{
			Topic:     &p.config.Topic,
			Partition: k.PartitionAny,
		},
		Key:   []byte(payload.EntityID),
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
			Str("entity_id", payload.EntityID).
			Int32("partition", m.TopicPartition.Partition).
			Int64("offset", int64(m.TopicPartition.Offset)).
			Msg("Message delivered")
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (p *Producer) ProducePayloadAsync(payload *Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &k.Message{
		TopicPartition: k.TopicPartition{
			Topic:     &p.config.Topic,
			Partition: k.PartitionAny,
		},
		Key:   []byte(payload.EntityID),
		Value: data,
	}

	return p.producer.Produce(msg, nil)
}

func (p *Producer) ProduceBatch(ctx context.Context, payloadChan <-chan *Payload) (int, error) {
	log := logger.Ctx(ctx)

	go p.handleDeliveryReports(ctx)

	count := 0
	for payload := range payloadChan {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		default:
			if err := p.ProducePayloadAsync(payload); err != nil {
				log.Error().Err(err).Str("entity_id", payload.EntityID).Msg("Failed to produce payload")
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

func (p *Producer) handleDeliveryReports(ctx context.Context) {
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

func (p *Producer) Close() {
	p.producer.Close()
}
