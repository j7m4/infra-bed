package kafka

type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

func DefaultKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		Brokers: []string{
			"persistent-cluster-kafka-bootstrap.streaming:9092",
		},
		Topic:         "payloads",
		ConsumerGroup: "payload-consumer-group",
	}
}
