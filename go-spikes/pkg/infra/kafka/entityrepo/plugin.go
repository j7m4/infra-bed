package entityrepo

import (
	"context"
	"encoding/json"
	"time"

	k "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

type ProducerPlugin struct {
	pluginCfg k.ProducerPluginConfig
}

func NewProducerPlugin(pluginCfg k.ProducerPluginConfig) *ProducerPlugin {
	return &ProducerPlugin{
		pluginCfg: pluginCfg,
	}
}

func (p *ProducerPlugin) GetRunDuration() time.Duration {
	return p.pluginCfg.RunDuration
}

func (p *ProducerPlugin) GetIntervalDuration() time.Duration {
	return p.pluginCfg.IntervalDuration
}

func (p *ProducerPlugin) ProduceListener(_ctx context.Context) {
	/*
		if p.produceStart.IsZero() {
			p.produceStart = time.Now()
		}
		p.produceEnd = time.Now()
	*/
}

func (p *ProducerPlugin) Payloads(ctx context.Context) (<-chan Payload, error) {
	return GeneratePayloads(ctx, p.pluginCfg)
}

type ConsumerPlugin struct {
	entities  map[string]*Payload
	pluginCfg k.ConsumerPluginConfig
	counter   int
}

func NewConsumerPlugin(pluginCfg k.ConsumerPluginConfig) *ConsumerPlugin {
	return &ConsumerPlugin{
		entities:  make(map[string]*Payload),
		pluginCfg: pluginCfg,
	}
}

func (c *ConsumerPlugin) ConsumeHandler(ctx context.Context, bytes []byte) error {
	var payload Payload
	log := logger.WithContext(ctx)
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return err
	}
	c.entities[payload.EntityID] = &payload
	c.counter++
	if c.counter%10000 == 0 {
		log.Info().
			Int("consumeCount", c.counter).
			Int("entityCount", len(c.entities)).
			Msg("consumed payloads")
	}
	return nil
}

func (c *ConsumerPlugin) GetRunDuration() time.Duration {
	return c.pluginCfg.RunDuration
}

func (c *ConsumerPlugin) GetIntervalDuration() time.Duration {
	return c.pluginCfg.IntervalDuration
}
