package glog

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestNewProcessor(t *testing.T) {
	NewProcessor("test")

	logrus.Info("4-")
	logrus.Info("5-")
	logrus.Info("6-")
}
func TestNewProcessor2(t *testing.T) {
	p := NewProcessor("test")
	cfg := NewWriterConfigLogstash()
	cfg.ZkHosts = []string{"bsw-ubuntu:2181"}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("a-")
	logrus.Info("b-")
	logrus.Info("c-")
	p.AddWriters(cfg)
	logrus.Info("a")
	logrus.Info("b")
	logrus.Info("c")
	i := 0
	for {
		logrus.Debug(fmt.Sprintf("%d", i))
		i++
		time.Sleep(time.Second)
	}
}
func testPanic() {
	panic("test")
}

func TestWithModule(t *testing.T) {
	NewProcessor("test")

	defer func() {
		if p := recover(); p != nil {
			WithModule("module").WithPanicStack().Warn(p)
		}
	}()

	testPanic()
}
