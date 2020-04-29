package qr

import (
	"github.com/skip2/go-qrcode"
)

// RenderString as a QR code
func RenderString(s string) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		panic(err)
	}
	print(q.ToSmallString(false))
}
