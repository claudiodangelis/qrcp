package logger

import (
	"fmt"
)

// Print prints its argument if the --quiet flag is not passed
func (l Logger) Print(args ...interface{}) {
	if !l.quiet {
		fmt.Println(args...)
	}
}

// Logger struct
type Logger struct {
	quiet bool
}

// New logger
func New(quiet bool) Logger {
	return Logger{
		quiet: quiet,
	}
}
