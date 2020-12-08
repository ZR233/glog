package glog

import (
	"context"
	"fmt"
	"github.com/ZR233/glog/adaptor"
	"github.com/ZR233/glog/adaptor/file"
	"github.com/ZR233/glog/adaptor/logstash"
	"github.com/ZR233/glog/helper"
	"github.com/sirupsen/logrus"
	"os"
	"reflect"
	"sync"
	"time"
)

type TextFormatter struct {
}

func (t TextFormatter) Format(e *logrus.Entry) (o []byte, err error) {
	str := fmt.Sprintf("%s[%s] %s\n",
		e.Time.Format("2006/01/02 15:04:05"), e.Level, e.Message)
	o = []byte(str)
	return
}

// 日志处理类
type Processor struct {
	appName    string
	fileWriter *file.Core
	writers    []adaptor.Writer
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
}

var processor *Processor

type Entry struct {
	logrus.Entry
}

func NewWriterConfigLogstash() *adaptor.ConfigLogstash {
	return &adaptor.ConfigLogstash{}
}

func NewProcessor(appName string) *Processor {
	p := &Processor{
		appName: appName,
	}
	processor = p
	p.ctx, p.cancel = context.WithCancel(context.Background())
	logrus.SetFormatter(&TextFormatter{})
	p.fileWriter = &file.Core{}
	p.fileWriter.Run(nil, appName, p.ctx)
	logrus.AddHook(p.GetHook())

	return p
}

func (p *Processor) AddWriters(writerConfigs ...adaptor.WriterConfig) {
	for _, w := range writerConfigs {
		p.addWriter(w)
	}

	go p.workRetry()
}

func (p *Processor) addWriter(config adaptor.WriterConfig) {
	var writer adaptor.Writer
	switch config.(type) {
	case *adaptor.ConfigLogstash:
		writer = &logstash.Core{}

	default:
		panic(fmt.Errorf("type (%s) is not writer config", reflect.TypeOf(config).Name()))
	}
	writer.Run(config, p.appName, p.ctx)
	go p.workDealWriteFail(writer)

	p.writers = append(p.writers, writer)
}

func (p *Processor) workDealWriteFail(writer adaptor.Writer) {
	c := writer.WriteFail()
	for {
		select {
		case <-p.ctx.Done():
			return
		case log := <-c:
			p.fileWriter.Write(log)
		}
	}
}

type Hook struct {
	processor *Processor
}

func (l Hook) Fire(entry *logrus.Entry) error {
	l.processor.log(entry)
	return nil
}
func (l Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (p *Processor) GetHook() *Hook {
	hook := &Hook{}
	hook.processor = p
	return hook
}

func (p *Processor) log(entry *logrus.Entry) {
	entry.Data["app"] = p.appName
	entry.Data["hostname"], _ = os.Hostname()

	p.mu.Lock()
	writers := make([]adaptor.Writer, len(p.writers))
	copy(writers, p.writers)
	p.mu.Unlock()

	if len(writers) == 0 {
		p.fileWriter.Write(entry)
	} else {
		for _, w := range writers {
			w.Write(entry)
		}
	}
}

func (p *Processor) getOkWriters() (writers []adaptor.Writer) {
	p.mu.Lock()
	for _, w := range p.writers {
		if w.GetStatus() == adaptor.StatusOk {
			writers = append(writers, w)
		}
	}
	p.mu.Unlock()

	return
}

func (p *Processor) workRetry() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-time.After(time.Second):
			writers := p.getOkWriters()

			// writer 正常时重试
			if len(writers) > 0 {
				logs := p.fileWriter.RetryWrite()
				lCount := len(logs)
				if lCount > 0 {
					logrus.Debugf("[glog]write tmp log(%d)", len(logs))
					for _, log := range logs {
						for _, w := range writers {
							w.Write(log)
						}
					}
				}
			}
		}
	}
}
func WithModule(module string) *Entry {
	return &Entry{*logrus.WithField("module", module)}
}

// 函数在recover层级调用
func (e *Entry) WithPanicStack() *Entry {
	return &Entry{*e.WithField("stack", helper.StackTrace(3))}
}
