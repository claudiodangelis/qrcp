package cmd

import (
	"fmt"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/qr"

	"github.com/claudiodangelis/qrcp/server"
	"github.com/spf13/cobra"
)

func sendCmdFunc(command *cobra.Command, args []string) error {
	payload, err := payload.FromArgs(args, zipFlag)
	if err != nil {
		return err
	}
	cfg := config.New(command.Flags())
	srv, err := server.New(cfg.Interface, cfg.Port, keepaliveFlag)
	if err != nil {
		return err
	}
	if err := srv.Send(payload); err != nil {
		return err
	}
	fmt.Println(srv.SendURL)
	qr.RenderString(srv.SendURL)
	if err := srv.Wait(); err != nil {
		return err
	}
	return nil
}

var sendCmd = &cobra.Command{
	Use: "send",
	// TODO: Add usage
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    sendCmdFunc,
}
