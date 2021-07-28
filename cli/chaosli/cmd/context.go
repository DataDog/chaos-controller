// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
)

const MAXTARGETSHOW = 10

func contextTargetsSize(disruption v1beta1.Disruption) ([]string, error) {
	spec := disruption.Spec
	labels := spec.Selector.String()
	level := string(spec.Level)
	podNamespaces := disruption.ObjectMeta.Namespace

	fmt.Println("Let's look at your targets...")

	s := spinner.New(spinner.CharSets[38], 100*time.Millisecond)
	s.Start()

	size, err := getTargetSize(disruption)

	if err != nil {
		return nil, err
	}

	// Remove header NAME from consideration
	size--
	// If the size is 0, first check if changing the level will do anything, otherwise
	// mention to the user that the labels they are using won't target anything

	if size <= 0 {
		errorString := fmt.Sprintf("\nThe label selectors chosen (%s) result in 0 targets, meaning this disruption would do nothing given the namespace/cluster/label combination.", labels)

		if level == types.DisruptionLevelPod {
			disruption.Spec.Level = types.DisruptionLevelNode
		} else {
			disruption.Spec.Level = types.DisruptionLevelPod
		}

		size, err = getTargetSize(disruption)
		// Remove header NAME from consideration
		size--

		if err != nil {
			return nil, err
		}

		if size > 0 {
			errorString = fmt.Sprintf("\nWe noticed that your target size is 0 for level %s given your label selectors. We checked to see if the %s level would give you results and we found %d %ss. Is this the level you wanted to use?", level, disruption.Spec.Level, size, disruption.Spec.Level)
		}

		return nil, fmt.Errorf(errorString)
	}

	cmd := fmt.Sprintf("kubectl get %s -n %s -l %s | awk '{print $1}'", level, podNamespaces, labels)
	targets, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		return nil, fmt.Errorf("could not grab list of targets names correlating to target selector: %v", err)
	}

	s.Stop()

	targetsShow := []string{}
	targetsAll := []string{}
	targetsSplit := strings.Split(string(targets), "\n")

	for i := 0; i < len(targetsSplit); i++ {
		if len(targetsShow) < MAXTARGETSHOW {
			targetsShow = append(targetsShow, targetsSplit[i])
		}

		if targetsSplit[i] == "NAME" || targetsSplit[i] == "" {
			continue
		}

		targetsAll = append(targetsAll, targetsSplit[i])
	}

	fmt.Printf("\n🎯 There are %d %ss that will be targeted\n", size, level)

	if size > MAXTARGETSHOW {
		fmt.Println("📝 Here are a small set of those targets:")
	}

	for _, target := range targetsShow {
		fmt.Println(target)
	}

	if size > MAXTARGETSHOW {
		fmt.Println("...")
	}

	PrintSeparator()

	return targetsAll, nil
}

func getTargetSize(disruption v1beta1.Disruption) (int, error) {
	level := disruption.Spec.Level
	podNamespace := disruption.Namespace
	labels := disruption.Spec.Selector.String()
	cmd := fmt.Sprintf("kubectl get %s -n %s -l %s | wc -l", level, podNamespace, labels)
	sizeString, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		return -1, fmt.Errorf("could not count the number of targets correlating to target selector: %v", err)
	}

	size, err := strconv.Atoi(strings.Trim(string(sizeString), "\n"))

	if err != nil {
		return -1, fmt.Errorf("could not convert string to integer in context target size: %v", err)
	}

	return size, nil
}

func grabDataForTargets(targets []string, disruption v1beta1.Disruption) ([]v1.Pod, []v1.Node, error) {
	namespace := disruption.ObjectMeta.Namespace
	level := disruption.Spec.Level
	pods := []v1.Pod{}
	nodes := []v1.Node{}

	s := spinner.New(spinner.CharSets[38], 100*time.Millisecond)
	s.Start()

	for _, targetName := range targets {
		pod := v1.Pod{}
		node := v1.Node{}

		cmd := fmt.Sprintf("kubectl get %s -o json -n %s %s", level, namespace, targetName)
		targetData, err := exec.Command("bash", "-c", cmd).Output()

		if err != nil {
			return nil, nil, fmt.Errorf("could not grab target data: %v", err)
		}

		if level == types.DisruptionLevelPod {
			if err := json.Unmarshal(targetData, &pod); err != nil {
				return nil, nil, fmt.Errorf("json encoding failed: %v", err)
			}

			pods = append(pods, pod)
		} else {
			if err := json.Unmarshal(targetData, &node); err != nil {
				return nil, nil, fmt.Errorf("json encoding failed: %v", err)
			}
			nodes = append(nodes, node)
		}
	}

	s.Stop()

	return pods, nodes, nil
}

func printContainerStatus(targetInfo []v1.Pod) {
	percentCollect := make(map[string]float64)

	fmt.Println("\nLets take" +
		" a look at the status of your Targeted Pod Containers...")

	totalContainers := 0

	for i, pod := range targetInfo {
		info := "\t🥸  Pod Name: " + pod.Name + "\n"

		for j := 0; j < len(pod.Status.ContainerStatuses); j++ {
			totalContainers++

			containerStatus := pod.Status.ContainerStatuses[j]
			info += "\t\t🤓 Container Name: " + containerStatus.Name + "\n" +
				"\t\t⭕️ Ready Status: " + strconv.FormatBool(containerStatus.Ready) + "\n"

			state := containerStatus.State

			switch {
			case state.Running != nil:
				info += "\t\t📝 State: Running\n\n"
				percentCollect["Running"] += 1.0
			case state.Waiting != nil:
				info += "\t\t📝 State: Waiting\n\n"
				percentCollect["Waiting"] += 1.0
			case state.Terminated != nil:
				info += "\t\t📝 State: Terminated\n\n"
				percentCollect["Terminated"] += 1.0
			}

			if containerStatus.Ready {
				percentCollect["Ready"] += 1.0
			}
		}

		if i < MAXTARGETSHOW {
			fmt.Printf(info + "\n\n")
		}
	}

	PrintSeparator()

	percentInfo := "Lets look at the overall status of your targeted pod's containers...\n"

	for key, value := range percentCollect {
		roundedValue := (value / float64(totalContainers)) * 100.00

		if key == "Ready" {
			percentInfo += "\tState:                 " + "Ready" + "\n" +
				"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
			percentInfo += "\tState:                 " + "Not Ready" + "\n" +
				"\tPercent:               " + fmt.Sprint(100.00-math.Round(roundedValue*100)/100) + "%\n\n"

			continue
		}

		percentInfo += "\tState:                 " + key + "\n" +
			"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
	}

	fmt.Println(percentInfo)
}

func printPodStatus(targetsInfo []v1.Pod) {
	percentCollectPhase := make(map[string]float64)
	percentCollectCondition := make(map[string]float64)

	fmt.Println("\nLets take a look at the status of your Targeted Pods...")

	for i := 0; i < len(targetsInfo); i++ {
		pod := targetsInfo[i]
		info := "\t🥸  Pod Name: " + pod.Name + "\n" +
			"\t👵🏾 Pod Host IP: " + pod.Status.HostIP + "\n" +
			"\t👧🏾 Pod IP: " + pod.Status.PodIP + "\n" +
			"\t🌒 Pod Phase: " + string(pod.Status.Phase) + "\n" +
			"\t📜 Pod Conditions:\n"

		for _, status := range pod.Status.Conditions {
			info = info +
				"\t\t⭕️ Type: " + string(status.Type) + "\n" +
				"\t\tℹ️  Status: " + string(status.Status) + "\n\n"
			percentCollectCondition[string(status.Type)+"/"+string(status.Status)] += 1.0 / float64(len(targetsInfo))
		}

		percentCollectPhase[string(pod.Status.Phase)] += 1.0 / float64(len(targetsInfo))

		if i < MAXTARGETSHOW {
			fmt.Printf(info + "\n\n")
		}
	}
	PrintSeparator()

	percentInfo := "Lets look at the overall status...\n"

	for key, value := range percentCollectPhase {
		roundedValue := value * 100.00
		percentInfo += "\tPhase:                 " + key + "\n" +
			"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
	}

	for key, value := range percentCollectCondition {
		roundedValue := value * 100.00
		percentInfo += "\tCondition Type/Status: " + key + "\n" +
			"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
	}

	fmt.Println(percentInfo)

	PrintSeparator()
}

func printNodeStatus(targetsInfo []v1.Node) {
	percentCollectCondition := make(map[string]float64)

	fmt.Println("\nLets take a look at the status of your Targeted Nodes...")

	for i := 0; i < len(targetsInfo); i++ {
		node := targetsInfo[i]
		info := "\t🥸  Node Name: " + node.Name + "\n" +
			"\t📲 Node Addresses: \n"

		for _, address := range node.Status.Addresses {
			info += "\t\t⭕️ Type: " + string(address.Type) + "\n" +
				"\t\tℹ️  Address: " + address.Address + "\n"
		}

		info += "\t📜 Node Conditions:\n"

		for _, status := range node.Status.Conditions {
			info = info +
				"\t\t⭕️ Type: " + string(status.Type) + "\n" +
				"\t\tℹ️  Status: " + string(status.Status) + "\n" +
				"\t\t🤨 Reason: " + status.Reason + "\n\n"
			percentCollectCondition[string(status.Type)+"/"+string(status.Status)] += 1.0 / float64(len(targetsInfo))
		}

		if i < MAXTARGETSHOW {
			fmt.Printf(info + "\n\n")
		}
	}

	PrintSeparator()

	percentInfo := "Lets look at the overall status...\n"

	for key, value := range percentCollectCondition {
		roundedValue := value * 100.00
		percentInfo += "\tCondition Type/Status: " + key + "\n" +
			"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
	}

	fmt.Println(percentInfo)
	PrintSeparator()
}

func checkKubectl() error {
	cmd := exec.Command("kubectl", "get", "pods", "-n", "chaos-engineering")
	_, err := cmd.Output()

	return err
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "contextualizes disruption config",
	Long:  `makes use of kubectl to give a better idea of how big the scope of the disruption will be.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		contextualize(path)
	},
}

func contextualize(path string) {
	disruption := ReadUnmarshalValidate(path)

	err := checkKubectl()
	if err != nil {
		log.Fatalf("Could not find/use kubectl command, make sure it is in your PATH variable and that all authorizations for the command are set (login to your authorization provider (e.g. Your VPN).")
	}

	targets, err := contextTargetsSize(disruption)
	if err != nil {
		log.Fatalf("Could not grab context regarding size and names of targets: %v", err)
	}

	podsData, nodesData, err := grabDataForTargets(targets, disruption)
	if err != nil {
		log.Fatalf("Attempted to grab data for targets and failed: %v", err)
	}

	if disruption.Spec.Level == types.DisruptionLevelPod {
		printPodStatus(podsData)
		printContainerStatus(podsData)
	} else {
		printNodeStatus(nodesData)
	}
}

func init() {
	contextCmd.Flags().String("path", "", "The path to the disruption file to be contextualized.")
	contextCmd.Println("This flag requires that you are connected to a cluster and kubectl works as intended on that cluster. All the results of this flag will be based on your current kubectx.")
}
