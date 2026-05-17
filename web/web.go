package web

import _ "embed"

//go:embed send.html
var Send string

//go:embed upload.html
var Upload string

//go:embed done.html
var Done string
