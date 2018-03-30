package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getRandomSuffix(t *testing.T) {
	for i := 0; i < 10; i++ {
		output := getRandomSuffix()
		assert.Regexp(t,
			"[[:alnum:]][[:alnum:]][[:alnum:]][[:alnum:]]", output,
			"Output is not 4 alpha-numeric characters.")
	}
}
