package server

import (
	"html/template"
	"io"
)

func serveTemplate(name string, tmpl string, w io.Writer, data interface{}) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		panic(err)
	}
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}
