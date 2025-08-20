package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	infra "github.com/infra-bed/go-spikes/pkg/infra/kafka"
	"github.com/infra-bed/go-spikes/pkg/infra/kafka/entityrepo"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/infra-bed/go-spikes/pkg/model"
)

func EntityRepoTest(w http.ResponseWriter, r *http.Request) {
	var err error
	var producerJob model.Job
	var consumerJob model.Job

	testConfig := configManager.GetTests().EntityRepoConfig

	kConfig := cfg.ApplyKafkaConfigOverrides(configManager.GetKafka(), testConfig.KafkaOverrides)

	if producerJob, err = infra.NewProducerJob[entityrepo.Payload](
		kConfig,
		entityrepo.NewProducerPlugin(testConfig.PluginsConfig.ProducerPluginConfig),
	); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to create producer engine")
		http.Error(w, "Failed to create producer engine", http.StatusInternalServerError)
		return
	}

	if consumerJob, err = infra.NewConsumerJob[entityrepo.Payload](kConfig, entityrepo.NewConsumerPlugin(
		testConfig.PluginsConfig.ConsumerPluginConfig,
	)); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to create consumer engine")
		http.Error(w, "Failed to create consumer engine", http.StatusInternalServerError)
		return
	}

	runner := model.NewRunner()
	jobs := []model.Job{producerJob, consumerJob}

	for _, job := range jobs {
		runner.Start(context.Background(), job)
	}

	var response = map[string]interface{}{
		"jobs": []string{
			producerJob.GetPlugin().GetName(),
			consumerJob.GetPlugin().GetName(),
		},
		"startTime": time.Now(),
	}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to write response")
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
