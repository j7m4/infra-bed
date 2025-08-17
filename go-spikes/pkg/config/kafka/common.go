package kafka

import "time"

type KafkaConfig struct {
	Brokers        []string       `mapstructure:"brokers"`
	Topic          string         `mapstructure:"topic"`
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
	if overrides.ProducerConfig.ClientId != "" {
		kc.ProducerConfig.ClientId = overrides.ProducerConfig.ClientId
	}
	if overrides.ProducerConfig.CompressionType != "" {
		kc.ProducerConfig.CompressionType = overrides.ProducerConfig.CompressionType
	}
	if overrides.ProducerConfig.MaxRetries > 0 {
		kc.ProducerConfig.MaxRetries = overrides.ProducerConfig.MaxRetries
	}
	if overrides.ProducerConfig.LogBatchSize > 0 {
		kc.ProducerConfig.LogBatchSize = overrides.ProducerConfig.LogBatchSize
	}
	if overrides.ConsumerConfig.ClientId != "" {
		kc.ConsumerConfig.ClientId = overrides.ConsumerConfig.ClientId
	}
	if overrides.ConsumerConfig.IsolationLevel != "" {
		kc.ConsumerConfig.IsolationLevel = overrides.ConsumerConfig.IsolationLevel
	}
	if overrides.ConsumerConfig.ConsumerGroup != "" {
		kc.ConsumerConfig.ConsumerGroup = overrides.ConsumerConfig.ConsumerGroup
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
	if overrides.ConsumerConfig.LogBatchSize > 0 {
		kc.ConsumerConfig.LogBatchSize = overrides.ConsumerConfig.LogBatchSize
	}

	return kc
}

type ProducerConfig struct {
	ClientId        string `mapstructure:"clientId"`
	CompressionType string `mapstructure:"compressionType"`
	MaxRetries      int    `mapstructure:"maxRetries"`
	Acks            string `mapstructure:"acks"` // "all", "1", "0"
	LogBatchSize    int    `mapstructure:"logBatchSize"`
}

type ConsumerConfig struct {
	ClientId           string        `mapstructure:"clientId"`
	IsolationLevel     string        `mapstructure:"isolationLevel"`
	ConsumerGroup      string        `mapstructure:"consumerGroup"`
	SessionTimeout     time.Duration `mapstructure:"sessionTimeout"`
	HeartbeatInterval  time.Duration `mapstructure:"heartbeatInterval"`
	MaxPollRecords     int           `mapstructure:"maxPollRecords"`
	AutoOffsetReset    string        `mapstructure:"autoOffsetReset"`
	AutoCommitInterval time.Duration `mapstructure:"autoCommitInterval"`
	AutoCommitEnabled  bool          `mapstructure:"autoCommitEnabled"`
	MaxPollInterval    time.Duration `mapstructure:"maxPollInterval"`
	LogBatchSize       int           `mapstructure:"logBatchSize"`
}
