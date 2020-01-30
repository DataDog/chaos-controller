package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkFailureInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a network failure in the given container",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")
		port, _ := cmd.Flags().GetInt("port")
		protocol, _ := cmd.Flags().GetString("protocol")
		probability, _ := cmd.Flags().GetInt("probability")

		// Prepare injection object
		i := injector.NetworkFailureInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
				},
				ContainerID: containerID,
			},
			Spec: &v1beta1.NetworkFailureSpec{
				Hosts:       hosts,
				Port:        port,
				Protocol:    protocol,
				Probability: probability,
			},
		}
		i.Inject()
	},
}

func init() {
	networkFailureInjectCmd.Flags().StringSlice("hosts", []string{}, "Hostname or IP address of the host to drop packets from and to")
	networkFailureInjectCmd.Flags().Int("port", 0, "Port to drop packets from and to")
	networkFailureInjectCmd.Flags().String("protocol", "", "Protocol to filter packets on (tcp or udp)")
	networkFailureInjectCmd.Flags().Int("probability", 100, "Percentage of probability to drop packets (100 is a total drop)")

	_ = networkFailureInjectCmd.MarkFlagRequired("port")
	_ = networkFailureInjectCmd.MarkFlagRequired("protocol")
}
