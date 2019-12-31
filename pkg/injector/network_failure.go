package injector

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/container"
	"github.com/DataDog/chaos-fi-controller/pkg/datadog"
	"github.com/DataDog/chaos-fi-controller/pkg/logger"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/coreos/go-iptables/iptables"
	"github.com/miekg/dns"
)

const iptablesTable = "filter"
const iptablesOutputChain = "OUTPUT"
const iptablesChaosChainPrefix = "CHAOS-"

// NetworkFailureInjector describes a network failure
type NetworkFailureInjector struct {
	ContainerInjector
	Spec *v1beta1.NetworkFailureInjectionSpec
}

// Inject injects the given network failure into the given container
func (i NetworkFailureInjector) Inject() {
	var ruleParts []string

	// Enter container network namespace
	c := container.New(i.ContainerID)
	c.EnterNetworkNamespace()
	defer container.ExitNetworkNamespace()

	// Resolve host if needed
	logger.Instance().Infow("resolving given host", "host", i.Spec.Failure.Host)
	ips, err := resolveHost(i.Spec.Failure.Host)
	if err != nil {
		datadog.EventInjectFailure(i.ContainerID, i.UID)
		datadog.MetricInjected(i.ContainerID, i.UID, false)
		logger.Instance().Fatalw("unable to resolve host", "error", err, "host", i.Spec.Failure.Host)
	}

	// Inject
	dedicatedChain := i.getDedicatedChainName()
	logger.Instance().Infow("starting the injection", "chain", dedicatedChain)
	ipt, err := iptables.New()
	if err != nil {
		datadog.EventInjectFailure(i.ContainerID, i.UID)
		datadog.MetricInjected(i.ContainerID, i.UID, false)
		logger.Instance().Fatalw("unable to load iptables driver", "error", err)
	}

	// Create a new dedicated chain
	if !chainExists(ipt, dedicatedChain) {
		logger.Instance().Infow("creating the dedicated iptables chain", "chain", dedicatedChain)
		err = ipt.NewChain(iptablesTable, dedicatedChain)
		if err != nil {
			datadog.EventInjectFailure(i.ContainerID, i.UID)
			datadog.MetricInjected(i.ContainerID, i.UID, false)
			logger.Instance().Fatalw("error while creating the dedicated chain",
				"error", err,
				"chain", dedicatedChain,
			)
		}
	}

	// Add the dedicated chain jump rule
	ruleParts = []string{"-j", dedicatedChain}
	rule := fmt.Sprintf("iptables -t %s %s", iptablesTable, strings.Join(ruleParts, " "))
	err = ipt.AppendUnique(iptablesTable, iptablesOutputChain, ruleParts...)
	if err != nil {
		datadog.EventInjectFailure(i.ContainerID, i.UID)
		datadog.MetricInjected(i.ContainerID, i.UID, false)
		logger.Instance().Fatalw("error while injecting the jump rule", "error", err, "rule", rule)
	}

	// Append all rules to the dedicated chain
	for _, ip := range ips {
		// Ignore IPv6 addresses
		if len(ip.IP) != 4 {
			continue
		}

		ruleParts = i.GenerateRuleParts(ip.String())
		rule := fmt.Sprintf("iptables -t %s -A %s %s", iptablesTable, dedicatedChain, strings.Join(ruleParts, " "))
		logger.Instance().Infow(
			"injecting drop rule",
			"table", iptablesTable,
			"chain", dedicatedChain,
			"ip", ip.String(),
			"port", i.Spec.Failure.Port,
			"protocol", i.Spec.Failure.Protocol,
			"probability", i.Spec.Failure.Probability,
			"rule", rule,
		)
		err = ipt.AppendUnique(iptablesTable, dedicatedChain, ruleParts...)
		if err != nil {
			datadog.EventInjectFailure(i.ContainerID, i.UID)
			datadog.MetricInjected(i.ContainerID, i.UID, false)
			datadog.MetricRulesInjected(i.ContainerID, i.UID, false)
			logger.Instance().Fatalw("error while injecting the drop rule", "error", err, "rule", rule)
			return
		}

		datadog.GetInstance().Event(&statsd.Event{
			Title: "network failure injected",
			Text:  fmt.Sprintf("the following rule has been injected: %s", rule),
			Tags: []string{
				"containerID:" + i.ContainerID,
				"UID:" + i.UID,
			},
		})
		datadog.MetricRulesInjected(i.ContainerID, i.UID, true)
	}

	datadog.MetricInjected(i.ContainerID, i.UID, true)
}

// Clean removes all the injected failures in the given container
func (i NetworkFailureInjector) Clean() {
	// Enter container network namespace
	c := container.New(i.ContainerID)
	c.EnterNetworkNamespace()
	defer container.ExitNetworkNamespace()

	dedicatedChain := i.getDedicatedChainName()
	logger.Instance().Infow("starting the cleaning", "chain", dedicatedChain)
	ipt, err := iptables.New()
	if err != nil {
		datadog.EventCleanFailure(i.ContainerID, i.UID)
		datadog.MetricCleaned(i.ContainerID, i.UID, false)
		logger.Instance().Fatalw("unable to load iptables driver", "error", err)
	}

	// Clear, delete the dedicated chain and its jump rule if it exists
	if chainExists(ipt, dedicatedChain) {
		// Delete the jump rule if it exists
		ruleParts := []string{"-j", dedicatedChain}
		exists, err := ipt.Exists(iptablesTable, iptablesOutputChain, ruleParts...)
		if err != nil {
			datadog.EventCleanFailure(i.ContainerID, i.UID)
			datadog.MetricCleaned(i.ContainerID, i.UID, false)
			logger.Instance().Fatalw("unable to check if the dedicated chain jump rule exists", "chain", dedicatedChain, "error", err)
		}
		if exists {
			logger.Instance().Infow("deleting the dedicated chain jump rule", "chain", dedicatedChain)
			err = ipt.Delete(iptablesTable, iptablesOutputChain, ruleParts...)
			if err != nil {
				datadog.EventCleanFailure(i.ContainerID, i.UID)
				datadog.MetricCleaned(i.ContainerID, i.UID, false)
				logger.Instance().Fatalw("failed to clean dedicated chain jump rule", "chain", dedicatedChain, "error", err)
			}
		}

		// Clear and delete the dedicated chain
		logger.Instance().Infow("clearing the dedicated chain", "chain", dedicatedChain)
		err = ipt.ClearChain(iptablesTable, dedicatedChain)
		if err != nil {
			datadog.EventCleanFailure(i.ContainerID, i.UID)
			datadog.MetricCleaned(i.ContainerID, i.UID, false)
			logger.Instance().Fatalw("failed to clean dedicated chain", "chain", dedicatedChain, "error", err)
		}
		logger.Instance().Infow("deleting the dedicated chain", "chain", dedicatedChain)
		err = ipt.DeleteChain(iptablesTable, dedicatedChain)
		if err != nil {
			datadog.EventCleanFailure(i.ContainerID, i.UID)
			datadog.MetricCleaned(i.ContainerID, i.UID, false)
			logger.Instance().Fatalw("failed to delete dedicated chain", "chain", dedicatedChain, "error", err)
		}
	}

	datadog.GetInstance().Event(&statsd.Event{
		Title: "network failure cleaned",
		Text:  "the rules have been cleaned",
		Tags: []string{
			"containerID:" + i.ContainerID,
			"UID:" + i.UID,
		},
	})
	datadog.MetricCleaned(i.ContainerID, i.UID, true)
}

// getDedicatedChainName crafts the chaos dedicated chain name
// from the failure resource UID
// it basically keeps the first and last part of the UID because
// of the iptables 29-chars chain name limit
func (i NetworkFailureInjector) getDedicatedChainName() string {
	parts := strings.Split(i.UID, "-")
	shortUID := parts[0] + parts[len(parts)-1]

	return iptablesChaosChainPrefix + shortUID
}

// resolveHost tries to resolve the given host
// it tries to resolve it as a CIDR, as a single IP, or as a hostname
// it returns a list of IP or an error if it fails to resolve the hostname
func resolveHost(host string) ([]*net.IPNet, error) {
	var ips []*net.IPNet
	//No Host Specified
	if len(host) == 0 {
		logger.Instance().Infow("host not specified, using 0.0.0.0/0")
		host = "0.0.0.0/0"
	}
	_, ipnet, err := net.ParseCIDR(host)
	if err != nil {
		logger.Instance().Infow("not a valid CIDR, trying to resolve it as a single IP", "host", host)
		ip := net.ParseIP(host)
		if ip == nil {
			logger.Instance().Infow("not a valid single IP, trying to resolve it as a hostname", "host", host)
			dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
			if err != nil {
				logger.Instance().Fatalw("unable to read resolv.conf file", "error", err)
			}
			dnsClient := dns.Client{}
			dnsMessage := dns.Msg{}
			dnsMessage.SetQuestion(host+".", dns.TypeA)
			response, _, err := dnsClient.Exchange(&dnsMessage, dnsConfig.Servers[0]+":53")
			if err != nil {
				logger.Instance().Fatalw("error while talking to the resolver", "error", err)
			}
			for _, answer := range response.Answer {
				rec := answer.(*dns.A)
				ips = append(ips, &net.IPNet{
					IP:   rec.A,
					Mask: net.CIDRMask(32, 32),
				})
			}
		} else {
			// ensure the parsed IP is an IPv4
			// the net.ParseIP function returns an IPv4 with an IPv6 length
			// the code blow ensures the parsed IP prefix is the default (empty) prefix
			// of an IPv6 address:
			// var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
			var a, b [12]byte
			copy(a[:], ip[0:12])
			b[10] = 0xff
			b[11] = 0xff
			if a != b {
				logger.Instance().Fatalw("given IP seems to be an IPv6 address, aborting")
			}

			// use a /32 mask for a single IP
			ips = append(ips, &net.IPNet{
				IP:   ip[12:16],
				Mask: net.CIDRMask(32, 32),
			})
		}
	} else {
		// use the given CIDR network
		ips = append(ips, ipnet)
	}

	if len(ips) == 0 {
		return nil, errors.New("failed to resolve the given host")
	}

	return ips, nil
}

// chainExists returns true if the given chain exists, false otherwise
func chainExists(ipt *iptables.IPTables, chain string) bool {
	chains, err := ipt.ListChains(iptablesTable)
	if err != nil {
		logger.Instance().Fatalw("unable to list iptables chain", "error", err)
	}
	for _, v := range chains {
		if v == chain {
			return true
		}
	}
	return false
}

//GenerateRuleParts generates the iptables rules to apply
func (i NetworkFailureInjector) GenerateRuleParts(ip string) []string {
	var ruleParts = []string{
		"-p", i.Spec.Failure.Protocol,
		"-d", ip,
		"--dport",
		strconv.Itoa(i.Spec.Failure.Port)}

	//Add modules (if any) here
	ruleParts = append(ruleParts, "-m")
	var numModules = 0

	//Probability Module
	if i.Spec.Failure.Probability != 0 && i.Spec.Failure.Probability != 100 {
		//Probability expected in decimal format
		var prob = float64(i.Spec.Failure.Probability) / 100.0
		ruleParts = append(ruleParts,
			"statistic", "--mode", "random", "--probability", fmt.Sprintf("%.2f", prob),
		)
		numModules++
	}

	//If no modules were defined, remove the tailing -m
	if numModules == 0 {
		ruleParts = ruleParts[:len(ruleParts)-1]
	}

	ruleParts = append(ruleParts,
		"-j",
		"DROP")
	return ruleParts
}
