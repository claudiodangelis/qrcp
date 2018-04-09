package main

import (
	"os"
	"path/filepath"
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

// setupShouldBeZipped returns the paths to the created file and directory,
// or an error if it ran into issues in creation
func setupShouldBeZipped() (string, string, error) {
	testFile := "goTestFile.testing"
	testDir := "GoTestDir"
	file, err := os.Create(testFile)
	if err != nil {
		return "", "", err
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	filePath := filepath.Join(wd, file.Name())

	err = os.Mkdir(testDir, os.ModePerm)
	if err != nil {
		return "", "", err
	}

	dirPath := filepath.Join(wd, testDir)
	return filePath, dirPath, nil

}

func Test_shouldBeZipped(t *testing.T) {
	// Tests:
	// len(args) is > 1
	// nonexistent file (error from os.Stat)
	// passed file
	// passed directory
	testFile, testDir, err := setupShouldBeZipped()
	if err != nil {
		t.Errorf("Failed setting up for Test_shouldBeZipped.\n%s\n", err)
	}

	var zipTests = []struct {
		input        []string
		expectedResp bool
		expectedErr  bool
		failMsg      string
	}{
		{[]string{"args", ">", "size", "1"}, true, false, "Needs true if multiple files."},
		{[]string{"not real file"}, false, true, "Need false if non-existent."},
		{[]string{testFile}, false, false, "Need false if a file."},
		{[]string{testDir}, true, false, "Needs true if a directory."},
	}

	for _, test := range zipTests {
		output, err := shouldBeZipped(test.input)

		if test.expectedErr {
			assert.NotNil(t, err, "Should return error from os.Stat().")
		}
		assert.Equal(t, test.expectedResp, output, test.failMsg)
	}

	err = os.Remove(testFile)
	if err != nil {
		t.Errorf("Failed removing test file for Test_shouldBeZipped.\n%s\n", err)
	}

	err = os.Remove(testDir)
	if err != nil {
		t.Errorf("Failed removing test dir for Test_shouldBeZipped.\n%s\n", err)
	}
}
