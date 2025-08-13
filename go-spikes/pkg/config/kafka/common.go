package kafka

import "time"

type KafkaConfig struct {
	Brokers        []string       `mapstructure:"brokers"`
	Topic          string         `mapstructure:"topic"`
	ConsumerGroup  string         `mapstructure:"consumerGroup"`
	ProducerConfig ProducerConfig `mapstructure:"producer"`
	ConsumerConfig ConsumerConfig `mapstructure:"consumer"`
}

func ApplyKafkaConfigOverrides(kc KafkaConfig, overrides KafkaConfig) KafkaConfig {
	if overrides.Brokers != nil {
		kc.Brokers = overrides.Brokers
	}
	if overrides.Topic != "" {
		kc.Topic = overrides.Topic
	}
	if overrides.ConsumerGroup != "" {
		kc.ConsumerGroup = overrides.ConsumerGroup
	}
	if overrides.ProducerConfig.BatchSize > 0 {
		kc.ProducerConfig.BatchSize = overrides.ProducerConfig.BatchSize
	}
	if overrides.ProducerConfig.BatchTimeout > 0 {
		kc.ProducerConfig.BatchTimeout = overrides.ProducerConfig.BatchTimeout
	}
	if overrides.ProducerConfig.CompressionType != "" {
		kc.ProducerConfig.CompressionType = overrides.ProducerConfig.CompressionType
	}
	if overrides.ProducerConfig.MaxRetries > 0 {
		kc.ProducerConfig.MaxRetries = overrides.ProducerConfig.MaxRetries
	}
	if overrides.ConsumerConfig.SessionTimeout > 0 {
		kc.ConsumerConfig.SessionTimeout = overrides.ConsumerConfig.SessionTimeout
	}
	if overrides.ConsumerConfig.HeartbeatInterval > 0 {
		kc.ConsumerConfig.HeartbeatInterval = overrides.ConsumerConfig.HeartbeatInterval
	}
	if overrides.ConsumerConfig.MaxPollRecords > 0 {
		kc.ConsumerConfig.MaxPollRecords = overrides.ConsumerConfig.MaxPollRecords
	}
	if overrides.ConsumerConfig.AutoOffsetReset != "" {
		kc.ConsumerConfig.AutoOffsetReset = overrides.ConsumerConfig.AutoOffsetReset
	}
	if overrides.ConsumerConfig.AutoCommitInterval > 0 {
		kc.ConsumerConfig.AutoCommitInterval = overrides.ConsumerConfig.AutoCommitInterval
	}
	if overrides.ConsumerConfig.AutoCommitEnabled {
		kc.ConsumerConfig.AutoCommitEnabled = overrides.ConsumerConfig.AutoCommitEnabled
	}
	if overrides.ConsumerConfig.MaxPollInterval > 0 {
		kc.ConsumerConfig.MaxPollInterval = overrides.ConsumerConfig.MaxPollInterval
	}

	return kc
}

type ProducerConfig struct {
	BatchSize       int           `mapstructure:"batchSize"`
	BatchTimeout    time.Duration `mapstructure:"batchTimeout"`
	CompressionType string        `mapstructure:"compressionType"`
	MaxRetries      int           `mapstructure:"maxRetries"`
	Acks            string        `mapstructure:"acks"` // "all", "1", "0"
}

type ConsumerConfig struct {
	SessionTimeout     time.Duration `mapstructure:"sessionTimeout"`
	HeartbeatInterval  time.Duration `mapstructure:"heartbeatInterval"`
	MaxPollRecords     int           `mapstructure:"maxPollRecords"`
	AutoOffsetReset    string        `mapstructure:"autoOffsetReset"`
	AutoCommitInterval time.Duration `mapstructure:"autoCommitInterval"`
	AutoCommitEnabled  bool          `mapstructure:"autoCommitEnabled"`
	MaxPollInterval    time.Duration `mapstructure:"maxPollInterval"`
}
