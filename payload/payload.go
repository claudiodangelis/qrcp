package payload

import (
	"log"
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

func dirInArgs(args []string) bool {
	for _, a := range args {
		f, err := os.Open(a)
		if err != nil {
			log.Fatalf("%v", err)
		}
		s, err := f.Stat()
		if err != nil {
			log.Fatalf("%v", err)
		}
		if s.IsDir() {
			return true
		}
	}
	return false
}

// FromArgs returns payloads from args
func FromArgs(args []string, zipFlag bool, delFlag bool) ([]Payload, error) {
	payloads := make([]Payload, 0)
	if dirInArgs(args) || zipFlag {
		archive, err := util.ZipFiles(args)
		if err != nil {
			return []Payload{}, err
		}
		payloads = append(payloads, Payload{Path: archive, Filename: filepath.Base(archive), DeleteAfterTransfer: delFlag})
	} else {
		for _, f := range args {
			payloads = append(payloads, Payload{Path: f, Filename: filepath.Base(f), DeleteAfterTransfer: delFlag})
		}
	}
	return payloads, nil
}
