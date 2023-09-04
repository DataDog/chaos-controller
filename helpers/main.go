// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package helpers

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cloudservice"
	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type TerminationStatus uint8

const (
	TSNotTerminated TerminationStatus = iota
	TSTemporarilyTerminated
	TSDefinitivelyTerminated
)

// GetScaledValueFromIntOrPercent This function returns a scaled value from an IntOrString type. If the IntOrString
// is a percentage string value it's treated as a percentage and scaled appropriately
// in accordance to the total, if it's an int value it's treated as a a simple value and
// if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func GetScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if intOrPercent == nil {
		return 0, k8serrors.NewBadRequest("nil value for IntOrString")
	}

	value, isPercent, err := chaosv1beta1.GetIntOrPercentValueSafely(intOrPercent)
	if err != nil {
		return 0, fmt.Errorf("invalid value for IntOrString: %w", err)
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

// DisruptionTerminationStatus determines if the disruption injection is temporarily or definitively terminated
// disruption can enter a temporary injection removal state when all targets have disappeared (due to rollout or manual deletion)
// disruption will enter a definitive ended state when remaining duration is over or has been deleted
func DisruptionTerminationStatus(instance chaosv1beta1.Disruption, chaosPods []corev1.Pod) TerminationStatus {
	// a not yet created disruption is neither temporary nor definitively ended
	if instance.CreationTimestamp.IsZero() {
		return TSNotTerminated
	}

	// a definitive state (expired duration or deletion) imply a definitively deleted injection
	// and should be returned prior to a temporarily terminated state
	if calculateRemainingDuration(instance) <= 0 || !instance.DeletionTimestamp.IsZero() {
		return TSDefinitivelyTerminated
	}

	if len(chaosPods) == 0 {
		// we were never injected, we are hence not terminated if we reach here
		if instance.Status.InjectionStatus.NeverInjected() {
			return TSNotTerminated
		}

		// we were injected before hence temporarily not terminated
		return TSTemporarilyTerminated
	}

	// if all pods exited successfully, we can consider the disruption is ended already
	// it can be caused by either an appromixative date sync (in a distributed infra it's hard)
	// or by deletion of targets leading to deletion of injectors
	// injection terminated with an error are considered NOT terminated
	for _, chaosPod := range chaosPods {
		for _, containerStatuses := range chaosPod.Status.ContainerStatuses {
			if containerStatuses.State.Terminated == nil || containerStatuses.State.Terminated.ExitCode != 0 {
				return TSNotTerminated
			}
		}
	}

	// this MIGHT be a temporary status, that could become definitive once disruption is expired or deleted
	return TSTemporarilyTerminated
}

func calculateRemainingDuration(instance chaosv1beta1.Disruption) time.Duration {
	return calculateDeadline(
		instance.Spec.Duration.Duration(),
		TimeToInject(instance.Spec.Triggers, instance.ObjectMeta.CreationTimestamp.Time),
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

// ValidateLabelSelector assert label selector matches valid grammar, avoids CORE-414
func ValidateLabelSelector(selector labels.Selector) error {
	labelGrammar := "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]"
	rgx := regexp.MustCompile(labelGrammar)

	if !rgx.MatchString(selector.String()) {
		return fmt.Errorf("given label selector is invalid, it does not match valid selector grammar: %s %s", selector.String(), labelGrammar)
	}

	return nil
}

// IsModifiedError tells us if this error is of the form:
// "Operation cannot be fulfilled on disruptions.chaos.datadoghq.com "chaos-network-drop": the object has been modified; please apply your changes to the latest version and try again"
// Sadly this doesn't seem to be one of the errors checkable with a function from "k8s.io/apimachinery/pkg/api/errors"
// So we parse the error message directly
func IsModifiedError(err error) bool {
	return strings.Contains(err.Error(), "please apply your changes to the latest version and try again")
}

// TimeToCreatePods takes the DisruptionTriggers field from a Disruption spec, along with the time.Time at which that disruption was created
// It returns the earliest time.Time at which the chaos-controller should begin creating chaos pods, given the specified DisruptionTriggers
func TimeToCreatePods(triggers chaosv1beta1.DisruptionTriggers, creationTimestamp time.Time) time.Time {
	if triggers.IsZero() {
		return creationTimestamp
	}

	if triggers.CreatePods.IsZero() {
		return creationTimestamp
	}

	var noPodsBefore time.Time

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.CreatePods.NotBefore.IsZero() {
		noPodsBefore = triggers.CreatePods.NotBefore.Time
	}

	if triggers.CreatePods.Offset.Duration() > 0 {
		noPodsBefore = creationTimestamp.Add(triggers.CreatePods.Offset.Duration())
	}

	if creationTimestamp.After(noPodsBefore) {
		return creationTimestamp
	}

	return noPodsBefore
}

// TimeToInject takes the DisruptionTriggers field from a Disruption spec, along with the time.Time at which that disruption was created
// It returns the earliest time.Time at which chaos pods should inject into their targets, given the specified DisruptionTriggers
func TimeToInject(triggers chaosv1beta1.DisruptionTriggers, creationTimestamp time.Time) time.Time {
	if triggers.IsZero() {
		return creationTimestamp
	}

	if triggers.Inject.IsZero() {
		return TimeToCreatePods(triggers, creationTimestamp)
	}

	var notInjectedBefore time.Time

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.Inject.NotBefore.IsZero() {
		notInjectedBefore = triggers.Inject.NotBefore.Time
	}

	if triggers.Inject.Offset.Duration() > 0 {
		// We measure the offset from the latter of two timestamps: creationTimestamp of the disruption, and spec.trigger.createPods
		notInjectedBefore = TimeToCreatePods(triggers, creationTimestamp).Add(triggers.Inject.Offset.Duration())
	}

	if creationTimestamp.After(notInjectedBefore) {
		return creationTimestamp
	}

	return notInjectedBefore
}

// CalculateRemainingDurationOfDisruption of a disruption
func CalculateRemainingDurationOfDisruption(instance chaosv1beta1.Disruption) time.Duration {
	return calculateDeadline(
		instance.Spec.Duration.Duration(),
		TimeToInject(instance.Spec.Triggers, instance.ObjectMeta.CreationTimestamp.Time),
	)
}

// TransformCloudSpecToHostsSpec from a cloud spec disruption, get all ip ranges of services provided and transform them into a list of hosts spec
func TransformCloudSpecToHostsSpec(cloudManager cloudservice.CloudServicesProvidersManager, cloudSpec *chaosv1beta1.NetworkDisruptionCloudSpec) ([]chaosv1beta1.NetworkDisruptionHostSpec, error) {
	var hosts []chaosv1beta1.NetworkDisruptionHostSpec

	clouds := cloudSpec.TransformToCloudMap()

	for cloudName, serviceList := range clouds {
		var serviceListNames []string

		for _, service := range serviceList {
			serviceListNames = append(serviceListNames, service.ServiceName)
		}

		ipRangesPerService, err := cloudManager.GetServicesIPRanges(cloudtypes.CloudProviderName(cloudName), serviceListNames)
		if err != nil {
			return nil, err
		}

		for _, serviceSpec := range serviceList {
			for _, ipRange := range ipRangesPerService[serviceSpec.ServiceName] {
				hosts = append(hosts, chaosv1beta1.NetworkDisruptionHostSpec{
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
