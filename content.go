package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhoonb/archivex"
)

// Content represents the content to be transfered
type Content struct {
	Path string
	// Should the content be deleted from disk after transfering? This is true
	// only if the content has been zipped by qr-filetransfer
	ShouldBeDeleted bool
}

// Name returns the base name of the content being transfered
func (c *Content) Name() string {
	return filepath.Base(c.Path)
}

// Delete the file from disk
func (c *Content) Delete() error {
	return os.Remove(c.Path)
}

// zipContent creates a new zip archive that stores the passed paths.
// It returns the path to the newly created zip file, and an error
func zipContent(args []string) (string, error) {
	fmt.Println("Adding the following items to a zip file:",
		strings.Join(args, " "))
	zip := new(archivex.ZipFile)
	tmpfile, err := ioutil.TempFile("", "qr-filetransfer")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	if err := os.Rename(tmpfile.Name(), tmpfile.Name()+".zip"); err != nil {
		return "", err
	}
	zip.Create(tmpfile.Name() + ".zip")
	for _, item := range args {
		f, err := os.Stat(item)
		if err != nil {
			return "", err
		}
		if f.IsDir() == true {
			zip.AddAll(item, true)
		} else {
			zip.AddFile(item)
		}
	}
	if err := zip.Close(); err != nil {
		return "", nil
	}
	return zip.Name, nil
}

// getContent returns an instance of Content and an error
func getContent(args []string) (Content, error) {
	content := Content{
		ShouldBeDeleted: false,
	}
	toBeZipped, err := shouldBeZipped(args)
	if err != nil {
		return content, err
	}
	if toBeZipped {
		content.ShouldBeDeleted = true
		content.Path, err = zipContent(args)
		if err != nil {
			return content, err
		}
	} else {
		content.Path = args[0]
	}
	return content, nil
}
