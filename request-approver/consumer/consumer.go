package consumer

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"viktorbarzin/webhook-handler/chatbot/auth"
)

type ApprovalRequest struct {
	User auth.User
}

func InitConsumer(bootstrapServer, topic, groupId string) (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(
		&kafka.ConfigMap{
			"bootstrap.servers": bootstrapServer,
			// "client.id":         "kek",
			"group.id": groupId,
			// "enable.auto.commit":   false,
			// "max.poll.interval.ms": 30000000,
			// "fetch.min.bytes":      10,
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create producer")
	}
	c.SubscribeTopics([]string{topic}, nil)
	return c, nil
}

func HandleApprovalRequests(c *kafka.Consumer) error {
	for {
		e, err := c.ReadMessage(-1)
		if err != nil {
			return errors.Wrapf(err, "reading from topic failed")
		}
		if _, err := c.CommitMessage(e); err != nil {
			glog.Warningf("failed to commit message: %s", string(e.Value))
			continue
		}
		glog.Infof("Message on %s: %s", e.TopicPartition, string(e.Value))
	}
}
