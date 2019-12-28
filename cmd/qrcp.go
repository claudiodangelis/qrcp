package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.PersistentFlags().BoolVarP(&zipFlag, "zip", "z", false, "true if the content should be zipped before transfering")
}

// Flags
var zipFlag bool

var rootCmd = &cobra.Command{
	Use:  "qrcp",
	Args: cobra.ArbitraryArgs,
	RunE: sendCmdFunc,
}

// Execute the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
