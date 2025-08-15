package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	k "github.com/infra-bed/go-spikes/pkg/infra/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra/kafka/entityrepo"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func EntityRepoTest(w http.ResponseWriter, r *http.Request) {
	var err error
	var producerPlugin *entityrepo.ProducerPlugin
	var producerEngine *k.ProducerEngine[entityrepo.Payload]
	var consumerPlugin *entityrepo.ConsumerPlugin
	var consumerEngine *k.ConsumerEngine[entityrepo.Payload]

	testConfig := configManager.GetTests().EntityRepoConfig

	kConfig := cfg.ApplyKafkaConfigOverrides(configManager.GetKafka(), testConfig.KafkaOverrides)

	producerPlugin = entityrepo.NewProducerPlugin(
		testConfig.PluginsConfig.ProducerPluginConfig,
	)
	if producerEngine, err = k.NewProducerEngine[entityrepo.Payload](kConfig, producerPlugin); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to create producer engine")
		http.Error(w, "Failed to create producer engine", http.StatusInternalServerError)
		return
	}

	consumerPlugin = entityrepo.NewConsumerPlugin(
		testConfig.PluginsConfig.ConsumerPluginConfig,
	)
	if consumerEngine, err = k.NewConsumerEngine[entityrepo.Payload](kConfig, consumerPlugin); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to create consumer engine")
		http.Error(w, "Failed to create consumer engine", http.StatusInternalServerError)
		return
	}

	spanName := "EntityRepoTest"
	callback := make(chan Response)

	execute := func(ctx context.Context) {
		go func() {
			producerEngine.Run(ctx)
			defer producerEngine.Close()
		}()
		go func() {
			consumerEngine.Run(ctx)
			defer consumerEngine.Close()
		}()
	}

	go func(callback chan Response) {
		tracerName := "PerfRunner"
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		traceCtx, span := otel.Tracer(tracerName).Start(ctx, spanName)
		spanCtx := trace.SpanContextFromContext(traceCtx)
		traceId := ""
		if spanCtx.IsValid() {
			traceId = spanCtx.TraceID().String()
		}
		callback <- Response{
			Message: fmt.Sprintf("Running %s", spanName),
			TraceID: traceId,
		}
		close(callback)
		log := logger.WithContext(traceCtx)
		log.Info().Str("name", spanName).Msg("PerfRunner started")

		defer span.End()
		defer cancel()
		go func() {
			execute(traceCtx)
		}()
		for {
			select {
			case <-ctx.Done():
				log.Info().Str("name", spanName).Msg("PerfRunner ended")
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}(callback)

	if err = json.NewEncoder(w).Encode(<-callback); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to write response")
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
