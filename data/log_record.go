package data

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Log struct {
	Time      time.Time `gorm:"index:idx_log_time"`
	Module    string    `gorm:"index"`
	Trace     string
	Level     logrus.Level `gorm:"index:idx_log_level"`
	Code      int          `gorm:"type:int;index:idx_log_code"`
	OptUserId int
	ExecTime  int
	Msg       string
	tableName string `gorm:"-"`
}

func (l Log) TableName() string {
	return l.tableName
}
