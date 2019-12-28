package util

import (
	"io/ioutil"
	"os"

	"github.com/jhoonb/archivex"
)

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []os.FileInfo) (string, error) {
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
	for _, file := range files {
		if file.IsDir() {
			zip.AddAll(file.Name(), true)
		} else {
			zip.AddFile(file.Name())
		}
	}
	if err := zip.Close(); err != nil {
		return "", nil
	}
	return zip.Name, nil
}
