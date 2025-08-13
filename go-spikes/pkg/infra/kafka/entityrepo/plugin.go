package entityrepo

import (
	"context"
	"encoding/json"
	"time"

	k "github.com/infra-bed/go-spikes/pkg/config/kafka"
)

type PluginEntityRepo struct {
	payloadsCfg  k.PayloadsConfig
	entities     map[string]*Payload
	consumeStart time.Time
	consumeEnd   time.Time
	produceStart time.Time
	produceEnd   time.Time
}

func NewPluginEntityRepo(cfg k.PayloadsConfig) *PluginEntityRepo {
	return &PluginEntityRepo{
		payloadsCfg: cfg,
		entities:    make(map[string]*Payload, cfg.EntityCount),
	}
}

func (p PluginEntityRepo) ConsumeHandler(_ctx context.Context, bytes []byte) error {
	var payload Payload
	if p.consumeStart.IsZero() {
		p.consumeStart = time.Now()
	}
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return err
	}
	p.consumeEnd = time.Now()
	return nil
}

func (p PluginEntityRepo) PublishListener(_ctx context.Context) {
	if p.produceStart.IsZero() {
		p.produceStart = time.Now()
	}
	p.produceEnd = time.Now()
}

func (p PluginEntityRepo) GeneratePayloads(_ctx context.Context) (<-chan *Payload, error) {
	return GeneratePayloads(p.payloadsCfg)
}
