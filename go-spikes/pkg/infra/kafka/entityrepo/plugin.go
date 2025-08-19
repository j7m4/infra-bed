package entityrepo

import (
	"context"
	"encoding/json"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/infra-bed/go-spikes/pkg/config"
	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
	infra "github.com/infra-bed/go-spikes/pkg/infra/kafka"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

type ProducerPlugin struct {
	pluginCfg    cfg.ProducerPluginConfig
	counter      int
	logBatchSize int
}

func NewProducerPlugin(pluginCfg cfg.ProducerPluginConfig) *ProducerPlugin {
	logBatchSize := pluginCfg.LogBatchSize
	if logBatchSize <= 0 {
		logBatchSize = config.DefaultLogBatchSize
	}
	return &ProducerPlugin{
		pluginCfg:    pluginCfg,
		logBatchSize: logBatchSize,
	}
}

func (p *ProducerPlugin) GetInitialDelayDuration() time.Duration {
	return p.pluginCfg.InitialDelayDuration
}

func (p *ProducerPlugin) GetRunDuration() time.Duration {
	return p.pluginCfg.RunDuration
}

func (p *ProducerPlugin) GetIntervalDuration() time.Duration {
	return p.pluginCfg.IntervalDuration
}

func (p *ProducerPlugin) ProduceMessageListener(ctx context.Context, engine infra.ProducerEngine[Payload], msg *k.Message) error {
	var err error
	var payload Payload
	log := logger.WithContext(ctx)
	if err = json.Unmarshal(msg.Value, &payload); err != nil {
		return err
	}
	p.counter++
	if p.counter%p.logBatchSize == 0 {
		log.Info().
			Int("produceCount", p.counter).
			Int64("offset", int64(msg.TopicPartition.Offset)).
			Msg("produced payloads")
	}
	return nil
}

func (p *ProducerPlugin) Payloads(ctx context.Context) (<-chan Payload, error) {
	return GeneratePayloads(ctx, p.pluginCfg)
}

type ConsumerPlugin struct {
	entities     map[string]*Payload
	pluginCfg    cfg.ConsumerPluginConfig
	counter      int
	logBatchSize int
}

func NewConsumerPlugin(pluginCfg cfg.ConsumerPluginConfig) *ConsumerPlugin {
	logBatchSize := pluginCfg.LogBatchSize
	if logBatchSize <= 0 {
		logBatchSize = config.DefaultLogBatchSize
	}
	return &ConsumerPlugin{
		entities:     make(map[string]*Payload),
		pluginCfg:    pluginCfg,
		logBatchSize: logBatchSize,
	}
}

func (c *ConsumerPlugin) ConsumeMessageHandler(ctx context.Context, engine infra.ConsumerEngine[Payload], msg *k.Message) error {
	var err error
	var payload Payload
	log := logger.WithContext(ctx)
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return err
	}
	c.entities[payload.EntityID] = &payload
	if err = engine.AcceptMessage(msg); err != nil {
		log.Error().Err(err).Msg("Failed to commit message")
	}
	c.counter++
	if c.counter%c.logBatchSize == 0 {
		log.Info().
			Int("consumeCount", c.counter).
			Int("entityCount", len(c.entities)).
			Int64("offset", int64(msg.TopicPartition.Offset)).
			Msg("consumed payloads")
	}
	return nil
}

func (c *ConsumerPlugin) GetInitialDelayDuration() time.Duration {
	return c.pluginCfg.InitialDelayDuration
}

func (c *ConsumerPlugin) GetRunDuration() time.Duration {
	return c.pluginCfg.RunDuration
}

func (c *ConsumerPlugin) GetIntervalDuration() time.Duration {
	return c.pluginCfg.IntervalDuration
}
