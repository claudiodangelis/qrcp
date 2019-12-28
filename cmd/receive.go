package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var receiveCmd = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"r"},
	Run: func(command *cobra.Command, args []string) {
		fmt.Println("receive!")
	},
}
