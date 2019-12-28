package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jhoonb/archivex"
)

// ZipFiles and return the resulting zip's filename
func ZipFiles(filenames []string) (string, error) {
	// TODO: Refactor to take []os.FileInfo rather than []string
	fmt.Println("Adding the following items to a zip file:",
		strings.Join(filenames, " "))
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
	for _, item := range filenames {
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
