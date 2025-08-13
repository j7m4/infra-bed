package kafka

import (
	"time"
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
		SilenceTimeout: 10 * time.Second,
	}
}
