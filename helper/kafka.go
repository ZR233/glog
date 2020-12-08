package helper

import (
	"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/sirupsen/logrus"
	"path"
	"time"
)

func GetBreakerHosts(zkHosts []string) (hosts []string) {
	zkLogger := logrus.New()
	zkLogger.SetLevel(logrus.WarnLevel)
	conn, _, err := zk.Connect(zkHosts, time.Second*5, zk.WithLogger(zkLogger))
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
