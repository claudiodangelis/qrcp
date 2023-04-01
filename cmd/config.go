package cmd

import (
	"fmt"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/spf13/cobra"
)

func configCmdFunc(command *cobra.Command, args []string) error {
	return config.Wizard(app)
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Configure qrcp",
	Long:    "Run an interactive configuration wizard for qrcp. With this command you can configure which network interface and port should be used to create the file server.",
	Aliases: []string{"c", "cfg"},
	RunE:    configCmdFunc,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate the legacy configuration file",
	Long:  "Migrate the legacy JSON configuration file to the new YAML format",
	Run: func(cmd *cobra.Command, args []string) {
		ok, err := config.Migrate(app)
		if err != nil {
			fmt.Println("error while migrating the legacy JSON configuration file:", err)
		}
		if ok {
			fmt.Println("Legacy JSON configuration file has been successfully deleted")
		}
	},
}
