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
	Run(config WriterConfig, coreBase *CoreBase)
	Write(entry *logrus.Entry)
	WriteFail() <-chan *logrus.Entry
	GetStatus() Status
}

type CoreBase struct {
	AppName, ModulePrefix string
	Ctx                   context.Context
}

type WriterBase struct {
	FailChan chan *logrus.Entry
	CoreBase
	Status Status
	sync.Mutex
}

func (c *WriterBase) GetStatus() Status {
	c.Lock()
	defer c.Unlock()
	return c.Status
}
