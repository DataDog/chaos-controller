// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc

package injector

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	httpChain = "CHAOS-HTTP"
)

type HTTPDisruptionInjector struct {
	config HTTPDisruptionInjectorConfig
	spec   v1beta1.HTTPDisruptionSpec
}

type HTTPDisruptionInjectorConfig struct {
	Config
	Iptables   network.Iptables
	FileWriter FileWriter

	ProxyExit chan struct{}
}

const InjectorHTTPCgroupClassID = "0x00110011"

func NewHTTPDisruptionInjector(spec v1beta1.HTTPDisruptionSpec, config HTTPDisruptionInjectorConfig) (Injector, error) {
	var err error
	if config.Iptables == nil {
		config.Iptables, err = network.NewIptables(config.Log, config.DryRun)
	}

	if config.FileWriter == nil {
		config.FileWriter = standardFileWriter{
			dryRun: config.DryRun,
		}
	}

	if config.ProxyExit == nil {
		config.ProxyExit = make(chan struct{}, 2)
	}

	return HTTPDisruptionInjector{
		config: config,
		spec:   spec,
	}, err
}

func (i HTTPDisruptionInjector) CreateIptablesRules(podIP string, port int) error {
	if err := i.config.Iptables.AddRuleWithIP(httpChain, "tcp", strconv.Itoa(port), "DNAT", podIP); err != nil {
		return fmt.Errorf("unable to create new iptables HTTP(S) rule: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelPod {
		if err := i.config.Iptables.AddCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", strconv.Itoa(port), httpChain); err != nil {
			return fmt.Errorf("unable to create new HTTP(S) iptables rule: %w", err)
		}
	}

	return nil
}

func (i HTTPDisruptionInjector) DeleteIptablesRules(port int) error {
	if err := i.config.Iptables.DeleteCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", strconv.Itoa(port), httpChain); err != nil {
		return fmt.Errorf("unable to remove injected HTTP(S) iptables rule: %w", err)
	}
	if err := i.config.Iptables.DeleteCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", strconv.Itoa(port), httpChain); err != nil {
		return fmt.Errorf("unable to remove injected HTTP(S) iptables rule: %w", err)
	}

	return nil
}

func (i HTTPDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding http disruption", "spec", i.spec)

	// get the chaos pod node IP from the environment variable
	podIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	i.startProxyServers()

	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	if err := i.config.Iptables.CreateChain(httpChain); err != nil {
		return fmt.Errorf("unable to create new iptables chain: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelPod {
		// write classid to container net_cls cgroup - for iptable filtering
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", InjectorHTTPCgroupClassID); err != nil {
			return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
		}
	}

	// Add iptables rules for each specified port, spec defaults to 80/443 from the cli
	for _, httpPort := range i.spec.HttpPorts {
		i.CreateIptablesRules(podIP, httpPort)
	}
	for _, httpsPort := range i.spec.HttpsPorts {
		i.CreateIptablesRules(podIP, httpsPort)
	}

	if i.config.Level == chaostypes.DisruptionLevelNode {
		return fmt.Errorf("unable to create HTTP disruption at the node level")
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

func (i HTTPDisruptionInjector) Clean() error {
	i.config.Log.Info("Stopping HTTP disruption proxy")

	if i.config.ProxyExit == nil {
		return fmt.Errorf("proxy exit channel is nil")
	}

	i.config.ProxyExit <- struct{}{}

	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container namespace: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelPod {
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0x0"); err != nil {
			return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
		}

		// Delete iptables rules for each specified port, spec defaults to 80/443 from the cli
		for _, httpPort := range i.spec.HttpPorts {
			i.DeleteIptablesRules(httpPort)
		}
		for _, httpsPort := range i.spec.HttpsPorts {
			i.DeleteIptablesRules(httpsPort)
		}
	}

	if i.config.Level == chaostypes.DisruptionLevelNode {
		return fmt.Errorf("unable to create HTTP disruption at the node level")
	}

	if err := i.config.Iptables.ClearAndDeleteChain(httpChain); err != nil {
		return fmt.Errorf("unable to remove injected iptables chain: %w", err)
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// startProxyServers starts both the HTTP and HTTPS proxy servers on their own goroutines.
// One more goroutine is started which blocks until it receives a signal on the `ProxyExit`
// channel at which point it calls the `Shutdown` method for both the HTTP and HTTPS servers.
func (i HTTPDisruptionInjector) startProxyServers() error {
	// Recover from registering multiple proxyHandlers to /
	defer func() {
		if r := recover(); r != nil {
			i.config.Log.Info("Extra proxyHandler registration")
		}
	}()
	http.HandleFunc("/", i.proxyHandler)

	var tlsServers []*http.Server
	var httpServers []*http.Server

	// Create servers and bind to every needed port
	for _, port := range i.spec.HttpPorts {
		httpServers = append(httpServers, &http.Server{Addr: strconv.Itoa(port)})

		go func() {
			i.config.Log.Info("Starting HTTP server on ", port)
			httpServers[len(httpServers) - 1].ListenAndServe()
		}()
	}

	for _, port := range i.spec.HttpPorts {
		tlsServers = append(tlsServers, &http.Server{Addr: strconv.Itoa(port)})

		go func() {
			i.config.Log.Info("Starting TLS server on ", port)
			tlsServers[len(tlsServers) - 1].ListenAndServeTLS("", "")
		}()
	}

	go func() {
		// Block until there is a signal to shutdown the proxy
		<-i.config.ProxyExit
		for _, server := range tlsServers {
			if err := server.Shutdown(context.TODO()); err != nil {
				i.config.Log.Error(err)
			}
		}
		for _, server := range tlsServers {
			if err := server.Shutdown(context.TODO()); err != nil {
				i.config.Log.Error(err)
			}
		}
	}()

	return nil
}

// proxyHandler handles the HTTP requests for both the HTTP and HTTPS servers.
// If a request matches a provided filter from the `HTTPDisruptionSpec` then it
// simply returns. Otherwise it continues to handle the requests.
func (i HTTPDisruptionInjector) proxyHandler(rw http.ResponseWriter, r *http.Request) {
	i.config.Log.Infow("request", "method", r.Method, "request host", r.Host, "url host", r.URL.Host, "path", r.URL.Path, "header", r.Header)

	// match requests by the spec
	for _, domain := range i.spec.Domains {
		if domain.Domain == r.URL.Host {
			i.config.Log.Debugw("dropping", "domain", domain.Domain)
			return
		}
	}

	var scheme string
	if r.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}

	// Check for override of Host field
	var reqHost string
	if r.Host != "" {
		reqHost = r.Host
	} else {
		reqHost = r.URL.Host
	}

	proxyURI := fmt.Sprintf("%s://%s%s", scheme, reqHost, r.URL.Path)

	req, err := http.NewRequest(r.Method, proxyURI, r.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	// TODO: We probably want to create a pool of clients somewhere
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	for key, values := range resp.Header {
		for _, val := range values {
			rw.Header().Add(key, val)
		}
	}

	rw.WriteHeader(resp.StatusCode)
	rw.Write(body)
}
