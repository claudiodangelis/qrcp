package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContent(t *testing.T) {
	c := Content{
		"/path/to/file.txt",
		false,
	}

	expected := "file.txt"
	output := c.Name()
	if expected != output {
		t.Errorf("Expected %s but got %s\n", expected, output)
	}
}

func Test_getContent(t *testing.T) {
	testFile := "goTestFile.txt"
	_, err := os.Create(testFile)
	if err != nil {
		t.Error("Failed setting up file for Test_getContent.")
	}

	var cases = []struct {
		input       []string
		expectedErr bool
		msg         string
	}{
		{[]string{"/fake/file", "/also/fake/file"}, true, "Expected error with non-existent files."},
		{[]string{testFile}, false, "Expected valid Content object."},
	}

	for _, test := range cases {
		c, err := getContent(test.input)

		if test.expectedErr {
			assert.NotNil(t, err, test.msg)
		} else {
			assert.Nil(t, err, "Expected no error.")
			assert.IsType(t, Content{}, c, test.msg)
		}
	}

	err = os.Remove(testFile)
	if err != nil {
		t.Error("Failed removing test file for Test_getContent.")
	}
}
