// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// This function returns a scaled value from an IntOrString type. If the IntOrString
// is a percentage string value it's treated as a percentage and scaled appropriately
// in accordance to the total, if it's an int value it's treated as a a simple value and
// if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func getScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if intOrPercent == nil {
		return 0, k8serrors.NewBadRequest("nil value for IntOrString")
	}

	value, isPercent, err := v1beta1.GetIntOrPercentValueSafely(intOrPercent)

	if err != nil {
		return 0, fmt.Errorf("invalid value for IntOrString: %v", err)
	}

	if isPercent {
		if roundUp {
			value = int(math.Ceil(float64(value) * (float64(total)) / 100))
		} else {
			value = int(math.Floor(float64(value) * (float64(total)) / 100))
		}
	}

	return value, nil
}

func calculateRemainingDuration(instance v1beta1.Disruption) time.Duration {
	return calculateDeadline(
		instance.Spec.Duration.Duration(),
		instance.ObjectMeta.CreationTimestamp.Time,
	)
}

// returned value can be negative if deadline is in the past
func calculateDeadline(duration time.Duration, creationTime time.Time) time.Duration {
	// first we must calculate the timout from when the disruption was created, not from now
	timeout := creationTime.Add(duration)
	now := time.Now() // rather not take the risk that the time changes by a second during this function

	// return the number of seconds between now and the deadline
	return timeout.Sub(now)
}

// assert label selector matches valid grammar, avoids CORE-414
func validateLabelSelector(selector labels.Selector) error {
	labelGrammar := "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]"
	rgx := regexp.MustCompile(labelGrammar)

	if !rgx.MatchString(selector.String()) {
		return fmt.Errorf("given label selector is invalid, it does not match valid selector grammar: %s %s", selector.String(), labelGrammar)
	}

	return nil
}

// transformCloudSpecToHostsSpec from a cloud spec disruption, get all ip ranges of services provided and transform them into a list of hosts spec
func transformCloudSpecToHostsSpec(cloudManager *cloudservice.CloudServicesProvidersManager, cloudSpec *v1beta1.NetworkDisruptionCloudSpec) ([]v1beta1.NetworkDisruptionHostSpec, error) {
	hosts := []v1beta1.NetworkDisruptionHostSpec{}
	clouds := cloudSpec.TransformToCloudMap()

	for cloudName, serviceList := range clouds {
		serviceListNames := []string{}

		for _, service := range serviceList {
			serviceListNames = append(serviceListNames, service.ServiceName)
		}

		ipRangesPerService, err := cloudManager.GetServicesIPRanges(types.CloudProviderName(cloudName), serviceListNames)
		if err != nil {
			return nil, err
		}

		for _, serviceSpec := range serviceList {
			for _, ipRange := range ipRangesPerService[serviceSpec.ServiceName] {
				hosts = append(hosts, v1beta1.NetworkDisruptionHostSpec{
					Host:      ipRange,
					Protocol:  serviceSpec.Protocol,
					Flow:      serviceSpec.Flow,
					ConnState: serviceSpec.ConnState,
				})
			}
		}
	}

	return hosts, nil
}

// isModifiedError tells us if this error is of the form:
// "Operation cannot be fulfilled on disruptions.chaos.datadoghq.com "chaos-network-drop": the object has been modified; please apply your changes to the latest version and try again"
// Sadly this doesn't seem to be one of the errors checkable with a function from "k8s.io/apimachinery/pkg/api/errors"
// So we parse the error message directly
func isModifiedError(err error) bool {
	return strings.Contains(err.Error(), "please apply your changes to the latest version and try again")
}

func TimeUntilCreatePods(triggers *v1beta1.DisruptionTriggers, creationTimestamp time.Time) time.Duration {
	if triggers == nil {
		return time.Duration(0)
	}

	if triggers.CreatePods == nil {
		return time.Duration(0)
	}

	var noPodsBefore time.Time

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.CreatePods.NotBefore.IsZero() {
		noPodsBefore = triggers.CreatePods.NotBefore.Time
	}

	if triggers.CreatePods.Offset.Duration() > 0 {
		noPodsBefore = creationTimestamp.Add(triggers.CreatePods.Offset.Duration())
	}

	return time.Until(noPodsBefore)
}

// TimeToInject (for now) returns the unix epoch offset in milliseconds at which we want to inject
func TimeToInject(triggers *v1beta1.DisruptionTriggers, creationTimestamp time.Time) int64 {
	if triggers == nil {
		return 0
	}

	if triggers.Inject == nil {
		return 0
	}

	var notInjectedBefore int64

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.Inject.NotBefore.IsZero() {
		notInjectedBefore = triggers.Inject.NotBefore.UnixMilli()
	}

	if triggers.Inject.Offset.Duration() > 0 {
		// We measure the offset from the latter of two timestamps: creationTimestamp of the disruption, and spec.trigger.createPods.notBefore
		offsetTime := creationTimestamp
		if triggers.CreatePods != nil && !triggers.CreatePods.NotBefore.IsZero() {
			offsetTime = triggers.CreatePods.NotBefore.Time
		}

		notInjectedBefore = offsetTime.Add(triggers.Inject.Offset.Duration()).UnixMilli()
	}

	return notInjectedBefore
}
