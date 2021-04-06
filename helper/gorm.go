package helper

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

type LoggerGorm struct {
	Debug                               bool
	LogLevel                            logger.LogLevel
	traceStr, traceErrStr, traceWarnStr string
	SlowThreshold                       time.Duration
}

func NewLoggerGorm(slowThreshold time.Duration) *LoggerGorm {
	var (
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v]"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v]"
	)
	return &LoggerGorm{
		SlowThreshold: slowThreshold,
		traceStr:      traceStr,
		traceWarnStr:  traceWarnStr,
		traceErrStr:   traceErrStr,
		LogLevel:      logger.Warn,
	}
}

func (l *LoggerGorm) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l LoggerGorm) Info(_ context.Context, s string, i ...interface{}) {
	logrus.Infof(s, i...)
}

func (l LoggerGorm) Warn(_ context.Context, s string, i ...interface{}) {
	logrus.Warnf(s, i...)
}

func (l LoggerGorm) Error(_ context.Context, s string, i ...interface{}) {
	logrus.Errorf(s, i...)
}

func (l LoggerGorm) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	msg := ""
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		file := utils.FileWithLineNum()
		entry := logrus.WithFields(logrus.Fields{
			"execTime": elapsed.Milliseconds(),
			"file":     file,
			"sql":      sql,
		})

		if rows == -1 {
			msg = fmt.Sprintf(l.traceErrStr, file, err, float64(elapsed.Nanoseconds())/1e6, "-")
		} else {
			msg = fmt.Sprintf(l.traceErrStr, file, err, float64(elapsed.Nanoseconds())/1e6, rows)
		}
		entry.Error(msg)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		sql, rows := fc()
		file := utils.FileWithLineNum()
		entry := logrus.WithFields(logrus.Fields{
			"execTime": elapsed.Milliseconds(),
			"file":     file,
			"sql":      sql,
		})

		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			msg = fmt.Sprintf(l.traceWarnStr, file, slowLog, float64(elapsed.Nanoseconds())/1e6, "-")
		} else {
			msg = fmt.Sprintf(l.traceWarnStr, file, slowLog, float64(elapsed.Nanoseconds())/1e6, rows)
		}
		entry.Warn(msg)
	default:
		if l.Debug {
			sql, rows := fc()
			file := utils.FileWithLineNum()
			entry := logrus.WithFields(logrus.Fields{
				"execTime": elapsed.Milliseconds(),
				"file":     file,
			})

			if rows == -1 {
				msg = fmt.Sprintf(l.traceStr, file, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			} else {
				msg = fmt.Sprintf(l.traceStr, file, float64(elapsed.Nanoseconds())/1e6, rows, sql)
			}
			entry.Info(msg)
		}
	}
}
