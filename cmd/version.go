package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.5.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version number",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println("qrcp", version)
	},
}
