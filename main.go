package main

import (
	"os"

	"github.com/claudiodangelis/qrcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
