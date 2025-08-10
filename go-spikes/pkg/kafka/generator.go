package kafka

import "fmt"

func (cfg *PayloadsConfig) GetTotalCount() (int, error) {
	if cfg == nil {
		return 0, fmt.Errorf("configuration cannot be nil")
	}
	if cfg.EntityCount <= 0 || cfg.IterationCount <= 0 || cfg.AttributeCount <= 0 {
		return 0, fmt.Errorf("all counts must be greater than zero")
	}
	return cfg.EntityCount * cfg.IterationCount, nil
}

type PayloadSpecs struct {
	EntityIdx      int
	IterIdx        int
	AttributeCount int
}

type Payload struct {
	EntityID   string
	Attributes map[string]interface{}
}

func createPayload(specs PayloadSpecs) (*Payload, error) {
	payload := &Payload{
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

func GeneratePayloads(cfg *PayloadsConfig) (<-chan *Payload, error) {
	if cfg == nil {
		cfg = DefaultPayloadsConfig()
	}
	if cfg.EntityCount <= 0 || cfg.IterationCount <= 0 || cfg.AttributeCount <= 0 {
		return nil, fmt.Errorf("invalid configuration: all counts must be greater than zero")
	}

	payloads := make(chan *Payload)

	go func() {
		defer close(payloads)
		for entityIdx := 0; entityIdx < cfg.EntityCount; entityIdx++ {
			for iterIdx := 0; iterIdx < cfg.IterationCount; iterIdx++ {
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
		}
	}()

	return payloads, nil
}
