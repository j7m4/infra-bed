package entityrepo

import (
	"fmt"
	"testing"

	cfg "github.com/infra-bed/go-spikes/pkg/config/kafka"
)

func TestGeneratePayloads_SmallSet(t *testing.T) {
	cfg := cfg.PayloadsConfig{
		EntityCount:    5,
		IterationCount: 4,
		AttributeCount: 3,
	}

	payloadChan, err := GeneratePayloads(cfg)
	if err != nil {
		t.Fatalf("Payloads returned error: %v", err)
	}

	payloads := make([]*Payload, 0)
	for payload := range payloadChan {
		payloads = append(payloads, payload)
	}

	expectedCount := cfg.EntityCount * cfg.IterationCount
	if len(payloads) != expectedCount {
		t.Errorf("Expected %d payloads, got %d", expectedCount, len(payloads))
	}

	entityIterMap := make(map[string]map[int]bool)
	for _, payload := range payloads {
		if _, exists := entityIterMap[payload.EntityID]; !exists {
			entityIterMap[payload.EntityID] = make(map[int]bool)
		}

		if len(payload.Attributes) != cfg.AttributeCount {
			t.Errorf("Expected %d attributes, got %d for entity %s",
				cfg.AttributeCount, len(payload.Attributes), payload.EntityID)
		}

		for i := 0; i < cfg.AttributeCount; i++ {
			attrKey := fmt.Sprintf("attr-%d", i)
			if _, exists := payload.Attributes[attrKey]; !exists {
				t.Errorf("Missing attribute %s in payload for entity %s", attrKey, payload.EntityID)
			}
		}
	}

	if len(entityIterMap) != cfg.EntityCount {
		t.Errorf("Expected %d unique entities, got %d", cfg.EntityCount, len(entityIterMap))
	}

	for entityID := range entityIterMap {
		found := false
		for i := 0; i < cfg.EntityCount; i++ {
			expectedID := fmt.Sprintf("entity-%d", i)
			if entityID == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected entity ID: %s", entityID)
		}
	}
}

func TestGeneratePayloads_VerifyAttributeValues(t *testing.T) {
	cfg := cfg.PayloadsConfig{
		EntityCount:    2,
		IterationCount: 2,
		AttributeCount: 2,
	}

	payloadChan, err := GeneratePayloads(cfg)
	if err != nil {
		t.Fatalf("Payloads returned error: %v", err)
	}

	payloads := make([]*Payload, 0)
	for payload := range payloadChan {
		payloads = append(payloads, payload)
	}

	expectedPayloads := []struct {
		entityID string
		attrs    map[string]string
	}{
		{
			entityID: "entity-0",
			attrs: map[string]string{
				"attr-0": "value-0-0-0",
				"attr-1": "value-0-0-1",
			},
		},
		{
			entityID: "entity-0",
			attrs: map[string]string{
				"attr-0": "value-0-1-0",
				"attr-1": "value-0-1-1",
			},
		},
		{
			entityID: "entity-1",
			attrs: map[string]string{
				"attr-0": "value-1-0-0",
				"attr-1": "value-1-0-1",
			},
		},
		{
			entityID: "entity-1",
			attrs: map[string]string{
				"attr-0": "value-1-1-0",
				"attr-1": "value-1-1-1",
			},
		},
	}

	if len(payloads) != len(expectedPayloads) {
		t.Fatalf("Expected %d payloads, got %d", len(expectedPayloads), len(payloads))
	}

	for i, payload := range payloads {
		expected := expectedPayloads[i]
		if payload.EntityID != expected.entityID {
			t.Errorf("Payload %d: expected EntityID %s, got %s", i, expected.entityID, payload.EntityID)
		}

		for key, expectedValue := range expected.attrs {
			if value, exists := payload.Attributes[key]; !exists {
				t.Errorf("Payload %d: missing attribute %s", i, key)
			} else if value != expectedValue {
				t.Errorf("Payload %d: attribute %s expected value %s, got %v",
					i, key, expectedValue, value)
			}
		}
	}
}

func TestGeneratePayloads_InvalidConfig(t *testing.T) {
	testCases := []struct {
		name string
		cfg  cfg.PayloadsConfig
	}{
		{
			name: "zero entity count",
			cfg: cfg.PayloadsConfig{
				EntityCount:    0,
				IterationCount: 1,
				AttributeCount: 1,
			},
		},
		{
			name: "negative iteration count",
			cfg: cfg.PayloadsConfig{
				EntityCount:    1,
				IterationCount: -1,
				AttributeCount: 1,
			},
		},
		{
			name: "zero attribute count",
			cfg: cfg.PayloadsConfig{
				EntityCount:    1,
				IterationCount: 1,
				AttributeCount: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GeneratePayloads(tc.cfg)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestCreatePayload(t *testing.T) {
	specs := PayloadSpecs{
		EntityIdx:      42,
		IterIdx:        7,
		AttributeCount: 3,
	}

	payload, err := createPayload(specs)
	if err != nil {
		t.Fatalf("createPayload returned error: %v", err)
	}

	expectedEntityID := "entity-42"
	if payload.EntityID != expectedEntityID {
		t.Errorf("Expected EntityID %s, got %s", expectedEntityID, payload.EntityID)
	}

	if len(payload.Attributes) != specs.AttributeCount {
		t.Errorf("Expected %d attributes, got %d", specs.AttributeCount, len(payload.Attributes))
	}

	expectedAttributes := map[string]string{
		"attr-0": "value-42-7-0",
		"attr-1": "value-42-7-1",
		"attr-2": "value-42-7-2",
	}

	for key, expectedValue := range expectedAttributes {
		if value, exists := payload.Attributes[key]; !exists {
			t.Errorf("Missing attribute %s", key)
		} else if value != expectedValue {
			t.Errorf("Attribute %s: expected %s, got %v", key, expectedValue, value)
		}
	}
}
