package utils

import (
	"example.com/proj/config"
	"github.com/segmentio/kafka-go"
)

func SetupKafkaTopic(cfg config.KafkaConnCfg) {
	conn := KafkaConn(cfg)

	if !IsTopicAlreadyExists(conn, cfg.Topic) {
		topicConfigs := []kafka.TopicConfig{
			{
				Topic:             cfg.Topic,
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		}

		err := conn.CreateTopics(topicConfigs...)
		if err != nil {
			panic(err.Error())
		}
	}
}
