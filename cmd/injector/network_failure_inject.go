package main

import (
	"github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/injector"
	"github.com/spf13/cobra"
)

var networkFailureInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a network failure in the given container",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		host, _ := cmd.Flags().GetString("host")
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
			Spec: &v1beta1.NetworkFailureInjectionSpec{
				Failure: v1beta1.NetworkFailureInjectionSpecFailure{
					Host:        host,
					Port:        port,
					Protocol:    protocol,
					Probability: probability,
				},
			},
		}
		i.Inject()
	},
}

func init() {
	networkFailureInjectCmd.Flags().String("host", "0.0.0.0/0", "Hostname or IP address of the host to drop packets from and to")
	networkFailureInjectCmd.Flags().Int("port", 0, "Port to drop packets from and to")
	networkFailureInjectCmd.Flags().String("protocol", "", "Protocol to filter packets on (tcp or udp)")
	networkFailureInjectCmd.Flags().Int("probability", 100, "Percentage of probability to drop packets (100 is a total drop)")

	networkFailureInjectCmd.MarkFlagRequired("port")
	networkFailureInjectCmd.MarkFlagRequired("protocol")
}
