package cmd

import (
	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/logger"
	"github.com/claudiodangelis/qrcp/qr"
	"github.com/claudiodangelis/qrcp/server"
	"github.com/spf13/cobra"
)

func receiveCmdFunc(command *cobra.Command, args []string) error {
	log := logger.New(quietFlag)
	// Load configuration
	cfg, err := config.New(configFileFlag, interfaceFlag, portFlag, pathFlag, fqdnFlag, keepaliveFlag, listallinterfacesFlag)
	if err != nil {
		return err
	}
	// Create the server
	srv, err := server.New(&cfg)
	if err != nil {
		return err
	}
	// Sets the output directory
	if err := srv.ReceiveTo(outputFlag); err != nil {
		return err
	}
	// Prints the URL to scan to screen
	log.Print("Scan the following URL with a QR reader to start the file transfer:")
	log.Print(srv.ReceiveURL)
	// Renders the QR
	qr.RenderString(srv.ReceiveURL)
	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	Short:   "Receive one or more files",
	Long:    "Receive one or more files. If not specified with the --output flag, the current working directory will be used as a destination.",
	Example: `# Receive files in the current directory
qrcp receive
# Receive files in a specific directory
qrcp receive --output /tmp
`,
	RunE: receiveCmdFunc,
}
