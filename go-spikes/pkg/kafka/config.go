package kafka

import (
	"fmt"
	"strings"
	"time"

	k "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ConnectionConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Brokers: []string{
			"persistent-cluster-kafka-bootstrap.streaming:9092",
		},
		Topic:         "payloads",
		ConsumerGroup: "payload-consumer-group",
	}
}

// ConsumeTestSettings defines the settings for consume tests:
// * TestTimeout - the maximum duration to wait for consuming messages
// * SilenceTimeout - the amount of time to wait for no messages before ending the test
type ConsumeTestSettings struct {
	TestTimeout    time.Duration
	SilenceTimeout time.Duration
}

func DefaultConsumeTestSettings() *ConsumeTestSettings {
	return &ConsumeTestSettings{
		TestTimeout:    30 * time.Second,
		SilenceTimeout: 2 * time.Second,
	}
}

// PayloadsConfig determines how the nature of Payload Generator's behavior with:
// * EntityCount - the number of unique Entities to include
// * IterationCount - how many iterations of payloads for each Entity
// * AttributeCount - the number of random attributes to generate for each Payload
// and the number
type PayloadsConfig struct {
	EntityCount    int
	IterationCount int
	AttributeCount int
}

func DefaultPayloadsConfig() *PayloadsConfig {
	return &PayloadsConfig{
		EntityCount:    10_000,
		IterationCount: 10,
		AttributeCount: 5,
	}
}

func DefaultConsumerConfigMap(connConfig *ConnectionConfig) *k.ConfigMap {
	return &k.ConfigMap{
		"bootstrap.servers":       strings.Join(DefaultConnectionConfig().Brokers, ","),
		"group.id":                DefaultConnectionConfig().ConsumerGroup,
		"client.id":               fmt.Sprintf("%s-client", DefaultConnectionConfig().ConsumerGroup),
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      false,
		"auto.commit.interval.ms": 5000,
		"session.timeout.ms":      10000,
		"max.poll.interval.ms":    300000,
	}
}

func DefaultProducerConfigMap(connConfig *ConnectionConfig) *k.ConfigMap {
	return &k.ConfigMap{
		"bootstrap.servers": strings.Join(connConfig.Brokers, ","),
		"client.id":         "payload-producer",
		"acks":              "all",
		"retries":           10,
		"linger.ms":         10,
		"compression.type":  "snappy",
	}
}
