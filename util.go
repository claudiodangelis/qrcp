package main

import (
	"log"
	"os"
)

// debug prints its argument if the -debug flag is passed
func debug(args ...string) {
	if *debugFlag == true {
		log.Println(args)
	}
}

// shouldBeZipped returns a boolean value indicating if the
// content should be zipped or not, and an error.
// The content should be zipped if:
// 1. the user passed the `-zip` flag
// 2. there are more than one file
// 3. the file is a directory
func shouldBeZipped(args []string) (bool, error) {
	if *zipFlag == true {
		return true, nil
	}
	if len(args) > 1 {
		return true, nil
	}
	file, err := os.Stat(args[0])
	if err != nil {
		return false, err
	}
	if file.IsDir() {
		return true, nil
	}
	return false, nil
}
