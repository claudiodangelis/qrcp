package cmd

import (
	"fmt"
	"os"

	"github.com/claudiodangelis/qrcp/util"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) {
	// Check if the content should be zipped
	shouldzip := len(args) > 1 || zipFlag
	var files []os.FileInfo
	// Check if content exists
	for _, arg := range args {
		file, err := os.Stat(arg)
		if err != nil {
			panic(err)
		}
		// If at least one argument is dir, the content will be zipped
		if file.IsDir() {
			shouldzip = true
		}
		files = append(files, file)
	}
	// Prepare the content
	var content string
	if shouldzip {
		zip, err := util.ZipFiles(files)
		if err != nil {
			panic(err)
		}
		content = zip
	} else {
		content = args[0]
	}
	// Prepare the server
	fmt.Println(content)
}

var sendCmd = &cobra.Command{
	Use:     "send",
	Aliases: []string{"s"},
	Run:     sendCmdFunc,
}
