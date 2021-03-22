package producer

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

func InitProducer(bootstrapServer string) (*kafka.Producer, error) {
	p, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers": bootstrapServer,
			// "client.id":         "kek",
			"acks": "all",
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create producer")
	}
	return p, nil
}

func Produce(p *kafka.Producer, topic string, payload []byte) error {
	delivery_chan := make(chan kafka.Event, 10000)
	err := p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: payload,
	},
		delivery_chan,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to produce message %s", payload)
	}

	e := <-delivery_chan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return errors.Wrapf(m.TopicPartition.Error, "delivery failed")
	}

	glog.Infof("Delivered message to topic %s [%d] at offset %v\n", *m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)

	close(delivery_chan)
	return nil
}
