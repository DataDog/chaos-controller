// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestTargetWatchStopsReplacedAndFinalStreams(t *testing.T) {
	var watchCount atomic.Int32
	firstStopped := make(chan struct{})
	secondStarted := make(chan struct{})
	secondStopped := make(chan struct{})
	forceClose := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("watch") != "true" {
			fmt.Fprintln(w, `{"kind":"Pod","apiVersion":"v1","metadata":{"name":"target","namespace":"app","resourceVersion":"1"}}`)
			return
		}

		index := watchCount.Add(1)
		w.WriteHeader(http.StatusOK)
		if index == 1 {
			fmt.Fprintln(w, `{"type":"BOOKMARK","object":{"kind":"Pod","apiVersion":"v1","metadata":{"name":"target","namespace":"app","resourceVersion":"2"}}}`)
		}
		w.(http.Flusher).Flush()
		if index == 2 {
			close(secondStarted)
		}
		select {
		case <-r.Context().Done():
		case <-forceClose:
		}
		if index == 1 {
			close(firstStopped)
		} else if index == 2 {
			close(secondStopped)
		}
	}))

	oldClientset, oldArgs, oldSignals, oldLog := clientset, disruptionArgs, signals, log
	var err error
	clientset, err = kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		close(forceClose)
		server.Close()
		t.Fatal(err)
	}
	disruptionArgs = chaosapi.DisruptionArgs{DisruptionNamespace: "app", TargetName: "target"}
	signals = make(chan os.Signal, 1)
	log = zap.NewNop().Sugar()
	done := make(chan error, 1)

	t.Cleanup(func() {
		defer func() {
			clientset, disruptionArgs, signals, log = oldClientset, oldArgs, oldSignals, oldLog
		}()
		defer server.Close()
		defer close(forceClose)
		select {
		case signals <- syscall.SIGTERM:
		default:
		}
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("watch loop returned an error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("watch loop did not stop after SIGTERM")
		}
		// Join the loop's signal-forwarding goroutine before restoring globals.
		select {
		case <-signals:
		case <-time.After(time.Second):
			t.Error("watch loop did not forward the exit signal")
		}
		select {
		case <-secondStopped:
		case <-time.After(2 * time.Second):
			t.Error("final watch HTTP stream remained open after the loop returned")
		}
	})

	go func() {
		done <- watchTargetAndReinject(time.Now().Add(30*time.Second), "network-disruption", 0, 0)
	}()

	select {
	case <-secondStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("bookmark did not start the replacement watch")
	}
	select {
	case <-firstStopped:
	case <-time.After(2 * time.Second):
		t.Error("superseded watch HTTP stream remained open after its bookmark")
	}
}
