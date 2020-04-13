package cmd

import (
	"github.com/claudiodangelis/qrcp/config"
	"github.com/spf13/cobra"
)

func configCmdFunc(command *cobra.Command, args []string) error {
	return config.Wizard()
}

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c", "cfg"},
	RunE:    configCmdFunc,
}
