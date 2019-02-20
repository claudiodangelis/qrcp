package content

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudiodangelis/qr-filetransfer/util"
)

// Content represents the content to be transferred
type Content struct {
	Path string
	// Should the content be deleted from disk after transferring? This is true
	// only if the content has been zipped by qr-filetransfer
	ShouldBeDeleted bool
}

// Name returns the base name of the content being transferred
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
	tmpfile, err := ioutil.TempFile("", "qr-filetransfer")
	if err != nil {
		return "", err
	}
	zipWriter := zip.NewWriter(tmpfile)
	for _, item := range args {
		err = filepath.Walk(item, func(filePath string, info os.FileInfo, err error) error {
			// keep walking if directory is encountered
			if info.IsDir() {
				return nil
			}
			// stop walking if previously encountered error
			if err != nil {
				return err
			}
			relPath := strings.TrimPrefix(filePath, filepath.Dir(item))
			zipFile, err := zipWriter.Create(relPath)
			if err != nil {
				return err
			}
			fileForArchiving, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer fileForArchiving.Close()
			_, err = io.Copy(zipFile, fileForArchiving)
			if err != nil {
				return err
			}
			// keep walking
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	err = zipWriter.Close()
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	if err := os.Rename(tmpfile.Name(), tmpfile.Name()+".zip"); err != nil {
		return "", err
	}
	return tmpfile.Name() + ".zip", nil
}

// Get returns an instance of Content and an error
func Get(args []string) (Content, error) {
	content := Content{
		ShouldBeDeleted: false,
	}
	toBeZipped, err := util.ShouldBeZipped(args)
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
