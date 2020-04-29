package qr

import (
	"github.com/skip2/go-qrcode"
)

// RenderString as a QR code
func RenderString(s string) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		println("Content too long for QRcode")
	}
	print(q.ToSmallString(false))
}
