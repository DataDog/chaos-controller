package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// helloCmd represents the hello command
var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "explains disruption config",
	Long:  `translates the yaml of the disruption configuration to english.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		fmt.Println(explanation(path))
	},
}

func explanation(filePath string) string {
	if filePath == "" {
		return "No Path Given, Exiting..."
	}
	return "Explanation TODO"
}

func init() {
	explainCmd.Flags().String("path", "", "The path to the disruption file to be explained.")
}
