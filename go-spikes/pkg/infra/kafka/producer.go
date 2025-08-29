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
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/infra-bed/go-spikes/pkg/metrics"
	"github.com/infra-bed/go-spikes/pkg/model"
	"github.com/infra-bed/go-spikes/pkg/tracing"
)

type ProducerJob[T any] interface {
	Run(ctx context.Context)
	Close()
	GetPlugin() model.Plugin
}

func NewProducerJob[T any](cfg cfg.KafkaConfig, plugin ProducerPlugin[T]) (model.Job, error) {
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

	return &producerJobImpl[T]{
		producer:     producer,
		config:       cfg,
		deliveryChan: make(chan k.Event, 1000),
		plugin:       plugin,
		logBatchSize: logBatchSize,
	}, nil
}

type producerJobImpl[T any] struct {
	producer     *k.Producer
	config       cfg.KafkaConfig
	deliveryChan chan k.Event
	plugin       ProducerPlugin[T]
	logBatchSize int
}

func (p *producerJobImpl[T]) GetPlugin() model.Plugin {
	return p.plugin
}

func (p *producerJobImpl[T]) Run(ctx context.Context) {
	log := logger.Ctx(ctx)
	var err error
	var payloads <-chan T

	if payloads, err = p.plugin.Payloads(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to generate Payloads")
		return
	}
	if err = p.producePayloads(ctx, payloads); err != nil {
		log.Error().Err(err).Msg("Failed to produce Kafka messages")
		return
	}
}

func (p *producerJobImpl[T]) Close() {
	log := logger.Get()
	p.producer.Close()
	close(p.deliveryChan)
	log.Info().Msg("Producer closed successfully")
}

func (p *producerJobImpl[T]) producePayloads(ctx context.Context, payloadChan <-chan T) error {
	count := 0
	batchProduceMsg := fmt.Sprintf("kafka.produce.batch: %d", p.logBatchSize)

	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
	batchCtx, batchSpan := tracing.StartSpanWithAttributes(
		ctx, 
		"kafka.producer.batch",
		tracing.KafkaAttributes(p.config.Topic, "0", "produce"),
	)
	log := logger.Ctx(batchCtx)
	defer batchSpan.End()
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR kafka

	intervalTimer := model.NewIntervalTimer(ctx, p.plugin)

	go p.fallbackProducerEventHandler(ctx)
	go p.messageDeliveryEventHandler(ctx)

	for payload := range payloadChan {
		intervalTimer.NextTickWait()
		select {
		case <-ctx.Done():
			log.Info().Int("count", count).Msg(batchProduceMsg)
			log.Info().Msg("producer done: producePayloads")
			return ctx.Err()
		default:
			if err := p.producePayloadAsync(batchCtx, payload); err != nil {
				log.Error().Err(err).Msg("Failed to produce payload")
				metrics.KafkaProduceErrors.WithLabelValues(p.config.Topic, "produce_error").Inc()
				continue
			}
			count++
			metrics.KafkaMessagesProduced.WithLabelValues(p.config.Topic, "0").Inc()
			
			if count%p.logBatchSize == 0 {
				// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR kafka
				// close out old and create new span
				batchSpan.End()
				batchCtx, batchSpan = tracing.StartSpanWithAttributes(
					ctx, 
					"kafka.producer.batch",
					tracing.KafkaAttributes(p.config.Topic, "0", "produce"),
				)
				log = logger.Ctx(batchCtx)
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
func (p *producerJobImpl[T]) producePayloadAsync(ctx context.Context, payload interface{}) error {
	ctx, span := tracing.StartSpanWithAttributes(
		ctx,
		"kafka.producer.message",
		tracing.KafkaAttributes(p.config.Topic, "any", "produce"),
	)
	defer span.End()

	data, err := json.Marshal(payload)
	if err != nil {
		tracing.RecordError(span, err, "Failed to marshal payload")
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Record message size
	metrics.KafkaMessageSize.WithLabelValues(p.config.Topic, "produce").Observe(float64(len(data)))

	key := sha256.New().Sum(data)
	msg := &k.Message{
		TopicPartition: k.TopicPartition{
			Topic:     &p.config.Topic,
			Partition: k.PartitionAny,
		},
		Key:   key,
		Value: data,
	}

	if err := p.producer.Produce(msg, p.deliveryChan); err != nil {
		tracing.RecordError(span, err, "Failed to produce message to Kafka")
		return err
	}

	tracing.AddSpanEvent(span, "message.produced")
	return nil
}

func (p *producerJobImpl[T]) messageDeliveryEventHandler(ctx context.Context) {
	log := logger.Ctx(ctx)
	counts := make(map[string]int)
	var totalCounts int

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("producer done: messageDeliveryEventHandler")
			return
		case e := <-p.deliveryChan:
			switch ev := e.(type) {
			case *k.Message:
				counts["Message"]++
				if ev.TopicPartition.Error != nil {
					log.Error().
						Err(ev.TopicPartition.Error).
						Str("key", string(ev.Key)).
						Msg("Delivery failed")
					metrics.KafkaProduceErrors.WithLabelValues(p.config.Topic, "delivery_failed").Inc()
				} else {
					err := p.plugin.ProduceMessageListener(ctx, p, ev)
					if err != nil {
						log.Error().Err(err).Msg("Failed on ProduceMessageListener")
						metrics.KafkaProduceErrors.WithLabelValues(p.config.Topic, "listener_error").Inc()
					}
				}
			default:
				counts["other"]++
			}
		}
		totalCounts = 0
		for _, count := range counts {
			totalCounts += count
		}
		if totalCounts%1000 == 0 {
			log.Debug().Any("distribution", counts).Msg("producer-delivery-events")
			counts = make(map[string]int) // Reset counts to avoid memory growth
		}
	}
}

// fallbackProducerEventHandler is limited to handling error-like Events from the producer.
// It will receive events from Produce() when there is no delivery channel specified.
// It will log delivery failures and Kafka errors.
func (p *producerJobImpl[T]) fallbackProducerEventHandler(ctx context.Context) {
	log := logger.Ctx(ctx)
	counts := make(map[string]int)
	var totalCounts int

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("producer done: fallbackProducerEventHandler")
			return
		case e := <-p.producer.Events():
			switch ev := e.(type) {
			case *k.Message:
				counts["Message"]++
				if ev.TopicPartition.Error != nil {
					log.Error().
						Err(ev.TopicPartition.Error).
						Str("key", string(ev.Key)).
						Msg("Delivery failed")
				}
			case k.Error:
				counts["error"]++
				log.Error().
					Err(ev).
					Int("code", int(ev.Code())).
					Msg("Kafka error")
			default:
				counts["other"]++
			}
			totalCounts = 0
			for _, count := range counts {
				totalCounts += count
			}
			if totalCounts%1000 == 0 {
				log.Debug().Any("distribution", counts).Msg("producer-fallback-events")
				counts = make(map[string]int) // Reset counts to avoid memory growth
			}
		}
	}
}
