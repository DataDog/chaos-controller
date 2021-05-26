// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/labels"
)

type NodeFailure struct {
	Shutdown *bool `yaml:"shutdown"`
}

type Network struct {
	Hosts          []string `yaml:"hosts"`
	Port           *int     `yaml:"port"`
	Protocol       *string  `yaml:"protocol"`
	Flow           *string  `yaml:"flow"`
	Drop           *int     `yaml:"drop"`
	Corrupt        *int     `yaml:"corrupt"`
	Delay          *int     `yaml:"delay"`
	DelayJitter    *int     `yaml:"delayJitter"`
	BandwidthLimit *int     `yaml:"bandwidthLimit"`
}

type CPUPressure struct{}

type Throttling struct {
	ReadBytesPerSec  *int `yaml:"readBytesPerSec"`
	WriteBytesPerSec *int `yaml:"writeBytesPerSec"`
}
type DiskPressure struct {
	Path       *string     `yaml:"path"`
	Throttling *Throttling `yaml:"throttling"`
}

type Record struct {
	Type  *string `yaml:"type"`
	Value *string `yaml:"value"`
}
type DNS struct {
	Hostname *string `yaml:"hostname"`
	Record   *Record `yaml:"record"`
}

type Spec struct {
	DryRun       *bool         `yaml:"dryRun"`
	Level        *string       `yaml:"level"`
	Selector     labels.Set    `yaml:"selector"`
	Containers   []string      `yaml:"containers"`
	Count        *string       `yaml:"count"`
	NodeFailure  *NodeFailure  `yaml:"nodeFailure"`
	Network      *Network      `yaml:"network"`
	CPUPressure  *CPUPressure  `yaml:"cpuPressure"`
	DiskPressure *DiskPressure `yaml:"diskPressure"`
	DNS          []DNS         `yaml:"dns"`
}

type DisruptionSpec struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec Spec `yaml:"spec"`
}

type Explanations struct {
	seperator string
}

func (e *Explanations) printSeperator() {
	fmt.Println(e.seperator)
}

func (e *Explanations) explainMetaSpec(spec Spec) {
	e.printSeperator()
	fmt.Println("🧰 has the following metadata  ...")
	if spec.DryRun != nil {
		if *spec.DryRun == true {
			fmt.Println("\tℹ️  has DryRun set to true meaning no actual disruption is being run.")
		} else {
			fmt.Println("\tℹ️  has DryRun set to false meaning this disruption WILL run.")
		}
	}

	level := "pod/node"
	if spec.Level != nil {
		if *spec.Level == "pod" {
			level = "pod"
			fmt.Println("\tℹ️  will be run on the Pod level, so everything in this disruption is scoped at this level.")
		} else if *spec.Level == "node" {
			level = "node"
			fmt.Println("\tℹ️  will be run on the Node level, so everything in this disruption is scoped at this level.")
		} else {
			fmt.Println("\tℹ️  level is unknown and will most likely cause errors when applied.")
		}
	}

	if spec.Selector != nil {
		fmt.Printf("\tℹ️  has the following selectors which will be used to target %ss\n\t\t🎯  %s\n", level, spec.Selector.String())
	}

	if spec.Containers != nil {
		if level == "node" {
			fmt.Println("\tℹ️  is using the node level. The Containers attribute only makes sense when using the pod level!")
		}
		fmt.Printf("\tℹ️  will target the following containers when targeting on the pod level\n\t\t🎯  %s\n", strings.Join(spec.Containers, ","))
	}

	if spec.Count != nil {
		fmt.Printf("\tℹ️  is going to target %s %s (either described as a percentage of total %ss or actual number of them).\n", *spec.Count, level, level)
	}
	e.printSeperator()
}

func (e *Explanations) explainNodeFailure(spec Spec) {
	nodeFailure := spec.NodeFailure
	if nodeFailure == nil {
		return
	}
	if nodeFailure.Shutdown != nil {
		if *nodeFailure.Shutdown == true {
			fmt.Println("💉 injects a node failure which shuts down the host (violently) instead of triggering a kernel panic so the host is kept down and not restarted.")
		} else {
			fmt.Println("💉 injects a node failure which triggers a kernel panic on the node.")
		}
	}
	e.printSeperator()
}

func (e *Explanations) explainCPUPressure(spec Spec) {
	cpuPressure := spec.CPUPressure
	if cpuPressure == nil {
		return
	}
	fmt.Println("💉 injects a cpu pressure disruption ...")
	e.printSeperator()
}

func (e *Explanations) explainDiskPressure(spec Spec) {
	diskPressure := spec.DiskPressure
	if diskPressure == nil {
		return
	}
	fmt.Println("💉 injects a disk pressure disruption ...")
	if diskPressure.Path == nil {
		fmt.Println("\t⚠️ no path specified for disk disruption, this is required!!")
	} else {
		fmt.Printf("\t🗂  on path %s\n", *diskPressure.Path)
	}
	if diskPressure.Throttling == nil {
		fmt.Println("\t⚠️ no throttling specified for disk disruption, this is required!!")
	} else {
		fmt.Println("\t🏃🏾‍♀️ with the following thresholds...")
		if diskPressure.Throttling.ReadBytesPerSec != nil {
			fmt.Printf("\t\t📖 %d read bytes per second\n", *diskPressure.Throttling.ReadBytesPerSec)
		}
		if diskPressure.Throttling.WriteBytesPerSec != nil {
			fmt.Printf("\t\t📝 %d write bytes per second\n", *diskPressure.Throttling.WriteBytesPerSec)
		}
	}
	e.printSeperator()
}

func (e *Explanations) explainDNS(spec Spec) {
	dns := spec.DNS
	if dns == nil || len(dns) == 0 {
		return
	}
	fmt.Println("💉 injects a dns disruption ...")
	fmt.Println("\t🥸  to spoof the following hostnames...")
	for _, data := range dns {
		fmt.Printf("\t\t👩🏽‍✈️ hostname: %s ...\n", *data.Hostname)
		fmt.Printf("\t\t\t🧾 has type %s\n", *data.Record.Type)
		fmt.Printf("\t\t\t🥷🏿  will be spoofed with %s\n", *data.Record.Value)
	}
	e.printSeperator()
}

func (e *Explanations) explainNetworkFailure(spec Spec) {
	network := spec.Network
	if network == nil {
		return
	}
	fmt.Println("💉 injects a network disruption ...")
	if network.Hosts == nil || len(network.Hosts) == 0 {
		fmt.Println("\t💥 specified no hosts meaning that no filtering on host will be done when applying network failures.")
	} else {
		fmt.Printf("\t💥 applies filters so that network failures only apply to outgoing/ingoing traffic from/to the following hosts:\n\t\t🎯 %s\n", strings.Join(network.Hosts, ","))
	}

	if network.Port == nil {
		fmt.Println("\t💥 specified no port meaning that no filtering on port will be done when applying network failures.")
	} else {
		fmt.Printf("\t💥 applies filters so that network failures only apply to port %d\n", *network.Port)
	}

	if network.Protocol == nil {
		fmt.Println("\t💥 specifies no Protocol, so by default all protocol are applied (tcp,udp,etc.)")
	} else {
		fmt.Printf("\t💥 applies filters so that network failures only apply packets with the %s protocol\n", *network.Protocol)
	}

	if network.Flow != nil && *network.Flow == "ingress" {
		fmt.Println("\t💥 applies network failures on incoming traffic to the targets only.")
	} else {
		fmt.Println("\t💥 applies network failures on outgoing traffic to the targets only.")
	}

	if network.Drop != nil {
		fmt.Printf("\t\t💣 applies a packet drop of %d percent.\n", *network.Drop)
	}
	if network.Corrupt != nil {
		fmt.Printf("\t\t💣 will corrupt packets at %d percent.\n", *network.Drop)
	}
	if network.Delay != nil {
		fmt.Printf("\t\t💣 applies a packet delay of %d ms.\n", *network.Delay)
		if network.DelayJitter != nil {
			fmt.Printf("\t\t\t💣 applies a jitter of %d ms to the delay value to add randomness to the delay.\n", *network.Delay)
		}
	}
	if network.BandwidthLimit != nil {
		fmt.Printf("\t\t💣 applies a bandwidth limit of %d ms.\n", *network.BandwidthLimit)
	}
	e.printSeperator()
}

func (e *Explanations) explainMultiDisruption(spec Spec) {
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
		e.printSeperator()
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
	var disruptionSpec DisruptionSpec
	if filePath == "" {
		fmt.Println(pathError)
	}
	fullPath, _ := filepath.Abs(filePath)
	disruption, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("disruption.Get err   #%v ", err)
		//panic(err)
		return
	}
	err = yaml.Unmarshal(disruption, &disruptionSpec)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Println("This Disruption...")
	e := Explanations{seperator: "======================================================================================================================================="}
	e.explainMetaSpec(disruptionSpec.Spec)
	e.explainMultiDisruption(disruptionSpec.Spec)
	e.explainNodeFailure(disruptionSpec.Spec)
	e.explainNetworkFailure(disruptionSpec.Spec)
	e.explainCPUPressure(disruptionSpec.Spec)
	e.explainDiskPressure(disruptionSpec.Spec)
	e.explainDNS(disruptionSpec.Spec)
}

func init() {
	explainCmd.Flags().String("path", "", "The path to the disruption file to be explained.")
}
