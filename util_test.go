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

func Test_shouldBeZipped(t *testing.T) {
	// Tests:
	// len(args) is > 1
	// nonexistent file (error from os.Stat)
	// passed file
	// passed directory
	var zipTests = []struct {
		input        []string
		expectedResp bool
		expectedErr  bool
		failMsg      string
	}{
		{[]string{"args", ">", "size", "1"}, true, false, "Needs true if multiple files."},
		{[]string{"not real file"}, false, true, "Need false if non-existent."},
		{[]string{"~/.bashrc"}, false, false, "Need false if a file."},
		{[]string{"/"}, true, false, "Needs true if a directory."},
	}

	for _, test := range zipTests {
		output, err := shouldBeZipped(test.input)

		if test.expectedErr {
			assert.NotNil(t, err, "Should return error from os.Stat().")
		}
		assert.Equal(t, test.expectedResp, output, test.failMsg)
	}
}
