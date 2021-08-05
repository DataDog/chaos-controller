// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var maxtargetshow int
var kubeconfig string
var verbose bool

// besides calculating size, this function also grabs the list of targets corresponding to the
// selector in the disruption. The targets can either be a list of pods or a list of nodes
// therefore only one of the return types is actually populated while the other is empty.
func contextTargetsSize(disruption v1beta1.Disruption) ([]v1.Pod, []v1.Node, error) {
	var pods v1.PodList

	var nodes v1.NodeList

	var allPods []v1.Pod

	var allNodes []v1.Node
	
	spec := disruption.Spec
	labels := spec.Selector.String()
	level := string(spec.Level)
	size := 0

	fmt.Println("Let's look at your targets...")

	if level == types.DisruptionLevelPod {
		pods = getPods(disruption)
		size = len(pods.Items)
	} else {
		nodes = getNodes(disruption)
		size = len(nodes.Items)
	}

	// If the size is 0, first check if changing the level will do anything, otherwise
	// mention to the user that the labels they are using won't target anything

	if size <= 0 {
		errorString := fmt.Sprintf("\nThe label selectors chosen (%s) result in 0 targets, meaning this disruption would do nothing given the namespace/cluster/label combination.", labels)

		if level == types.DisruptionLevelPod {
			disruption.Spec.Level = types.DisruptionLevelNode
		} else {
			disruption.Spec.Level = types.DisruptionLevelPod
		}

		size = getTargetSize(disruption)

		if size > 0 {
			errorString = fmt.Sprintf("\nWe noticed that your target size is 0 for level %s given your label selectors. We checked to see if the %s level would give you results and we found %d %ss. Is this the level you wanted to use?", level, disruption.Spec.Level, size, disruption.Spec.Level)
		}

		return nil, nil, fmt.Errorf(errorString)
	}

	if level == types.DisruptionLevelPod {
		allPods = showPods(pods)
	} else {
		allNodes = showNodes(nodes)
	}

	PrintSeparator()

	return allPods, allNodes, nil
}

func showPods(pods v1.PodList) []v1.Pod {
	targetsShow := []string{}

	targetsAll := make([]v1.Pod, len(pods.Items))

	for i, pod := range pods.Items {
		if len(targetsShow) < maxtargetshow {
			targetsShow = append(targetsShow, pod.Name)
		}

		targetsAll[i] = pod
	}

	fmt.Printf("\n🎯 There are %d pods that will be targeted\n", len(pods.Items))

	if len(pods.Items) > maxtargetshow {
		fmt.Println("📝 Here are a small set of those targets:")
	}

	for _, target := range targetsShow {
		fmt.Println(target)
	}

	if len(pods.Items) > maxtargetshow {
		fmt.Println("...")
	}

	return targetsAll
}

func showNodes(nodes v1.NodeList) []v1.Node {
	targetsShow := []string{}
	targetsAll := []v1.Node{}

	for _, node := range nodes.Items {
		if len(targetsShow) < maxtargetshow {
			targetsShow = append(targetsShow, node.Name)
		}

		targetsAll = append(targetsAll, node)
	}

	fmt.Printf("\n🎯 There are %d nodes that will be targeted\n", len(nodes.Items))

	if len(nodes.Items) > maxtargetshow {
		fmt.Println("📝 Here are a small set of those targets:")
	}

	for _, target := range targetsShow {
		fmt.Println(target)
	}

	if len(nodes.Items) > maxtargetshow {
		fmt.Println("...")
	}

	return targetsAll
}

func getTargetSize(disruption v1beta1.Disruption) int {
	level := disruption.Spec.Level
	size := 0

	if level == types.DisruptionLevelPod {
		size = len(getPods(disruption).Items)
	} else {
		size = len(getNodes(disruption).Items)
	}

	return size
}

func getPods(disruption v1beta1.Disruption) v1.PodList {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(disruption.Spec.Selector).String(),
	}
	pods, err := clientset.CoreV1().Pods(disruption.ObjectMeta.Namespace).List(context.TODO(), options)

	if err != nil {
		panic(err.Error())
	}

	return *pods
}

func getNodes(disruption v1beta1.Disruption) v1.NodeList {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	options := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(disruption.Spec.Selector).String(),
	}
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), options)

	if err != nil {
		panic(err.Error())
	}

	return *nodes
}

func printContainerStatus(targetInfo []v1.Pod) {
	percentCollect := make(map[string]float64)

	if verbose {
		fmt.Printf("\nLets take a look at the status of your Targeted Pod Containers...\n\n")
	}

	totalContainers := 0

	for i, pod := range targetInfo {
		info := "\t🥸  Pod Name: " + pod.Name + "\n"

		// adding 1 to these states so we can run percentages. Since we have the number
		// of instances with that specific state (the += 1.0) and we know the total
		// number of instances, we can easily get a percentage.
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

		if i < maxtargetshow && verbose {
			fmt.Print(info)
		}
	}

	if verbose {
		PrintSeparator()
	}

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

	if verbose {
		fmt.Printf("\nLets take a look at the status of your Targeted Pods...\n\n")
	}

	// adding 1 to these states so we can run percentages. Since we have the number
	// of instances with that specific state (the += 1.0) and we know the total
	// number of instances, we can easily get a percentage.
	for i, pod := range targetsInfo {
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

		if i < maxtargetshow && verbose {
			fmt.Print(info)
		}
	}

	if verbose {
		PrintSeparator()
	}

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

	if verbose {
		fmt.Printf("\nLets take a look at the status of your Targeted Nodes...\n\n")
	}

	for i, node := range targetsInfo {
		info := "\t🥸  Node Name: " + node.Name + "\n" +
			"\t📲 Node Addresses: \n"

		for _, address := range node.Status.Addresses {
			info += "\t\t⭕️ Type: " + string(address.Type) + "\n" +
				"\t\tℹ️  Address: " + address.Address + "\n"
		}

		info += "\t📜 Node Conditions:\n"

		// adding 1 to these states so we can run percentages. Since we have the number
		// of instances with that specific state (the += 1.0) and we know the total
		// number of instances, we can easily get a percentage.
		for _, status := range node.Status.Conditions {
			info = info +
				"\t\t⭕️ Type: " + string(status.Type) + "\n" +
				"\t\tℹ️  Status: " + string(status.Status) + "\n" +
				"\t\t🤨 Reason: " + status.Reason + "\n\n"
			percentCollectCondition[string(status.Type)+"/"+string(status.Status)] += 1.0 / float64(len(targetsInfo))
		}

		if i < maxtargetshow && verbose {
			fmt.Print(info)
		}
	}

	if verbose {
		PrintSeparator()
	}

	percentInfo := "Lets look at the overall status...\n"

	for key, value := range percentCollectCondition {
		roundedValue := value * 100.00
		percentInfo += "\tCondition Type/Status: " + key + "\n" +
			"\tPercent:               " + fmt.Sprint(math.Round(roundedValue*100)/100) + "%\n\n"
	}

	fmt.Println(percentInfo)
	PrintSeparator()
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "contextualizes disruption config",
	Long:  `makes use of kubectl to give a better idea of how big the scope of the disruption will be.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		kubeconfig, _ = cmd.Flags().GetString("kubeconfig")
		verbose, _ = cmd.Flags().GetBool("verbose")
		maxtargetshow, _ = cmd.Flags().GetInt("maxtargetshow")
		contextualize(path)
	},
}

func contextualize(path string) {
	disruption := ReadUnmarshalValidate(path)

	if len(kubeconfig) == 0 {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	pods, nodes, err := contextTargetsSize(disruption)

	if err != nil {
		log.Fatalf("Could not grab context regarding size and names of targets: %v", err)
	}

	// validate should catch if the disruption level is invalid, safe to assume default else is Node
	if disruption.Spec.Level == types.DisruptionLevelPod {
		printPodStatus(pods)
		printContainerStatus(pods)
	} else {
		printNodeStatus(nodes)
	}
}

func init() {
	contextCmd.Flags().String("path", "", "The path to the disruption file to be contextualized.")
	contextCmd.Flags().String("kubeconfig", "", "The path to your kube configuration directory (.../.kube/config). defaults to ~/.kube/config.")
	contextCmd.Flags().Bool("verbose", false, "If set, will describe a small set of 5 (default) of your targets. Otherwise it only describes percentages of the group of targets in total.")
	contextCmd.Flags().Int("maxtargetshow", 5, "Only really applies when verbose is set to true; This value determines how many targets will be described in the output.")
	err := contextCmd.MarkFlagRequired("path")

	if err != nil {
		return
	}

	contextCmd.Println("This command requires that you are connected to a kubernetes cluster. All the results of this command will be based on your current kubectx.")
	PrintSeparator()
}
