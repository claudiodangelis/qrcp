package cmd

import (
	"github.com/spf13/cobra"
)

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	// TODO add usage
	RunE: func(command *cobra.Command, args []string) error {
		// TODO: Implement command
		return nil
	},
}
