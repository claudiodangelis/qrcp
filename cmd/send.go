package cmd

import (
	"fmt"
	"os"

	"github.com/claudiodangelis/qrcp/qr"

	"github.com/claudiodangelis/qrcp/server"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) error {
	// Check if the content should be zipped
	shouldzip := len(args) > 1 || zipFlag
	var files []os.FileInfo
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
		files = append(files, file)
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
	fmt.Println(content)
	srv, err := server.Start("", 123)
	if err != nil {
		return err
	}
	srv.SetFilename(content)
	qr.RenderString(srv.SendURL)
	return nil
}

var sendCmd = &cobra.Command{
	Use: "send",
	// TODO: Add usage
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    sendCmdFunc,
}
