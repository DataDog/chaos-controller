package main

import (
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkFailureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network failures",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")

		i := injector.NetworkFailureInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
				},
				ContainerID: containerID,
			},
		}
		i.Clean()
	},
}
