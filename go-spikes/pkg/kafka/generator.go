package kafka

import "fmt"

// Config determines how the nature of Payload Generator's behavior with:
// * EntityCount - the number of unique Entities to include
// * IterationCount - how many iterations of payloads for each Entity
// * AttributeCount - the number of random attributes to generate for each Payload
// and the number
type Config struct {
	EntityCount    int
	IterationCount int
	AttributeCount int
}

func DefaultConfig() *Config {
	return &Config{
		EntityCount:    10_000,
		IterationCount: 10,
		AttributeCount: 5,
	}
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

func GeneratePayloads(cfg *Config) (<-chan *Payload, error) {
	if cfg == nil {
		cfg = DefaultConfig()
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
