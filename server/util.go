package server

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	"github.com/claudiodangelis/qrcp/pages"
)

var (
	tmplDone   = template.Must(template.New("done").Parse(pages.Done))
	tmplUpload = template.Must(template.New("upload").Parse(pages.Upload))
)

func serveTemplate(name string, w io.Writer, data interface{}) {
	var t *template.Template
	switch name {
	case "done":
		t = tmplDone
	case "upload":
		t = tmplUpload
	default:
		panic("unknown template: " + name)
	}
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

// getFileName generates a file name based on the existing files in the directory
// if name isn't taken leave it unchanged
// else change name to format "name(number).ext"
func getFileName(newFilename string, fileNamesInTargetDir []string) string {
	existing := make(map[string]struct{}, len(fileNamesInTargetDir))
	for _, name := range fileNamesInTargetDir {
		existing[name] = struct{}{}
	}
	if _, ok := existing[newFilename]; !ok {
		return newFilename
	}
	fileExt := filepath.Ext(newFilename)
	base := strings.TrimSuffix(newFilename, fileExt)
	for number := 1; ; number++ {
		candidate := fmt.Sprintf("%s(%d)%s", base, number, fileExt)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}
