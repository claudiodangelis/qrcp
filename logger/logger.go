package logger

import (
	"fmt"
	"log"
)

// Debug prints its argument if the -debug flag is passed
// and -quiet flag is not passed
func (l Logger) Debug(args ...string) {
	if l.quiet == false && l.debug == true {
		log.Println(args)
	}
}

// Info prints its argument if the -quiet flag is not passed
func (l Logger) Info(args ...interface{}) {
	if l.quiet == false {
		fmt.Println(args...)
	}
}

// Logger struct
type Logger struct {
	quiet bool
	debug bool
}

// New logger
func New() Logger {
	return Logger{}
}
