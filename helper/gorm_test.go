package helper

import (
	"gorm.io/gorm/logger"
	"reflect"
	"testing"
	"time"
)

func TestNewLoggerGorm(t *testing.T) {
	type args struct {
		slowThreshold time.Duration
	}
	tests := []struct {
		name string
		args args
		want logger.Interface
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLoggerGorm(tt.args.slowThreshold); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLoggerGorm() = %v, want %v", got, tt.want)
			}
		})
	}
}
