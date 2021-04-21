package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// helloCmd represents the hello command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a disruption.",
	Long:  `creates a disruption given input from the user.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		fmt.Println(create(path))
	},
}

func create(filePath string) string {
	if filePath == "" {
		return "No Path Given, Exiting..."
	}
	return "Creation TODO"
}

func init() {
	createCmd.Flags().String("path", "", "The path to create the disruption config.")
}
