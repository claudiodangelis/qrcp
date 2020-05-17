package cmd

import (
	"github.com/claudiodangelis/qrcp/config"
	"github.com/spf13/cobra"
)

func configCmdFunc(command *cobra.Command, args []string) error {
	return config.Wizard(configFlag, listallinterfacesFlag)
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Configure qrcp",
	Long:    "Run an interactive configuration wizard for qrcp. With this command you can configure which network interface and port should be used to create the file server.",
	Aliases: []string{"c", "cfg"},
	RunE:    configCmdFunc,
}
