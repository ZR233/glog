package adaptor

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
)

type ConfigLogstash struct {
	ZkHosts []string
}
type Status int

const (
	_ Status = iota
	StatusOk
	StatusFail
)

type WriterConfig interface {
}

type Writer interface {
	Run(config WriterConfig, prefix string, ctx context.Context)
	Write(entry *logrus.Entry)
	WriteFail() <-chan *logrus.Entry
	GetStatus() Status
}

type WriterBase struct {
	FailChan chan *logrus.Entry
	Ctx      context.Context
	AppName  string
	Status   Status
	sync.Mutex
}

func (c *WriterBase) GetStatus() Status {
	c.Lock()
	defer c.Unlock()
	return c.Status
}
