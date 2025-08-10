package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-infra-spikes/go-spikes/pkg/kafka"
	"github.com/go-infra-spikes/go-spikes/pkg/logger"
)

func KafkaConsume(w http.ResponseWriter, r *http.Request) {
	var consumer *kafka.Consumer
	var err error
	var count int
	var start time.Time
	var duration time.Duration
	ctx := r.Context()
	log := logger.Ctx(ctx)

	consumer, err = kafka.NewConsumer(kafka.DefaultKafkaConfig())
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kafka consumer")
		http.Error(w, fmt.Sprintf("Kafka create consumer error: %v", err), http.StatusInternalServerError)
		return
	}
	defer func(consumer *kafka.Consumer) {
		err := consumer.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close Kafka consumer")
		} else {
			log.Debug().Msg("Kafka consumer closed successfully")
		}
	}(consumer)

	start = time.Now()

	if err = consumer.Consume(ctx, func(ctx context.Context, payload *kafka.Payload) error {
		count++
		// timer will stop when last message is read
		duration = time.Since(start)
		log.Debug().Str("entity ", payload.EntityID)
		return nil
	}); err != nil {
		log.Error().Err(err).Msg("Failed to consume Kafka messages")
		http.Error(w, fmt.Sprintf("Kafka consume error: %v", err), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("kafka message count %d", count),
		Duration: duration.String(),
	}); err != nil {
		log.Error().Err(err).Msg("Failed write KafkaConsume HTTP response!")
	}
}

func KafkaProduce(w http.ResponseWriter, r *http.Request) {
	var producer *kafka.Producer
	var err error
	var count int
	var start time.Time
	var duration time.Duration
	ctx := r.Context()
	log := logger.Ctx(ctx)

	if producer, err = kafka.NewProducer(kafka.DefaultKafkaConfig()); err != nil {
		log.Error().Err(err).Msg("Failed to create Kafka producer")
		http.Error(w, fmt.Sprintf("Kafka create producer error: %v", err), http.StatusInternalServerError)
		return
	}
	defer producer.Close()

	start = time.Now()

	var payloads <-chan *kafka.Payload
	if payloads, err = kafka.GeneratePayloads(nil); err != nil {
		log.Error().Err(err).Msg("Failed to generate Kafka payloads")
		http.Error(w, fmt.Sprintf("Kafka generate payloads error: %v", err), http.StatusInternalServerError)
		return
	}
	if count, err = producer.ProduceBatch(ctx, payloads); err != nil {
		log.Error().Err(err).Msg("Failed to produce Kafka messages")
		http.Error(w, fmt.Sprintf("Kafka produce error: %v", err), http.StatusInternalServerError)
		return
	}
	duration = time.Since(start)

	if err = json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("kafka message count %d", count),
		Duration: duration.String(),
	}); err != nil {
		log.Error().Err(err).Msg("Failed write KafkaProduce HTTP response!")
	}
}
