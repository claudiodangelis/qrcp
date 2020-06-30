package qr

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

// RenderString as a QR code
func RenderString(s string) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		log.Panic(err)
	}

	if runtime.GOOS == "windows" {
		//create png from string
		png, err := qrcode.Encode(s, qrcode.Medium, 256)
		if err != nil {
			log.Panic(err)
		}
		//create temp file in temp dir
		f, err := ioutil.TempFile(os.TempDir(), "qrcp*.png")
		if err != nil {
			log.Panic(err)
		}
		//remove temp file after use
		defer os.Remove(f.Name())

		//write png to temp file
		_, err = f.Write(png)
		if err != nil {
			log.Panic(err)
		}
		//open temp file using native file-handler
		exec.Command("rundll32", "url.dll,FileProtocolHandler", f.Name()).Start()
	}

	fmt.Println(q.ToSmallString(false))
}
