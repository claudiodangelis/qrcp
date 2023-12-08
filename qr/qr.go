package qr

import (
	"fmt"
	"image"
	"log"

	"github.com/skip2/go-qrcode"
)

// RenderString as a QR code
func RenderString(s string, inverseColor bool) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(q.ToSmallString(inverseColor))
}

// RenderImage returns a QR code as an image.Image
func RenderImage(s string) image.Image {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		log.Fatal(err)
	}
	return q.Image(256)
}
