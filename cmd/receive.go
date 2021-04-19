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
	configOptions := config.Options{
		Interface:         interfaceFlag,
		Port:              portFlag,
		Path:              pathFlag,
		FQDN:              fqdnFlag,
		KeepAlive:         keepaliveFlag,
		ListAllInterfaces: listallinterfacesFlag,
		Secure:            secureFlag,
		TLSCert:           tlscertFlag,
		TLSKey:            tlskeyFlag,
		Output:            outputFlag,
	}
	cfg, err := config.New(configFlag, configOptions)
	if err != nil {
		return err
	}
	// Create the server
	srv, err := server.New(&cfg)
	if err != nil {
		return err
	}
	// Sets the output directory
	if err := srv.ReceiveTo(cfg.Output); err != nil {
		return err
	}
	// Prints the URL to scan to screen
	log.Print("Scan the following URL with a QR reader to start the file transfer:")
	log.Print(srv.ReceiveURL)
	// Renders the QR
	qr.RenderString(srv.ReceiveURL)
	if browserFlag {
		srv.DisplayQR(srv.ReceiveURL)
	}
	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	Short:   "Receive one or more files",
	Long:    "Receive one or more files. The destination directory can be set with the config wizard, or by passing the --output flag. If none of the above are set, the current working directory will be used as a destination directory.",
	Example: `# Receive files in the current directory
qrcp receive
# Receive files in a specific directory
qrcp receive --output /tmp
`,
	RunE: receiveCmdFunc,
}
