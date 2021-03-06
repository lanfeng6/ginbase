package dbandmq

import (
	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	. "github.com/leyle/ginbase/consolelog"
)

const DEFAULT_SEND_RETRY_MAX = 5
const PRODUCER_NAME = "PRODUCER"
const CONSUMER_NAME = "CONSUMER"

type MqOption struct {
	Host []string  `json:"host" yaml:"host"`
	Topic []string `json:"topic" yaml:"topic"`
	GroupId string `json:"groupid" yaml:"groupid"`
	SendRetryMax int `json:"sendretrymax" yaml:"sendretrymax"`
	Stop chan struct{}
}


func(m *MqOption) Info(name string) {
	Logger.Infof("", "Current connect [%s] kafka: Host[%s], topic[%s], groupId[%s], retrymax[%d]\n", name, m.Host, m.Topic, m.GroupId, m.SendRetryMax)
}

func NewKafkaProducer(opt *MqOption) (sarama.SyncProducer, error) {
	if opt.SendRetryMax == 0 {
		opt.SendRetryMax = DEFAULT_SEND_RETRY_MAX
	}
	opt.Info(PRODUCER_NAME)

	cf := sarama.NewConfig()
	cf.Producer.RequiredAcks = sarama.WaitForAll
	cf.Producer.Retry.Max = opt.SendRetryMax
	cf.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(opt.Host, cf)
	if err != nil {
		Logger.Errorf("", "failed to create sync kafka producer, %s", err.Error())
		return nil, err
	}

	return producer, nil
}

func SendMsg(producer sarama.SyncProducer, topic, key string, data []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key: sarama.StringEncoder(key),
		Value: sarama.StringEncoder(string(data)),
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		Logger.Errorf("", "send kafka msg failed, %s, %s, %s", topic, key, err.Error())
		return err
	}
	Logger.Infof("", "msgId: %s, partition: %d, offset: %d", key, partition, offset)
	return nil
}


func NewKafkaConsumer(opt *MqOption) (*cluster.Consumer, error) {
	opt.Info(CONSUMER_NAME)

	cf := cluster.NewConfig()
	cf.Consumer.Return.Errors = true
	cf.Group.Return.Notifications = true
	consumer, err := cluster.NewConsumer(opt.Host, opt.GroupId, opt.Topic, cf)
	if err != nil {
		Logger.Errorf("", "failed to create kafka consumer, %s", err.Error())
		return nil, err
	}

	return consumer, nil
}

func ConsumeMsg(opt *MqOption, handleF func(message *sarama.ConsumerMessage)) error {
	var err error
	consumer, err := NewKafkaConsumer(opt)
	if err != nil {
		return err
	}
	defer consumer.Close()

	go func() {
		for err := range consumer.Errors() {
			Logger.Errorf("", "consume msg error: %s, %s, %s, %s", opt.Host, opt.Topic, opt.GroupId, err.Error())
		}
	}()

	go func() {
		for ntf := range consumer.Notifications() {
			Logger.Infof("", "consume msg rebalanced, %v", ntf)
		}
	}()

	for {
		select {
		case msg, ok := <-consumer.Messages():
			if ok {
				go func(msg *sarama.ConsumerMessage) {
					handleF(msg)
					consumer.MarkOffset(msg, "")
				}(msg)
			}
		case <-opt.Stop:
			Logger.Warnf("", "stop consume: %s, %s, %s", opt.Host, opt.Topic, opt.GroupId)
			return nil
		}
	}
}