package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.PersistentFlags().BoolVarP(&zipFlag, "zip", "z", false, "true if the content should be zipped before transfering")
	rootCmd.PersistentFlags().IntVarP(&portFlag, "port", "p", 0, "port to use for the server")
}

// Flags
var zipFlag bool
var portFlag int
var rootCmd = &cobra.Command{
	Use:  "qrcp",
	Args: cobra.MinimumNArgs(1),
	RunE: sendCmdFunc,
}

// Execute the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
