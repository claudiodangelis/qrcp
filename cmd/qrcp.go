package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
	rootCmd.AddCommand(receiveCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	// Global command flags
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "only print errors")
	rootCmd.PersistentFlags().BoolVarP(&keepaliveFlag, "keep-alive", "k", false, "keep server alive after transferring")
	rootCmd.PersistentFlags().BoolVarP(&listallinterfacesFlag, "list-all-interfaces", "l", false, "list all available interfaces when choosing the one to use")
	rootCmd.PersistentFlags().IntVarP(&portFlag, "port", "p", 0, "port to use for the server")
	rootCmd.PersistentFlags().StringVar(&pathFlag, "path", "", "path to use. Defaults to a random string")
	rootCmd.PersistentFlags().StringVarP(&interfaceFlag, "interface", "i", "", "network interface to use for the server")
	rootCmd.PersistentFlags().StringVarP(&fqdnFlag, "fqdn", "d", "", "fully-qualified domain name to use for the resulting URLs")
	rootCmd.PersistentFlags().BoolVarP(&zipFlag, "zip", "z", false, "zip content before transferring")
	rootCmd.PersistentFlags().BoolVarP(&delFlag, "del", "D", false, "delete content after sending")
	// Receive command flags
	receiveCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "output directory for receiving files")
}

// Flags
var zipFlag bool
var portFlag int
var interfaceFlag string
var outputFlag string
var keepaliveFlag bool
var quietFlag bool
var fqdnFlag string
var pathFlag string
var listallinterfacesFlag bool
var delFlag bool

// The root command (`qrcp`) is like a shortcut of the `send` command
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
