package util

import (
	"fmt"
	"runtime/debug"
)

func PrintStackAndPanic(err error) {
	fmt.Printf("Error %s Stack trace:\n%s", err, debug.Stack())
	panic("PANIC")
}
