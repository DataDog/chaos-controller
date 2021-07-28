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

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	httpPort = ":80"
	tlsPort  = ":443"

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

func (i HTTPDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding http disruption", "spec", i.spec)

	// get the chaos pod node IP from the environment variable
	podIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	i.startProxyServers()

	// TODO: Add IP table updates here
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	if err := i.config.Iptables.CreateChain(httpChain); err != nil {
		return fmt.Errorf("unable to create new iptables chain: %w", err)
	}

	if err := i.config.Iptables.AddRuleWithIP(httpChain, "tcp", "8080", "DNAT", podIP); err != nil {
		return fmt.Errorf("unable to create new iptables HTTP rule: %w", err)
	}

	if err := i.config.Iptables.AddRuleWithIP(httpChain, "tcp", "443", "DNAT", podIP); err != nil {
		return fmt.Errorf("unable to create new iptables HTTPS rule: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelPod {
		// write classid to container net_cls cgroup - for iptable filtering
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", InjectorHTTPCgroupClassID); err != nil {
			return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
		}

		if err := i.config.Iptables.AddCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", "8080", httpChain); err != nil {
			return fmt.Errorf("unable to create new HTTP iptables rule: %w", err)
		}

		if err := i.config.Iptables.AddCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", "443", httpChain); err != nil {
			return fmt.Errorf("unable to create new HTTPS iptables rule: %w", err)
		}
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

		if err := i.config.Iptables.DeleteCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", "8080", httpChain); err != nil {
			return fmt.Errorf("unable to remove injected HTTP iptables rule: %w", err)
		}
		if err := i.config.Iptables.DeleteCgroupFilterRule("OUTPUT", InjectorHTTPCgroupClassID, "tcp", "443", httpChain); err != nil {
			return fmt.Errorf("unable to remove injected HTTPS iptables rule: %w", err)
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

// startProxyServers starts both the HTT and HTTPS proxy servers on their own goroutines.
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

	tlsServer := &http.Server{Addr: tlsPort}
	httpServer := &http.Server{Addr: httpPort}

	go func() {
		i.config.Log.Info("Starting TLS server on ", tlsPort)
		tlsServer.ListenAndServeTLS("", "")
	}()

	go func() {
		i.config.Log.Info("Starting HTTP server on ", httpPort)
		httpServer.ListenAndServe()
	}()

	go func() {
		// Block until there is a signal to shutdown the proxy
		<-i.config.ProxyExit
		if err := tlsServer.Shutdown(context.TODO()); err != nil {
			i.config.Log.Error(err)
		}

		if err := httpServer.Shutdown(context.TODO()); err != nil {
			i.config.Log.Error(err)
		}
	}()

	return nil
}

// proxyHandler handles the HTTP requests for both the HTTP and HTTPS servers.
// If a request matches a provided filter from the `HTTPDisruptionSpec` then it
// simply returns. Otherwise it continues to handle the requests.
func (i HTTPDisruptionInjector) proxyHandler(rw http.ResponseWriter, r *http.Request) {
	i.config.Log.Infow("request", "method", r.Method, "path", r.URL.Path, "header", r.Header)

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

	proxyURI := fmt.Sprintf("%s://%s%s", scheme, r.URL.Host, r.URL.Path)

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
