package kafka

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/infra-bed/go-spikes/pkg/config"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/rs/zerolog/log"
	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
)

type ProducerEngine[T any] interface {
	Run(ctx context.Context)
	Close()
}

func NewProducerEngine[T any](cfg cfg.KafkaConfig, plugin ProducerPlugin[T]) (ProducerEngine[T], error) {
	configMap := &k.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Brokers, ","),
		"client.id":         cfg.ProducerConfig.ClientId,
		"acks":              cfg.ProducerConfig.Acks,
		"retries":           10,
		"linger.ms":         10,
		"compression.type":  cfg.ProducerConfig.CompressionType,
	}

	producer, err := k.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	logBatchSize := cfg.ProducerConfig.LogBatchSize
	if logBatchSize <= 0 {
		logBatchSize = config.DefaultLogBatchSize
	}

	return &producerEngineImpl[T]{
		producer:     producer,
		config:       cfg,
		deliveryChan: make(chan k.Event, 1000),
		plugin:       plugin,
		// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
		tracer: otel.Tracer("KafkaProducer"),
		// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
		logBatchSize: logBatchSize,
	}, nil
}

type producerEngineImpl[T any] struct {
	producer     *k.Producer
	config       cfg.KafkaConfig
	deliveryChan chan k.Event
	plugin       ProducerPlugin[T]
	tracer       trace.Tracer
	logBatchSize int
}

func (p *producerEngineImpl[T]) Run(ctx context.Context) {
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

func (p *producerEngineImpl[T]) Close() {
	p.producer.Close()
	close(p.deliveryChan)
	log.Info().Msg("Producer closed successfully")
}

func (p *producerEngineImpl[T]) producePayloads(ctx context.Context, payloadChan <-chan T) error {
	log := logger.Ctx(ctx)

	if p.plugin.GetInitialDelayDuration() > 0 {
		select {
		case <-infra.StartInitialDelayTimer(ctx, p.plugin):
			log.Trace().Msg("Producer initialDelay")
		}
		log.Trace().Msg("Producer post-initialDelay")
	}

	go p.fallbackProducerEventHandler(ctx)
	go p.messageDeliveryEventHandler(ctx)

	count := 0
	batchProduceMsg := fmt.Sprintf("kafka.produce.batch: %d", p.logBatchSize)

	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	batchCtx, batchSpan := p.tracer.Start(ctx, batchProduceMsg)
	log = logger.Ctx(batchCtx)
	defer batchSpan.End()
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka

	runTimer := infra.StartRunTimer(ctx, p.plugin)
	intervalTimer := infra.NewIntervalTimer(ctx, p.plugin)

	for payload := range payloadChan {
		intervalTimer.NextTickWait()
		select {
		case <-ctx.Done():
			log.Info().Int("count", count).Msg(batchProduceMsg)
			log.Info().Msg("producer context done")
			return ctx.Err()
		case <-runTimer:
			log.Info().Int("count", count).Msg(batchProduceMsg)
			log.Info().Msg("producer run time elapsed")
			return ctx.Err()
		default:
			if err := p.producePayloadAsync(payload); err != nil {
				log.Error().Err(err).Msg("Failed to produce payload")
				continue
			}
			count++
			if count%p.logBatchSize == 0 {
				// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
				// close out old and create new span
				batchSpan.End()
				batchCtx, batchSpan = p.tracer.Start(ctx, batchProduceMsg)
				// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka
				log.Info().Int("count", count).Msg(batchProduceMsg)
			}
		}
	}
	log.Info().Int("count", count).Msg(batchProduceMsg)

	p.producer.Flush(15 * 1000)
	log.Info().Int("count", count).Msg("Finished producing payloads")
	return nil
}

// producePayloadAsync produces a single payload asynchronously.
// It marshals the payload to JSON, computes a SHA-256 hash for the key.
// An alternative would be to produce messages transactionally.
func (p *producerEngineImpl[T]) producePayloadAsync(payload interface{}) error {
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

	return p.producer.Produce(msg, p.deliveryChan)
}

func (p *producerEngineImpl[T]) messageDeliveryEventHandler(ctx context.Context) {
	log := logger.Ctx(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-p.deliveryChan:
			switch ev := e.(type) {
			case *k.Message:
				if ev.TopicPartition.Error != nil {
					log.Error().
						Err(ev.TopicPartition.Error).
						Str("key", string(ev.Key)).
						Msg("Delivery failed")
				} else {
					err := p.plugin.ProduceMessageListener(ctx, p, ev)
					if err != nil {
						log.Error().Err(err).Msg("Failed on ProduceMessageListener")
					}
				}
			}
		}
	}
}

// fallbackProducerEventHandler is limited to handling error-like Events from the producer.
// It will receive events from Produce() when there is no delivery channel specified.
// It will log delivery failures and Kafka errors.
func (p *producerEngineImpl[T]) fallbackProducerEventHandler(ctx context.Context) {
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
