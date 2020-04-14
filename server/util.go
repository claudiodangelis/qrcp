package server

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
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

// getFileName generates a file name based on the existing files in the directory
// if name isn't taken leave it unchanged
// else change name to format "name(number).ext"
func getFileName(newFilename string, fileNamesInTargetDir []string) string {
	fileExt := filepath.Ext(newFilename)
	fileName := strings.TrimSuffix(newFilename, fileExt)
	number := 1
	i := 0
	for i < len(fileNamesInTargetDir) {
		if newFilename == fileNamesInTargetDir[i] {
			newFilename = fmt.Sprintf("%s(%v)%s", fileName, number, fileExt)
			number++
			i = 0
		}
		i++
	}
	return newFilename
}
