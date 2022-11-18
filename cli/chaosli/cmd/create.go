// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a disruption.",
	Long:  `creates a disruption given input from the user.`,
	Run: func(cmd *cobra.Command, args []string) {
		spec, _ := createSpec()
		err := spec.Validate()
		if err != nil {
			fmt.Printf("There were some problems when validating your disruption: %v", err)
		}
		jsonRep, err := json.MarshalIndent(spec, "", " ")
		if err != nil {
			fmt.Printf("json err: %v", err)
		}

		jsonRep = []byte(fmt.Sprintf(`{"apiVersion": "chaos.datadoghq.com/v1beta1", "kind": "Disruption", "metadata": %s, "spec": %s}`, getMetadata(), jsonRep))

		y, err := yaml.JSONToYAML(jsonRep)
		if err != nil {
			fmt.Printf("yaml err: %v", err)
		}

		path, _ := cmd.Flags().GetString("path")
		err = ioutil.WriteFile(path, y, 0644) // #nosec
		if err != nil {
			fmt.Printf("writeFile err: %v", err)
		}

		fmt.Printf("We wrote your disruption to %s, thanks!", path)
	},
}

const intro = `Hello! This tool will walk you through creating a disruption. Please reply to the prompts, and use Ctrl+C to end.
The generated disruption will have "dryRun:true" set for safety, which means you can safely apply it without injecting any failure.`

func init() {
	createCmd.Flags().String("path", "disruption.yaml", "The file to write the new disruption to.")

	if err := createCmd.MarkFlagRequired("path"); err != nil {
		return
	}
}

func createSpec() (v1beta1.DisruptionSpec, error) {
	fmt.Println(intro)

	spec := v1beta1.DisruptionSpec{}

	err := promptForKind(&spec)

	if err != nil {
		return spec, err
	}

	spec.Level = getLevel()
	spec.Selector = getSelectors()
	spec.StaticTargeting = getStaticTargeting()
	spec.Count = getCount()

	isPulsingCompatible := true

	for _, disruptionKind := range spec.GetKindNames() {
		if disruptionKind == types.DisruptionKindContainerFailure || disruptionKind == types.DisruptionKindNodeFailure {
			isPulsingCompatible = false
			break
		}
	}

	if isPulsingCompatible {
		spec.Pulse = getPulse()
	}

	if spec.Level == types.DisruptionLevelPod {
		spec.Containers = getContainers()
	}

	if spec.ContainerFailure == nil && spec.CPUPressure == nil && spec.DiskPressure == nil && spec.NodeFailure == nil && spec.GRPC == nil && spec.DiskFailure == nil && spec.Level == types.DisruptionLevelPod && len(spec.Containers) == 0 {
		spec.OnInit = getOnInit()
	}

	spec.DryRun = getDryRun()

	return spec, nil
}

func promptForKind(spec *v1beta1.DisruptionSpec) error {
	initial := "Let's begin by choosing the type of disruption to apply! Which disruption kind would you like to add?"
	followUp := "Would you like to add another disruption kind? It's not necessary, most disruptions involve only one kind. Select .. to finish adding kinds."
	kinds := []string{"dns", "network", "cpu", "disk", "node failure", "container failure", "disk failure"}
	helpText := `The DNS disruption allows for overriding the A or CNAME records returned by DNS queries.
The Network disruption allows for injecting a variety of different network issues into your target.
The CPU and Disk disruptions apply cpu pressure or IO throttling to your target, respectively.
Tne Node Failure disruption can either shutdown or restart the targeted node, or the node hosting the targeted pod.

Select one for more information on it.`

	query := initial

	for {
		response, err := selectInput(query, kinds, helpText)
		if err != nil {
			return err
		}

		switch response {
		case "..":
			return nil
		case "dns":
			spec.DNS = getDNS()

			if spec.DNS == nil {
				continue
			}

			err := spec.DNS.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your DNS disruption's spec: %v\n\n", err)

				spec.DNS = nil

				continue
			}
		case "network":
			spec.Network = getNetwork()

			if spec.Network == nil {
				continue
			}

			err := spec.Network.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your network disruption's spec: %v\n\n", err)

				spec.Network = nil

				continue
			}
		case "cpu":
			spec.CPUPressure = getCPUPressure()

			if spec.CPUPressure == nil {
				continue
			}

			err := spec.CPUPressure.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your CPU pressure disruption's spec: %v\n\n", err)

				spec.CPUPressure = nil

				continue
			}
		case "disk pressure":
			spec.DiskPressure = getDiskPressure()

			if spec.DiskPressure == nil {
				continue
			}

			err := spec.DiskPressure.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your disk throttling disruption's spec: %v\n\n", err)

				spec.DiskPressure = nil

				continue
			}
		case "disk failure":
			spec.DiskFailure = getDiskFailure()

			if spec.DiskFailure == nil {
				continue
			}

			err := spec.DiskFailure.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your disk failure disruption's spec: %v\n\n", err)

				spec.DiskFailure = nil

				continue
			}
		case "node failure":
			spec.NodeFailure = getNodeFailure()

			if spec.NodeFailure == nil {
				continue
			}

			err := spec.NodeFailure.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your node failure disruption's spec: %v\n\n", err)

				spec.NodeFailure = nil

				continue
			}
		case "container failure":
			spec.ContainerFailure = getContainerFailure()

			if spec.ContainerFailure == nil {
				continue
			}

			err := spec.ContainerFailure.Validate()
			if err != nil {
				fmt.Printf("There were some problems with your container failure disruption's spec: %v\n\n", err)

				spec.ContainerFailure = nil

				continue
			}
		}

		i := indexOfString(kinds, response)
		kinds = append(kinds[:i], kinds[i+1:]...)

		if query == initial {
			kinds = append(kinds, "..")
		}

		query = followUp
	}
}

func confirmKind(kind string, helpText string) bool {
	return confirmOption(fmt.Sprintf("Would you like to include the disruption %s?", kind), helpText)
}

func confirmOption(query string, helpText string) bool {
	var result bool

	prompt := &survey.Confirm{
		Message: query,
		Help:    helpText,
	}

	err := survey.AskOne(prompt, &result)

	if err == terminal.InterruptErr {
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("confirmOption failed: %v", err)
	}

	return result
}

func getInput(query string, helpText string, opts ...survey.AskOpt) string {
	var result string

	prompt := &survey.Input{
		Message: query,
		Help:    helpText,
	}
	err := survey.AskOne(prompt, &result, opts...)

	if err == terminal.InterruptErr {
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("getInput failed: %v", err)
	}

	return result
}

func selectInput(query string, inputs []string, helpText string) (string, error) {
	var result string

	prompt := &survey.Select{
		Message: query,
		Options: inputs,
		Help:    helpText,
	}

	err := survey.AskOne(prompt, &result)

	if err == terminal.InterruptErr {
		os.Exit(1)
	}

	return result, err
}

func getSliceInput(query string, helpText string, opts ...survey.AskOpt) []string {
	var results string

	prompt := &survey.Multiline{
		Message: query,
		Help:    helpText,
	}

	err := survey.AskOne(prompt, &results, opts...)

	if err == terminal.InterruptErr {
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("getSliceInput failed: %v\n", err)
	}

	return strings.Split(results, "\n")
}

func getMetadata() []byte {
	fmt.Println("Last step, you just have to name your disruption, and specify what k8s namespace it should live in.")

	validator := func(val interface{}) error {
		if str, ok := val.(string); ok {
			errs := validation.IsDNS1123Subdomain(str)
			if errs != nil {
				return fmt.Errorf("the name and namespace need to be valid DNS-1123 subdomains: %s", errs)
			}
		} else {
			return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
		}

		return nil
	}

	name := getInput("Please name your disruption.",
		"This will be the name used when you want to run `kubectl describe disruption`",
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	)
	namespace := getInput(
		"What namespace should your disruption be created in?",
		"If you are targeting pods, you _must_ create the disruption in the same namespace as the targeted pods.",
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	)

	return []byte(fmt.Sprintf(`{"name": %s, "namespace": %s}`, name, namespace))
}

func getDNS() v1beta1.DNSDisruptionSpec {
	if !confirmKind("DNS Disruption", "Overrides DNS resolution for specified hostnames with a MitM DNS attack. All other DNS requests will use the target's normal DNS resolver.") {
		return nil
	}

	getHostRecordPair := func() v1beta1.HostRecordPair {
		hrPair := v1beta1.HostRecordPair{}

		hrPair.Hostname = getInput("Specify a hostname to target",
			"When your target makes a DNS request for this hostname; the disruption will make sure the value you specify is returned, rather than the real record.",
			survey.WithValidator(survey.Required),
		)
		hrPair.Record.Type, _ = selectInput("the type of DNS record to inject",
			[]string{"A", "CNAME"},
			"We only support these two types of DNS requests for now. An A record request gets back an IP for a hostname, while a CNAME request maps an alias domain name to the canonical name.")
		helpText := "We're specifying an A record, so the value should be an IP address. You can specify multiple IP addresses, if desired. Simply delimit them with commas, no whitespace! The disruption will round-robin between the options."

		if hrPair.Record.Type == "CNAME" {
			helpText = "We're specifying a CNAME record, so the value should be a hostname to redirect to."
		}

		hrPair.Record.Value = getInput("What value would you like to inject into this DNS record?", helpText, survey.WithValidator(survey.Required))

		return hrPair
	}

	fmt.Println("Let's specify a DNS record to inject!")

	spec := v1beta1.DNSDisruptionSpec{}
	spec = append(spec, getHostRecordPair())

	for confirmOption("Would you like to override another DNS record?", "") {
		spec = append(spec, getHostRecordPair())
	}

	return spec
}

func getDiskPressure() *v1beta1.DiskPressureSpec {
	if !confirmKind("Disk Pressure", "Simulates disk pressure by applying IO throttling to the target") {
		return nil
	}

	spec := &v1beta1.DiskPressureSpec{}

	spec.Path = getInput(
		"Specify a path to apply IO throttling to, e.g., /mnt/data",
		"Specify a specific mount point to target the disk mounted there",
		survey.WithValidator(survey.Required),
	)

	if confirmOption("Would you like to apply read throttling?", "This applies read-based IO throttling (check the docs)") {
		readBPS, _ := strconv.Atoi(getInput("Specify the target amount of throttling, in bytes per second.", "check the docs", survey.WithValidator(integerValidator)))
		spec.Throttling.ReadBytesPerSec = &readBPS
	}

	if confirmOption("Would you like to apply write throttling?", "This applies write-based IO throttling (check the docs)") {
		writeBPS, _ := strconv.Atoi(getInput("Specify the target amount of throttling, in bytes per second.", "check the docs", survey.WithValidator(integerValidator)))
		spec.Throttling.WriteBytesPerSec = &writeBPS
	}

	return spec
}

func getCPUPressure() *v1beta1.CPUPressureSpec {
	if confirmKind("CPU Pressure", "Applies CPU pressure to the target") {
		return &v1beta1.CPUPressureSpec{}
	}

	return nil
}

func getNodeFailure() *v1beta1.NodeFailureSpec {
	if !confirmKind("Node Failure", "This will either shutdown or restart the targeted node (or node hosting the targeted pod)") {
		return nil
	}

	spec := &v1beta1.NodeFailureSpec{}
	spec.Shutdown = confirmOption("Would you like to shutdown the node permanently?",
		"Choosing yes will terminate the VM completely. If you don't enable this, we will just restart the target node.")

	return spec
}

func getContainerFailure() *v1beta1.ContainerFailureSpec {
	if !confirmKind("Container Failure", "This will terminate the targeted pod's container(s) gracefully (SIGTERM) or non-gracefully (SIGKILL)") {
		return nil
	}

	spec := &v1beta1.ContainerFailureSpec{}
	spec.Forced = confirmOption("Would you like to terminate the pod's containers non-gracefully?",
		"Choosing yes will terminate the pod's containers non-gracefully. If you don't enable this, we will terminate the target containers gracefully.")

	return spec
}

func getHosts() []v1beta1.NetworkDisruptionHostSpec {
	if !confirmOption("Would you like to specify any hosts?",
		"If you want to target _all_ traffic, or only want to target k8s services, don't specify any hosts.") {
		return nil
	}

	var hosts []v1beta1.NetworkDisruptionHostSpec

	getHost := func() v1beta1.NetworkDisruptionHostSpec {
		host := v1beta1.NetworkDisruptionHostSpec{}

		fmt.Println(`Each "host" is a 3-tuple of host, port, and protocol. `)

		host.Host = getInput("Add a host to target (or leave blank)",
			"This will affect the network traffic between these hosts and your target. These can be hostnames, IPs, or CIDR blocks. These _cannot_ be k8s services.",
		)
		host.Port, _ = strconv.Atoi(getInput("What port would you like to target? (or leave blank for all)", "If specified, we will only affect traffic using this port", survey.WithValidator(integerValidator)))

		if confirmOption("Would you like to specifically target only tcp or udp traffic?", "The default is to target all traffic.") {
			host.Protocol, _ = selectInput("Please choose then (or ctrl+c to go back)", []string{"tcp", "udp"}, "This will cause only the traffic using this protocol to be affected.")
		}

		host.Flow, _ = selectInput(
			"Choose a flow direction",
			[]string{v1beta1.FlowEgress, v1beta1.FlowIngress},
			fmt.Sprintf("%s will affect traffic leaving the target. %s will not really affect traffic entering the target, but actually will affect replies to the inbound traffic.",
				v1beta1.FlowEgress, v1beta1.FlowIngress),
		)

		return host
	}

	hosts = append(hosts, getHost())

	for confirmOption("Would you like to add another host/port/protocol tuple?", "") {
		hosts = append(hosts, getHost())
	}

	return hosts
}

func getServices() []v1beta1.NetworkDisruptionServiceSpec {
	if !confirmOption("Would you like to specify any k8s services?",
		"If you want to target _all_ traffic, or only want to target hosts, don't specify any services.") {
		return nil
	}

	var services []v1beta1.NetworkDisruptionServiceSpec

	getService := func() v1beta1.NetworkDisruptionServiceSpec {
		service := v1beta1.NetworkDisruptionServiceSpec{}

		service.Name = getInput("What is the name of this service?", "", survey.WithValidator(survey.Required))
		service.Namespace = getInput("What namespace is this service in?", "", survey.WithValidator(survey.Required))

		return service
	}

	services = append(services, getService())

	for confirmOption("Would you like to add another k8s service?", "") {
		services = append(services, getService())
	}

	return services
}

func getNetwork() *v1beta1.NetworkDisruptionSpec {
	if !confirmKind("Network Disruption", "Injects a variety of possible network issues") {
		return nil
	}

	spec := &v1beta1.NetworkDisruptionSpec{}

	fmt.Println(`The network disruption will inject issues between your targets, and the hosts + kubernetes services they communicate with.
 We need to handle targeting "regular" hosts from kubernetes services differently for technical reasons that are explained in the docs.`)

	spec.Hosts = getHosts()
	spec.Services = getServices()

	if confirmOption("Would you like to drop packets?", "Packets will be dropped before leaving the target") {
		spec.Drop, _ = strconv.Atoi(getInput("What % of packets should we affect?", "1-100", survey.WithValidator(survey.Required), survey.WithValidator(percentageValidator)))
	}

	if confirmOption("Would you like to duplicate packets?", "Packets will be duplicated immediately before leaving the target") {
		spec.Duplicate, _ = strconv.Atoi(getInput("What % of packets should we affect?", "1-100", survey.WithValidator(survey.Required), survey.WithValidator(percentageValidator)))
	}

	if confirmOption("Would you like to corrupt packets?", "Packets will be corrupted before leaving the target") {
		spec.Corrupt, _ = strconv.Atoi(getInput("What % of packets should we affect?", "1-100", survey.WithValidator(survey.Required), survey.WithValidator(percentageValidator)))
	}

	if confirmOption("Would you like to delay packets?", "Packets will be delayed before leaving the target") {
		delay, _ := strconv.ParseUint(getInput("How much to delay (in ms)?", "This will be the median amount of delay to apply", survey.WithValidator(survey.Required)), 10, 0)
		spec.Delay = uint(delay)

		delayJitter, _ := strconv.ParseUint(getInput("What jitter on that delay (in ms)?", "This will be normally distributed around the delay you specified earlier. This will cause packets to re-order!"), 10, 0)
		spec.DelayJitter = uint(delayJitter)
	}

	if confirmOption("Would you like to limit bandwidth?", "bandwidthlimit") {
		spec.BandwidthLimit, _ = strconv.Atoi(getInput("What bandwidth limit should we set (in bytes per second)?", ">0", survey.WithValidator(survey.Required), survey.WithValidator(integerValidator)))
	}

	if confirmOption("Would you like to add additional allowedHosts?", "These will be added to an allowlist and will be excluded from the disruption.") {
		spec.AllowedHosts = getHosts()
	}

	spec.DisableDefaultAllowedHosts = confirmOption("Would you like to add disable the default allowedHosts?", "DANGER: You can leave a node entirely unable to connect to the k8s api or cloud provider, which may have consequences.")

	return spec
}

func getContainers() []string {
	if !confirmOption("Would you like to target a specific container[s]?", "The default is to target all containers in the target pod.") {
		return nil
	}

	containers := getSliceInput("Please enter a comma-delimited list of container name[s] to target.", "Please specify their names, not their IDs!")

	return containers
}

func getCount() *intstr.IntOrString {
	validator := func(val interface{}) error {
		if str, ok := val.(string); ok {
			count := intstr.Parse(str)
			return v1beta1.ValidateCount(&count)
		}

		return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
	}
	result := getInput(
		"How many targets would you like to disrupt? This can be an integer, or a percentage.",
		"Please specify an integer >0 or a percentage from 1% - 100%. If specifying a percentage, you must suffix with the % character, or we will think its an integer!",
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	)

	wrappedResult := intstr.FromString(result)

	return &wrappedResult
}

func getOnInit() bool {
	onInitExplanations := "An OnInit disruption is a disruption which will be launched on the initialization of your targeted pod(s), enabling the disruption to be working directly at the start of the disrupted pod(s).\nTo make it work, you need to add \"chaos.datadoghq.com/disrupt-on-init: \"true\"\" to the labels of your targeted pod(s) and redeploy them."

	fmt.Println(onInitExplanations)

	return confirmOption("Do you want to enable on initialization disruptions?", "OnInit is disabled by default")
}

func getPulse() *v1beta1.DisruptionPulse {
	validator := func(val interface{}) error {
		if str, ok := val.(string); ok {
			_, err := time.ParseDuration(str)
			if err != nil {
				return err
			}

			duration := v1beta1.DisruptionDuration(str)
			if duration.Duration() < types.PulsingDisruptionMinimumDuration {
				return fmt.Errorf("duration must be greater than %s", types.PulsingDisruptionMinimumDuration)
			}

			return nil
		}

		return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
	}

	if !confirmOption("A pulsing disruption is a disruption which will be injected and active for a certain amount of time, then cleaned and dormant for a certain amount of time, and so on until it is removed. Would you like your disruptions to be pulsing?", "The default is non pulsing disruptions.") {
		return nil
	}

	activeDuration := v1beta1.DisruptionDuration(getInput(
		"What would be the duration of the disruption in an active state during the pulse? This can be a golang's time.Duration.",
		fmt.Sprintf("Please specify a golang's time.Duration's >%s, e.g., \"45s\", \"15m30s\", \"4h30m\".", types.PulsingDisruptionMinimumDuration),
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	))

	dormantDuration := v1beta1.DisruptionDuration(getInput(
		"What would be the duration of the disruption in a dormant state during the pulse? This can be a golang's time.Duration.",
		fmt.Sprintf("Please specify a golang's time.Duration's >%s, e.g., \"45s\", \"15m30s\", \"4h30m\".", types.PulsingDisruptionMinimumDuration),
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	))

	return &v1beta1.DisruptionPulse{
		ActiveDuration:  activeDuration,
		DormantDuration: dormantDuration,
	}
}

func getSelectors() labels.Set {
	validator := func(val interface{}) error {
		if str, ok := val.(string); ok {
			for _, s := range strings.Split(str, "\n") {
				if !strings.Contains(s, "=") {
					return fmt.Errorf("please specify label selectors in the form key=value")
				}
			}
		} else {
			return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
		}

		return nil
	}
	selectors := getSliceInput(
		"Add a label selector[s] for targeting.",
		`Please specify this in the form of "key=value", e.g., "app=hello-node". One label selector per new-line. If you specify multiple, we will only target the union of all selectors.
For example, if you set both "app=hello-node" and "pod-name=ubuntu-uuid", then no matter how many pods with the label "app=hello-node" there were, the disruption would only target pods who also had the label "pod-name=ubuntu-uuid".`,
		survey.WithValidator(survey.Required),
		survey.WithValidator(validator),
	)

	var selectorLabels labels.Set

	for _, s := range selectors {
		sAsSet, err := labels.ConvertSelectorToLabelsMap(s)

		if err != nil {
			fmt.Printf("invalid selector string: %v", err)
			return nil
		}

		selectorLabels = labels.Merge(selectorLabels, sAsSet)
	}

	return selectorLabels
}

func getLevel() types.DisruptionLevel {
	level, err := selectInput(
		"Select the Disruption Level.",
		[]string{types.DisruptionLevelNode, types.DisruptionLevelPod},
		"This will affect targeting with the label selectors, as well as injecting (depending on the disruption kind).",
	)

	if err != nil {
		level = types.DisruptionLevelPod
	}

	return types.DisruptionLevel(level)
}

func getStaticTargeting() bool {
	staticTargetingExplanations := "StaticTargeting means the target selection will only happen once at disruption creation, and will never be run again. New targets will not be targeted. StaticTargeting is temporarily defaulting to true, and will eventually default to false"

	fmt.Println(staticTargetingExplanations)

	a := confirmOption("Would you like to enable StaticTargeting? Blocks new pods from being targeted after the initial injection.", staticTargetingExplanations)

	return a
}

func getDryRun() bool {
	return confirmOption(`Would you like us to set the dryRun option? If toggled to "true", then applying the disruption will be "safe".`,
		`If selected, then when you apply this disruption, targets will be selected, and chaos pods created, but we won't inject any failures. Simply delete the option from the yaml or change it to "dryRun: false" in order to apply the disruption normally.`)
}

func indexOfString(slice []string, indexed string) int {
	for i, item := range slice {
		if item == indexed {
			return i
		}
	}

	return -1
}

func percentageValidator(val interface{}) error {
	if str, ok := val.(string); ok {
		if str == "" {
			return nil
		}

		input := intstr.Parse(str)
		value, _, _ := v1beta1.GetIntOrPercentValueSafely(&input)

		if value < 0 || value > 100 {
			return fmt.Errorf("input must be a valid percentage value, between 0-100: got %s", str)
		}
	} else {
		return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
	}

	return nil
}

func integerValidator(val interface{}) error {
	if str, ok := val.(string); ok {
		if str == "" {
			return nil
		}

		_, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("this value must be an integer: got %v", err)
		}
	} else {
		return fmt.Errorf("expected a string response, rather than type %v", reflect.TypeOf(val).Name())
	}

	return nil
}
