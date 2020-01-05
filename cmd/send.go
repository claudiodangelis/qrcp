package cmd

import (
	"os"
	"path/filepath"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/qr"

	"github.com/claudiodangelis/qrcp/server"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) error {
	cfg := config.Load()
	// Check if the content should be zipped
	shouldzip := len(args) > 1 || zipFlag
	var files []string
	// Check if content exists
	for _, arg := range args {
		file, err := os.Stat(arg)
		if err != nil {
			return err
		}
		// If at least one argument is dir, the content will be zipped
		if file.IsDir() {
			shouldzip = true
		}
		files = append(files, arg)
	}
	// Prepare the content
	// TODO: Make less ugly
	var content string
	if shouldzip {
		zip, err := util.ZipFiles(files)
		if err != nil {
			return err
		}
		content = zip
	} else {
		content = args[0]
	}
	// Prepare the server
	if portFlag > 0 {
		cfg.Port = portFlag
	}
	if interfaceFlag != "" {
		cfg.Interface = interfaceFlag
	}
	payload := payload.Payload{
		Path:                content,
		Filename:            filepath.Base(content),
		DeleteAfterTransfer: shouldzip,
	}
	srv, err := server.Start(&cfg, &payload)
	if err != nil {
		return err
	}
	qr.RenderString(srv.SendURL)
	srv.Wait()
	return nil
}

var sendCmd = &cobra.Command{
	Use: "send",
	// TODO: Add usage
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    sendCmdFunc,
}
