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
	cfg := config.Load()
	// Create the server
	srv, err := server.New(cfg.Interface, cfg.Port, false)
	if err != nil {
		return err
	}
	srv.ReceiveTo(outputFlag)
	log.Print(srv.ReceiveURL)
	qr.RenderString(srv.ReceiveURL)
	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	// TODO add usage
	RunE: receiveCmdFunc,
}
