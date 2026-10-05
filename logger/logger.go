package logger

import (
	"fmt"
	"os"

	"github.com/claudiodangelis/qrcp/util"
)

// Print prints its argument if the --quiet flag is not passed
func (l Logger) Print(args ...interface{}) {
	if l.quiet {
		return
	}

	if util.OutputIsPipe() {
		fmt.Fprintln(os.Stderr, args...)
	} else {
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
