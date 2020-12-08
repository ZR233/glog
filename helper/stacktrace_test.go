package helper

import (
	"testing"
	"time"
)

func TestStackTrace(t *testing.T) {
	st := StackTrace(0)
	println(st)
}
func testDeep2() {
	testDeep1()
}
func testDeep1() {
	panic("test")
}

func TestStackTrace1(t *testing.T) {
	go func() {
		for {
			time.Sleep(time.Second)
		}
	}()
	go func() {
		for {
			time.Sleep(time.Second)
		}
	}()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				st := StackTracePanic()
				println(st)
			}
		}()
		testDeep2()
	}()
	time.Sleep(time.Second)

}
