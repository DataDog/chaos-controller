package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "local-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "display chaosli version",
	Long:  `shows the currently used version of the chaosli - upgrade with brew if necessary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("chaosli", Version)
	},
}
