// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
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
		drop, _ := cmd.Flags().GetInt("drop")
		corrupt, _ := cmd.Flags().GetInt("corrupt")
		allowEstablishment, _ := cmd.Flags().GetBool("allow-establishment")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		// prepare injection object
		spec := v1beta1.NetworkFailureSpec{
			Hosts:              hosts,
			Port:               port,
			Protocol:           protocol,
			Drop:               drop,
			Corrupt:            corrupt,
			AllowEstablishment: allowEstablishment,
		}
		i := injector.NewNetworkFailureInjector(uid, spec, c, log, ms)
		i.Inject()
	},
}

func init() {
	networkFailureInjectCmd.Flags().StringSlice("hosts", []string{}, "Hostname or IP address of the host to drop packets from and to")
	networkFailureInjectCmd.Flags().Int("port", 0, "Port to drop packets from and to")
	networkFailureInjectCmd.Flags().String("protocol", "", "Protocol to filter packets on (tcp or udp)")
	networkFailureInjectCmd.Flags().Int("drop", 100, "Percentage to drop packets (100 is a total drop)")
	networkFailureInjectCmd.Flags().Int("corrupt", 100, "Percentage to corrupt packets (100 is a total corruption)")
	networkFailureInjectCmd.Flags().Bool("allow-establishment", false, "If true, the packet drops only occur once the connection is established")

	_ = networkFailureInjectCmd.MarkFlagRequired("port")
	_ = networkFailureInjectCmd.MarkFlagRequired("protocol")
}
