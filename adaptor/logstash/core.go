package logstash

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/ZR233/glog/adaptor"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/sirupsen/logrus"
	"path"
	"time"
)

const (
	topic = "logstash"
)

type Core struct {
	adaptor.ConfigLogstash
	adaptor.WriterBase
	producer sarama.AsyncProducer
	fmt      logrus.Formatter
}

func (c *Core) Write(entry *logrus.Entry) {
	logBytes, err := c.fmt.Format(entry)
	if err != nil {
		err = fmt.Errorf("[glog]format entry error\n%w", err)
		println(err)
	}

	//构建发送的消息，
	msg := &sarama.ProducerMessage{
		Key:   sarama.StringEncoder("key"),
		Value: sarama.ByteEncoder(logBytes),
		Topic: topic,
	}
	msg.Metadata = entry
	select {
	case <-c.Ctx.Done():
		return
	case c.producer.Input() <- msg:
	}

}

func (c *Core) getBreakerHosts() (hosts []string) {

	conn, _, err := zk.Connect(c.ZkHosts, time.Second*5, zk.WithLogger(logrus.New()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// get
	children, _, err := conn.Children("/brokers/ids")
	if err != nil {
		panic(err)
	}
	var data []byte
	for _, child := range children {
		// get
		data, _, err = conn.Get(path.Join("/brokers/ids", child))
		if err != nil {
			logrus.Error(err)
			continue
		}

		var broker struct {
			Host string
			Port int
		}

		err = json.Unmarshal(data, &broker)
		if err != nil {
			logrus.Error(err)
			continue
		}

		hosts = append(hosts, fmt.Sprintf("%s:%d", broker.Host, broker.Port))
	}

	return
}
func (c *Core) Run(config adaptor.WriterConfig, prefix string, ctx context.Context) {
	c.AppName = prefix
	c.Status = adaptor.StatusOk
	c.ConfigLogstash = *config.(*adaptor.ConfigLogstash)
	c.FailChan = make(chan *logrus.Entry, 10)
	c.Ctx = ctx
	c.fmt = GetLogstashFormatter()

	kafkaHosts := c.getBreakerHosts()

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Partitioner = sarama.NewRandomPartitioner

	producer, err := sarama.NewAsyncProducer(kafkaHosts, saramaConfig)
	if err != nil {
		panic(err)
	}
	c.producer = producer

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 10):
				c.Lock()
				c.Status = adaptor.StatusOk
				c.Unlock()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-c.Ctx.Done():
				return
			case fail := <-c.producer.Errors():
				if failLog, ok := fail.Msg.Metadata.(*logrus.Entry); ok {
					c.FailChan <- failLog
				} else {
					logrus.Error("[glog]", fail.Error())
				}

				c.Lock()
				lastStatus := c.Status
				c.Status = adaptor.StatusFail
				c.Unlock()

				if lastStatus != adaptor.StatusFail {
					logrus.Error("[glog]kafka send fail: \n", fail.Error())
				}
			}
		}
	}()
}
func (c *Core) WriteFail() <-chan *logrus.Entry {
	return c.FailChan
}
func GetLogstashFormatter() *logrus.JSONFormatter {
	return &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.999+08:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "@timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "caller",
		},
	}
}
