package helper

import (
	"runtime/debug"
	"strings"
)

func StackTracePanic() (stack string) {
	return StackTrace(2)
}

func StackTrace(deep int) (stack string) {
	stack = string(debug.Stack())
	strArr := strings.Split(stack, ":\n")
	if len(strArr) < 2 {
		return
	}
	deep += 2

	stack = strArr[1]
	var rows []string
	strArr = strings.Split(stack, "\n")
	row := ""
	for i, line := range strArr {

		if i%2 == 0 {
			row = line
		} else {
			row += "\n" + line
			rows = append(rows, row)
		}
	}
	rowCount := len(rows)
	if deep >= rowCount {
		deep = rowCount - 1
	}

	rows = rows[deep:]

	return strings.Join(rows, "\n")
}
