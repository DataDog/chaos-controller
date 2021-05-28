// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func printSeparator() {
	fmt.Println("=======================================================================================================================================")
}

func explainMetaSpec(spec v1beta1.DisruptionSpec) {
	printSeparator()
	fmt.Println("🧰 has the following metadata  ...")

	if spec.DryRun {
		fmt.Println("\tℹ️  has DryRun set to true meaning no actual disruption is being run.")
	} else {
		fmt.Println("\tℹ️  has DryRun set to false meaning this disruption WILL run.")
	}

	if spec.Level == chaostypes.DisruptionLevelPod {
		fmt.Println("\tℹ️  will be run on the Pod level, so everything in this disruption is scoped at this level.")
	} else if spec.Level == chaostypes.DisruptionLevelNode {
		fmt.Println("\tℹ️  will be run on the Node level, so everything in this disruption is scoped at this level.")
	}

	if spec.Selector != nil {
		fmt.Printf("\tℹ️  has the following selectors which will be used to target %ss\n\t\t🎯  %s\n", spec.Level, spec.Selector.String())
	}

	if spec.Containers != nil {
		if spec.Level == chaostypes.DisruptionLevelNode {
			fmt.Println("\tℹ️  is using the node level. The Containers attribute only makes sense when using the pod level!")
		}

		fmt.Printf("\tℹ️  will target the following containers when targeting on the pod level\n\t\t🎯  %s\n", strings.Join(spec.Containers, ","))
	}

	fmt.Printf("\tℹ️  is going to target %s %s (either described as a percentage of total %ss or actual number of them).\n", spec.Count, spec.Level, spec.Level)
	printSeparator()
}

func explainNodeFailure(spec v1beta1.DisruptionSpec) {
	nodeFailure := spec.NodeFailure

	if nodeFailure == nil {
		return
	}

	if nodeFailure.Shutdown {
		fmt.Println("💉 injects a node failure which shuts down the host (violently) instead of triggering a kernel panic so the host is kept down and not restarted.")
	} else {
		fmt.Println("💉 injects a node failure which triggers a kernel panic on the node.")
	}

	printSeparator()
}

func explainCPUPressure(spec v1beta1.DisruptionSpec) {
	cpuPressure := spec.CPUPressure

	if cpuPressure == nil {
		return
	}

	fmt.Println("💉 injects a cpu pressure disruption ...")
	printSeparator()
}

func explainDiskPressure(spec v1beta1.DisruptionSpec) {
	diskPressure := spec.DiskPressure

	if diskPressure == nil {
		return
	}

	fmt.Println("💉 injects a disk pressure disruption ...")

	if diskPressure.Path == "" {
		fmt.Println("\t🗂  on path N/A")
	} else {
		fmt.Printf("\t🗂  on path %s\n", diskPressure.Path)
	}

	fmt.Println("\t🏃🏾‍♀️ with the following thresholds...")

	if diskPressure.Throttling.ReadBytesPerSec != nil {
		fmt.Printf("\t\t📖 %d read bytes per second\n", *diskPressure.Throttling.ReadBytesPerSec)
	}

	if diskPressure.Throttling.WriteBytesPerSec != nil {
		fmt.Printf("\t\t📝 %d write bytes per second\n", *diskPressure.Throttling.WriteBytesPerSec)
	}

	printSeparator()
}

func explainDNS(spec v1beta1.DisruptionSpec) {
	dns := spec.DNS

	if dns == nil || len(dns) == 0 {
		return
	}

	fmt.Println("💉 injects a dns disruption ...")
	fmt.Println("\t🥸  to spoof the following hostnames...")

	for _, data := range dns {
		fmt.Printf("\t\t👩🏽‍✈️ hostname: %s ...\n", data.Hostname)
		fmt.Printf("\t\t\t🧾 has type %s\n", data.Record.Type)
		fmt.Printf("\t\t\t🥷🏿  will be spoofed with %s\n", data.Record.Value)
	}

	printSeparator()
}

func explainNetworkFailure(spec v1beta1.DisruptionSpec) {
	network := spec.Network

	if network == nil {
		return
	}

	fmt.Println("💉 injects a network disruption ...")
	fmt.Println("\t💥  will apply filters so that network failures apply to outgoing/ingoing traffic from/to the following hosts/ports/protocols triplets:")

	for _, data := range network.Hosts {
		fmt.Printf("\t\t🎯 Host: %s\n", data.Host)
		fmt.Printf("\t\t\t⛵️ Port: %d\n", data.Port)
		fmt.Printf("\t\t\t🧾 Protocol: %s\n", data.Protocol)
	}

	fmt.Println("\t💥  will apply filters so that network failures apply to outgoing/ingoing traffic from/to the following services/namespaces pairs:")

	for _, data := range network.Services {
		fmt.Printf("\t\t🎯 Service: %s\n", data.Name)
		fmt.Printf("\t\t\t⛵️ Namespace: %s\n", data.Namespace)
	}

	if network.Flow == v1beta1.FlowIngress {
		fmt.Println("\t💥 applies network failures on incoming traffic instead of outgoing.")
	} else {
		fmt.Println("\t💥 applies network failures on outgoing traffic.")
	}

	if network.Drop != 0 {
		fmt.Printf("\t\t💣 applies a packet drop of %d percent.\n", network.Drop)
	}

	if network.Corrupt != 0 {
		fmt.Printf("\t\t💣 will corrupt packets at %d percent.\n", network.Drop)
	}

	if network.Delay != 0 {
		fmt.Printf("\t\t💣 applies a packet delay of %d ms.\n", network.Delay)

		if network.DelayJitter != 0 {
			fmt.Printf("\t\t\t💣 applies a jitter of %d ms to the delay value to add randomness to the delay.\n", network.Delay)
		}
	}

	if network.BandwidthLimit != 0 {
		fmt.Printf("\t\t💣 applies a bandwidth limit of %d ms.\n", network.BandwidthLimit)
	}

	printSeparator()
}

func explainMultiDisruption(spec v1beta1.DisruptionSpec) {
	existsMulti := false

	if spec.NodeFailure != nil {
		if spec.CPUPressure != nil || spec.DNS != nil || spec.DiskPressure != nil || spec.Network != nil {
			fmt.Println("⚠️  You are attempting to run a Node Failure Disruption in addition to another one of our other failures.\n" +
				"   Keep in mind that once the Node Failure runs (the kernel panic) the other disruptions will most likely not.")

			existsMulti = true
		}
	}

	if spec.Network != nil && spec.CPUPressure != nil {
		fmt.Println("⚠️  You are attempting to run a Network Disruption and a CPU Pressure Disruption.\n" +
			"   Keep in mind that CPU Pressure will most likely add additional issues to the network therefore your network disruption\n" +
			"   will be less exact in its defined failures.")

		existsMulti = true
	}

	if existsMulti {
		printSeparator()
	}
}

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "explains disruption config",
	Long:  `translates the yaml of the disruption configuration to english.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		explanation(path)
	},
}

func explanation(filePath string) {
	var disruption v1beta1.Disruption

	if filePath == "" {
		fmt.Println(pathError)
	}

	fullPath, _ := filepath.Abs(filePath)
	disruptionBytes, err := ioutil.ReadFile(fullPath)

	if err != nil {
		log.Printf("disruption.Get err   #%v ", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(disruptionBytes, &disruption)

	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	err = disruption.Spec.Validate()

	if err != nil {
		log.Fatalf("There were some problems when validating your disruption: %v", err)
	}

	fmt.Println("This Disruption...")

	explainMetaSpec(disruption.Spec)
	explainMultiDisruption(disruption.Spec)
	explainNodeFailure(disruption.Spec)
	explainNetworkFailure(disruption.Spec)
	explainCPUPressure(disruption.Spec)
	explainDiskPressure(disruption.Spec)
	explainDNS(disruption.Spec)
}

func init() {
	explainCmd.Flags().String("path", "", "The path to the disruption file to be explained.")
}
