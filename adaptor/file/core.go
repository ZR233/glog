package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ZR233/glog/adaptor"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Core struct {
	adaptor.ConfigLogstash
	logFile *os.File
	fmt     logrus.Formatter
	adaptor.WriterBase
}

func (c *Core) Write(entry *logrus.Entry) {
	c.Lock()
	defer c.Unlock()

	data, err := c.fmt.Format(entry)
	if err != nil {
		log.Println("entry format error: \n", err)
		return
	}
	_, err = c.logFile.Write(data)
	if err != nil {
		log.Println("entry write file error: \n", err)
		return
	}
}
func (c *Core) logFileName() string {
	return fmt.Sprintf("%s.log", c.AppName)
}

func (c *Core) Run(config adaptor.WriterConfig, prefix string, ctx context.Context) {
	c.fmt = &logrus.JSONFormatter{}
	c.AppName = prefix
	c.Ctx = ctx
	var err error
	c.logFile, err = os.OpenFile(c.logFileName(), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(fmt.Errorf("[glog.NewProcessor] create open file: %s\n%w",
			c.logFileName(), err))
	}
	_, _ = c.logFile.Seek(0, io.SeekEnd)
}
func (c *Core) RetryWrite() (logs []*logrus.Entry) {
	c.Lock()
	defer c.Unlock()

	_, _ = c.logFile.Seek(0, io.SeekStart)
	data, err := ioutil.ReadAll(c.logFile)
	if err != nil {
		log.Println("retry read file error: \n", err)
		return
	}
	rows := bytes.Split(data, []byte("\n"))
	for _, row := range rows {
		if len(row) > 0 {
			rowCase := map[string]interface{}{}

			err = json.Unmarshal(row, &rowCase)
			if err != nil {
				log.Println("retry read file error: \n", err)
				continue
			}

			entry := logrus.NewEntry(logrus.StandardLogger())

			if v, ok := rowCase[logrus.FieldKeyTime]; ok {
				t, _ := time.Parse(time.RFC3339, v.(string))
				entry.Time = t
				delete(rowCase, logrus.FieldKeyTime)
			}
			if v, ok := rowCase[logrus.FieldKeyLevel]; ok {
				t, _ := logrus.ParseLevel(v.(string))
				entry.Level = t
				delete(rowCase, logrus.FieldKeyLevel)
			}
			if v, ok := rowCase[logrus.FieldKeyMsg]; ok {
				entry.Message = v.(string)
				delete(rowCase, logrus.FieldKeyMsg)
			}
			entry.Data = rowCase

			logs = append(logs, entry)
		}
	}

	err = c.logFile.Truncate(0)
	if err != nil {
		log.Println("retry clean file error: \n", err)
	}
	_, _ = c.logFile.Seek(0, io.SeekEnd)
	return
}

func (c *Core) WriteFail() <-chan *logrus.Entry {
	panic("implement me")
}
