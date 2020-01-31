package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkLatencyCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network latency",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		i := injector.NetworkLatencyInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
					Log: log,
				},
				ContainerID: containerID,
			},
			Spec: &v1beta1.NetworkLatencySpec{
				Hosts: hosts,
			},
		}
		i.Clean()
	},
}
