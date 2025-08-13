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
	"go.opentelemetry.io/otel"
)

func EntityRepoTest(w http.ResponseWriter, r *http.Request) {
	var plugin *entityrepo.PluginEntityRepo

	testConfig := configManager.GetTests().EntityRepoConfig

	plugin = entityrepo.NewPluginEntityRepo(testConfig.PayloadsConfig)
	kConfig := cfg.ApplyKafkaConfigOverrides(configManager.GetKafka(), testConfig.KafkaOverrides)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		traceCtx, span := otel.Tracer("EntityRepoTest").Start(ctx, "EntityRepoTest")
		defer span.End()
		go func() {
			k.RunProducer(traceCtx, kConfig, plugin)
		}()
		go func() {
			k.RunConsumer(traceCtx, kConfig, plugin)
		}()
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("running"),
	})
}
