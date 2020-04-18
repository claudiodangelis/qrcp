package main

import (
	"log"

	"github.com/claudiodangelis/qrcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if err.Error() == "^C" {
			return
		}
		log.Fatalln(err)
	}
}
