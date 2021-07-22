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

	"github.com/DataDog/chaos-controller/network"
)

const (
	httpPort = ":80"
	tlsPort  = ":443"
)

type HTTPDisruptionInjector struct {
	config HTTPDisruptionInjectorConfig
}

type HTTPDisruptionInjectorConfig struct {
	Config
	Iptables   network.Iptables
	FileWriter FileWriter

	ProxyExit chan struct{}
}

func NewHTTPDisruptionInjector(config HTTPDisruptionInjectorConfig) (Injector, error) {
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
		config.ProxyExit = make(chan struct{})
	}

	return HTTPDisruptionInjector{
		config: config,
	}, err
}

func (i HTTPDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding http disruption", "spec", i.config) // TODO: Change this to spec

	// TODO: Add IP table updates here

	// TODO: Start the proxy servers
	i.startProxyServer()

	return nil
}

func (i HTTPDisruptionInjector) Clean() error {
	i.config.Log.Info("Stopping HTTP disruption proxy")

	i.config.ProxyExit <- struct{}{}

	// TODO: Clean up IP Tables

	return nil
}

func (i HTTPDisruptionInjector) startProxyServer() error {
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
		tlsServer.Shutdown(context.TODO())
		httpServer.Shutdown(context.TODO())
	}()

	return nil
}

func (i HTTPDisruptionInjector) proxyHandler(rw http.ResponseWriter, r *http.Request) {
	i.config.Log.Infow("REQUEST", "METHOD", r.Method, "PATH", r.URL.Path, "HEADER", r.Header)
	if r.Method == http.MethodPost {
		i.config.Log.Debug("DROPPING REQUEST ", r.Host+r.URL.Path)
		return
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

	rw.Write(body)
}
