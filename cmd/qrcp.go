package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.PersistentFlags().BoolVarP(&zipFlag, "zip", "z", false, "zip content before transfering")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "only print errors")
	rootCmd.PersistentFlags().BoolVarP(&keepaliveFlag, "keep-alive", "k", false, "keep server alive after transfering")
	rootCmd.PersistentFlags().IntVarP(&portFlag, "port", "p", 0, "port to use for the server")
	rootCmd.PersistentFlags().StringVarP(&interfaceFlag, "interface", "i", "", "network interface to use for the server")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "output directory for receiving files")
}

// Flags
var zipFlag bool
var portFlag int
var interfaceFlag string
var outputFlag string
var keepaliveFlag bool
var quietFlag bool

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
