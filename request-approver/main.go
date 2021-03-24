package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/viktorbarzin/webhook-handler/request-approver/consumer"
	"github.com/viktorbarzin/webhook-handler/request-approver/producer"
)

const (
	BootstrapServer = "kafka.kafka.svc.cluster.local"
	GroupID         = "request-approver"
	SubscribedTopic = "topic"
)

func main() {
	if err := run(); err != nil {
		glog.Fatalf("Run failed: %s", err.Error())
	}
}

func run() error {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	// go produce()

	c, err := consumer.InitConsumer(BootstrapServer, SubscribedTopic, "test-consumer")
	defer c.Close()
	if err != nil {
		return errors.Wrapf(err, "failed to create consumer")
	}

	glog.Infof("starting request approval handler")
	return consumer.HandleApprovalRequests(c)
}

func pollSleep() {
	time.Sleep(time.Second * 50)
}

func produce() error {
	glog.Info("Producing...")
	p, err := producer.InitProducer(BootstrapServer)
	if err != nil {
		return errors.Wrapf(err, "failed to init producer")
	}
	err = producer.Produce(p, SubscribedTopic, []byte("Test message: "+time.Now().String()))
	if err != nil {
		return errors.Wrapf(err, "failed to send message")
	}
	return nil
}
