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
	cfg := config.Load()
	cfg.KeepAlive = keepaliveFlag
	fmt.Println("keep alive is", cfg.KeepAlive)
	// TODO: Maybe move this somewhere else?
	if portFlag > 0 {
		cfg.Port = portFlag
	}
	if interfaceFlag != "" {
		cfg.Interface = interfaceFlag
	}
	srv, err := server.New(cfg.Interface, cfg.Port, cfg.KeepAlive)
	if err != nil {
		return err
	}
	if err := srv.SetForSend(payload); err != nil {
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
