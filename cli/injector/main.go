package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "chaos-fi",
	Short: "Datadog chaos failures injection application",
	Run:   nil,
}

func init() {
	rootCmd.AddCommand(networkFailureCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(networkLatencyCmd)
	rootCmd.PersistentFlags().String("uid", "", "UID of the failure resource")
	cobra.MarkFlagRequired(rootCmd.PersistentFlags(), "uid")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
