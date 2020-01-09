package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkLatencyInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject network latency in the given container",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		delay, _ := cmd.Flags().GetUint("delay")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		// Prepare injection object
		i := injector.NetworkLatencyInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
				},
				ContainerID: containerID,
			},
			Spec: &v1beta1.NetworkLatencyInjectionSpec{
				Delay: delay,
				Hosts: hosts,
			},
		}
		i.Inject()
	},
}

func init() {
	networkLatencyInjectCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	networkLatencyInjectCmd.MarkFlagRequired("delay")
}
