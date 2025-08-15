package kafka

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

type ProducerEngine[T any] struct {
	producer     *k.Producer
	config       cfg.KafkaConfig
	deliveryChan chan k.Event
	plugin       ProducerPlugin[T]
	tracer       trace.Tracer
}

func NewProducerEngine[T any](cfg cfg.KafkaConfig, plugin ProducerPlugin[T]) (*ProducerEngine[T], error) {
	configMap := &k.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Brokers, ","),
		"client.id":         fmt.Sprintf("%s-producer", cfg.ConsumerGroup),
		"acks":              cfg.ProducerConfig.Acks,
		"retries":           10,
		"linger.ms":         10,
		"compression.type":  cfg.ProducerConfig.CompressionType,
	}

	producer, err := k.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &ProducerEngine[T]{
		producer:     producer,
		config:       cfg,
		deliveryChan: make(chan k.Event, 1000),
		plugin:       plugin,
		tracer:       otel.Tracer("KafkaProducer"),
	}, nil
}

func (p *ProducerEngine[T]) Run(ctx context.Context) {
	var err error
	var payloads <-chan T
	var log = logger.WithContext(ctx)

	if payloads, err = p.plugin.Payloads(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to generate Payloads")
		return
	}
	if err = p.producePayloads(ctx, payloads); err != nil {
		log.Error().Err(err).Msg("Failed to produce Kafka messages")
		return
	}
}

func (p *ProducerEngine[T]) Close() {
	p.producer.Close()
	log.Info().Msg("Producer closed successfully")
}

func (p *ProducerEngine[T]) GetRunDuration() time.Duration {
	return 120 * time.Second
}

func (p *ProducerEngine[T]) producePayloads(ctx context.Context, payloadChan <-chan T) error {
	log := logger.Ctx(ctx)

	go p.handleProducerEvents(ctx)

	count := 0
	batchSize := 10000
	batchProduceMsg := fmt.Sprintf("kafka.produce.batch: %d", batchSize)

	batchCtx, batchSpan := p.tracer.Start(ctx, batchProduceMsg)
	log = logger.Ctx(batchCtx)
	defer batchSpan.End()

	runTimer := infra.StartRunTimer(ctx, p.plugin)
	intervalTicker := infra.StartIntervalTicker(ctx, p.plugin)

	for payload := range payloadChan {
		select {
		case <-intervalTicker.C:
			log.Trace().Msg("Producer tick")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-runTimer:
			return ctx.Err()
		default:
			if err := p.producePayloadAsync(payload); err != nil {
				log.Error().Err(err).Msg("Failed to produce payload")
				continue
			}
			count++
			if count%batchSize == 0 {
				// close out old and create new span
				batchSpan.End()
				batchCtx, batchSpan = p.tracer.Start(ctx, batchProduceMsg)
				log.Info().Int("count", count).Msg(batchProduceMsg)
			}
		}
	}

	p.producer.Flush(15 * 1000)
	log.Info().Int("count", count).Msg("Finished producing payloads")
	return nil
}

// producePayloadAsync produces a single payload asynchronously.
// It marshals the payload to JSON, computes a SHA-256 hash for the key.
// An alternative would be to produce messages transactionally.
func (p *ProducerEngine[T]) producePayloadAsync(payload interface{}) error {
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

// handleProducerEvents is limited to handling error-like Events from the producer.
func (p *ProducerEngine[T]) handleProducerEvents(ctx context.Context) {
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
