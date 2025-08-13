package kafka

type EntityRepoConfig struct {
	PayloadsConfig PayloadsConfig `mapstructure:"payloads"`
	KafkaOverrides KafkaConfig    `mapstructure:"kafkaOverrides"`
}

// PayloadsConfig determines how the nature of Payload Generator's behavior with:
// * EntityCount - the number of unique Entities to include
// * IterationCount - how many iterations of payloads for each Entity
// * AttributeCount - the number of random attributes to generate for each Payload
// and the number
type PayloadsConfig struct {
	EntityCount    int `mapstructure:"entityCount"`
	IterationCount int `mapstructure:"iterationCount"`
	AttributeCount int `mapstructure:"attributeCount"`
}
