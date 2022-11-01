// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

func init() {
}

func main() {
	//TODO
	//1. Wait for a request to run a test
	//2. When a request is received, place that request in a global queue in case several people are attempting to test at once
	//   a. Have CI continuously hit this end point and if its currently in the queue, returns its place in queue
	//3. In another go thread, pop requests to test from the queue
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
}
