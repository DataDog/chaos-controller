// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
)

var VERSION string
var STATUS string
var logger *zap.SugaredLogger

type SpecificRequest struct {
	CustomDisruption       v1beta1.Disruption `json:"disruption"`
	PreInstalledDisruption string             `json:"preinstalled"`
}

type Response struct {
	Disruption       string    `json:"disruption"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	Results          string    `json:"results"`
	ResultsExplained string    `json:"resultsExplained"`
}

func version(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(VERSION); err != nil {
		logger.Errorw("Failed to Encode Version: %w", err)
	}
}

func status(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(STATUS); err != nil {
		logger.Errorw("Failed to Encode STATUS: %w", err)
	}
}

func handleRequests() {
	http.HandleFunc("/version", version)
	http.HandleFunc("/status", status)

	STATUS = "ready for requests"
}

func main() {
	//TODO
	//1. Wait for a request to run a test
	//4. Find out what relevant metrics would be (if only testing CPU, only CPU metrics matter)
	//5. Get relevant metrics from datadog for the past 3 minutes
	//6. Deploy the version to test
	//7. Once the testing version is deployed and read, depending on the request, create individual disruptions
	//8. For each disruption, let it bake for 3 minutes
	//9. After baking, grab the last 3 minutes of data to compare to the stable 3 minutes of data
	//10. If data looks rights, pass the test and move on to the next disruption and repeat starting from 8 until all
	//    disruptions are completed for the given request
	//11. For each disruption removal, wait 2 minutes to make sure the state of world goes back to stable values measured
	//    in the beginning
	//12. Once the entire request is finished, do 1 of 2 things:
	//    a. If queue is empty, return the chaos controller in the staging cluster back to latest:stable
	//    b. If queue is not empty, take the next request
	STATUS = "initializing"
	logger = &zap.SugaredLogger{}
	handleRequests()
}
