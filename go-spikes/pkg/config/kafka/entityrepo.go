package kafka

import "time"

type EntityRepoConfig struct {
	PluginsConfig  PluginsConfig `mapstructure:"plugins"`
	KafkaOverrides KafkaConfig   `mapstructure:"kafkaOverrides"`
}

type PluginsConfig struct {
	ConsumerPluginConfig ConsumerPluginConfig `mapstructure:"consumer"`
	ProducerPluginConfig ProducerPluginConfig `mapstructure:"producer"`
}

// ProducerPluginConfig determines how the nature of ProducerEngine's Plugin behaves with:
// * EntityCount - the number of unique Entities to include
// * AttributeCount - the number of random attributes to generate for each Payload
// * RunDuration - the total duration to run the ProducerEngine
// * IntervalDuration - the interval between producing payloads
type ProducerPluginConfig struct {
	EntityCount      int           `mapstructure:"entityCount"`
	AttributeCount   int           `mapstructure:"attributeCount"`
	RunDuration      time.Duration `mapstructure:"runDuration"`
	IntervalDuration time.Duration `mapstructure:"intervalDuration"`
}

// ConsumerPluginConfig determines how the nature of Payload Generator's behavior with:
// * RunDuration - the total duration to run the ProducerEngine
// * IntervalDuration - the interval between producing payloads
type ConsumerPluginConfig struct {
	RunDuration      time.Duration `mapstructure:"runDuration"`
	IntervalDuration time.Duration `mapstructure:"intervalDuration"`
}
