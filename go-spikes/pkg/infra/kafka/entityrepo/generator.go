package entityrepo

import (
	"context"
	"fmt"

	k "github.com/infra-bed/go-spikes/pkg/config/kafka"
	"github.com/rs/zerolog/log"
)

type PayloadSpecs struct {
	EntityIdx      int
	IterIdx        int
	AttributeCount int
}

type Payload struct {
	EntityID   string
	Attributes map[string]interface{}
}

func createPayload(specs PayloadSpecs) (Payload, error) {
	payload := Payload{
		EntityID:   fmt.Sprintf("entity-%d", specs.EntityIdx),
		Attributes: make(map[string]interface{}),
	}

	for i := 0; i < specs.AttributeCount; i++ {
		attrKey := fmt.Sprintf("attr-%d", i)
		attrValue := fmt.Sprintf("value-%d-%d-%d", specs.EntityIdx, specs.IterIdx, i)
		payload.Attributes[attrKey] = attrValue
	}

	return payload, nil
}

func GeneratePayloads(ctx context.Context, cfg k.ProducerPluginConfig) (<-chan Payload, error) {
	if cfg.EntityCount <= 0 || cfg.AttributeCount <= 0 {
		return nil, fmt.Errorf("invalid configuration: all counts must be greater than zero")
	}

	payloads := make(chan Payload)

	go func() {
		var iterIdx int
		var entityIdx int
		defer func() {
			close(payloads)
			log.Info().Msg("Generator closed successfully")
		}()
		for {
			entityIdx++
			select {
			case <-ctx.Done():
				return
			default:
				specs := PayloadSpecs{
					EntityIdx:      entityIdx,
					IterIdx:        iterIdx,
					AttributeCount: cfg.AttributeCount,
				}
				payload, err := createPayload(specs)
				if err != nil {
					fmt.Printf(
						"Error creating payload for entity %d, iteration %d: %v\n",
						entityIdx, iterIdx, err,
					)
					continue
				}
				payloads <- payload
			}
			if entityIdx < cfg.EntityCount {
				entityIdx++
			} else {
				entityIdx = 0
			}
			iterIdx++
		}
	}()

	return payloads, nil
}
