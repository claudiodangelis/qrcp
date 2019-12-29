package util

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/jhoonb/archivex"
)

// ZipFiles and return the resulting zip's filename
func ZipFiles(files []os.FileInfo) (string, error) {
	zip := new(archivex.ZipFile)
	tmpfile, err := ioutil.TempFile("", "qrcp")
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

// GetRandomURLPath returns a random string of 4 alphanumeric characters
func GetRandomURLPath() string {
	timeNum := time.Now().UTC().UnixNano()
	alphaString := strconv.FormatInt(timeNum, 36)
	return alphaString[len(alphaString)-4:]
}

// GetSessionID returns a base64 encoded string of 40 random characters
func GetSessionID() (string, error) {
	randbytes := make([]byte, 40)
	if _, err := io.ReadFull(rand.Reader, randbytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(randbytes), nil
}
