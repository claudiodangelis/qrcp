package payload

import (
	"os"
	"path/filepath"

	"github.com/claudiodangelis/qrcp/util"
)

// Payload to transfer
type Payload struct {
	Filename            string
	Path                string
	DeleteAfterTransfer bool
}

// Delete the payload from disk
func (p Payload) Delete() error {
	return os.RemoveAll(p.Path)
}

// FromArgs returns a payload from args
func FromArgs(args []string, zipFlag bool) (Payload, error) {
	shouldzip := len(args) > 1 || zipFlag
	var files []string
	// Check if content exists
	for _, arg := range args {
		file, err := os.Stat(arg)
		if err != nil {
			return Payload{}, err
		}
		// If at least one argument is dir, the content will be zipped
		if file.IsDir() {
			shouldzip = true
		}
		files = append(files, arg)
	}
	// Prepare the content
	// TODO: Research cleaner code
	var content string
	if shouldzip {
		zip, err := util.ZipFiles(files)
		if err != nil {
			return Payload{}, err
		}
		content = zip
	} else {
		content = args[0]
	}
	return Payload{
		Path:                content,
		Filename:            filepath.Base(content),
		DeleteAfterTransfer: shouldzip,
	}, nil
}
