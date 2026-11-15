package server

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/claudiodangelis/qrcp/pages"
)

// The pages are parsed once: they are constants, so failing to parse one is
// a bug, reported when qrcp starts (and by the tests) rather than in a handler
var (
	uploadPage = template.Must(template.New("upload").Parse(pages.Upload))
	donePage   = template.Must(template.New("done").Parse(pages.Done))
)

// serveTemplate writes a page. Failing to write it means the client went
// away, which must not prevent the handler from completing.
func serveTemplate(t *template.Template, w io.Writer, data interface{}) {
	if err := t.Execute(w, data); err != nil {
		log.Printf("Unable to send the %s page: %v", t.Name(), err)
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
