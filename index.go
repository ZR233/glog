package glog

import (
	"context"
	"fmt"
	"github.com/ZR233/glog/v2/adaptor"
	"github.com/ZR233/glog/v2/adaptor/file"
	"github.com/ZR233/glog/v2/adaptor/logstash"
	"github.com/ZR233/glog/v2/helper"
	"github.com/sirupsen/logrus"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	FieldKeyModule = "module"
)

var processor *Processor

func Init(appName, module string) {
	processor = NewProcessor(appName, module)
}

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
	adaptor.CoreBase
	fileWriter *file.Core
	writers    []adaptor.Writer
	cancel     context.CancelFunc
	mu         sync.Mutex
}

type Entry struct {
	logrus.Entry
}

func NewWriterConfigLogstash() *adaptor.ConfigLogstash {
	return &adaptor.ConfigLogstash{}
}
func UseKafka(zkHosts []string) {
	cfg := NewWriterConfigLogstash()
	cfg.ZkHosts = zkHosts
	processor.AddWriters(cfg)
}

func NewProcessor(appName, modulePrefix string) *Processor {
	p := &Processor{}
	p.AppName = appName
	p.ModulePrefix = modulePrefix

	p.Ctx, p.cancel = context.WithCancel(context.Background())
	logrus.SetFormatter(&TextFormatter{})
	p.fileWriter = &file.Core{}
	p.fileWriter.Run(nil, &p.CoreBase)
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
	writer.Run(config, &p.CoreBase)
	go p.workDealWriteFail(writer)

	p.writers = append(p.writers, writer)
}

func (p *Processor) workDealWriteFail(writer adaptor.Writer) {
	c := writer.WriteFail()
	for {
		select {
		case <-p.Ctx.Done():
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
	entry.Data["app"] = p.AppName
	entry.Data["hostname"], _ = os.Hostname()
	modulePrefix := p.ModulePrefix
	if v, ok := entry.Data[FieldKeyModule]; ok {
		if s, ok2 := v.(string); ok2 {
			modulePrefix = strings.Join([]string{modulePrefix, s}, ".")
		}
	}
	entry.Data[FieldKeyModule] = modulePrefix

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
		case <-p.Ctx.Done():
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
