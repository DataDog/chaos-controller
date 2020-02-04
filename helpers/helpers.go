// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package helpers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/DataDog/chaos-fi-controller/types"
	"github.com/miekg/dns"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ChaosFailureInjectionImageVariableName is the name of the chaos failure injection image variable
	ChaosFailureInjectionImageVariableName = "CHAOS_FI_IMAGE"
)

// GeneratePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func GeneratePod(instanceName string, pod *corev1.Pod, args []string, mode types.PodMode, kind types.DisruptionKind) *corev1.Pod {
	image, ok := os.LookupEnv(ChaosFailureInjectionImageVariableName)
	if !ok {
		image = "chaos-fi"
	}

	privileged := true
	hostPathDirectory := corev1.HostPathDirectory
	hostPathFile := corev1.HostPathFile
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("chaos-%s-%s-", instanceName, mode),
			Namespace:    pod.Namespace,
			Labels: map[string]string{
				types.PodModeLabel:        string(mode),
				types.TargetPodLabel:      pod.Name,
				types.DisruptionKindLabel: string(kind),
			},
			Annotations: map[string]string{
				"datadoghq.com/local-dns-cache": "true",
			},
		},
		Spec: corev1.PodSpec{
			NodeName:      pod.Spec.NodeName,
			RestartPolicy: "Never",
			Containers: []corev1.Container{
				{
					Name:            "chaos-fi",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            args,
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							MountPath: "/run/containerd",
							Name:      "containerd",
						},
						corev1.VolumeMount{
							MountPath: "/mnt/proc",
							Name:      "proc",
						},
						corev1.VolumeMount{
							MountPath: "/mnt/sysrq",
							Name:      "sysrq",
						},
						corev1.VolumeMount{
							MountPath: "/mnt/sysrq-trigger",
							Name:      "sysrq-trigger",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "containerd",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/containerd",
							Type: &hostPathDirectory,
						},
					},
				},
				corev1.Volume{
					Name: "proc",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/proc",
							Type: &hostPathDirectory,
						},
					},
				},
				corev1.Volume{
					Name: "sysrq",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/proc/sys/kernel/sysrq",
							Type: &hostPathFile,
						},
					},
				},
				corev1.Volume{
					Name: "sysrq-trigger",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/proc/sysrq-trigger",
							Type: &hostPathFile,
						},
					},
				},
			},
		},
	}
}

// GetMatchingPods returns a pods list containing all pods matching the given label selector and namespace
func GetMatchingPods(c client.Client, namespace string, selector labels.Set) (*corev1.PodList, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if len(selector) < 1 || selector == nil {
		return nil, errors.New("selector can't be an empty set")
	}

	// filter pods based on the nfi's label selector, and only consider those within the same namespace as the nfi
	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     namespace,
	}

	// fetch pods from label selector
	pods := &corev1.PodList{}
	err := c.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// PickRandomPods returns a shuffled sub-slice with a size of n of the given slice
func PickRandomPods(n uint, pods []corev1.Pod) []corev1.Pod {
	// copy slice to don't modify the given one
	list := append([]corev1.Pod(nil), pods...)

	// shuffle the slice
	rand.Seed(time.Now().Unix())
	for i := len(list) - 1; i > 0; i-- {
		j := rand.Intn(i)
		list[i], list[j] = list[j], list[i]
	}

	// return the whole shuffled slice if the requested size is greater than the size of the slice
	if int(n) > len(list) {
		return list
	}

	return list[:n]
}

// GetOwnedPods returns a list of pods owned by the given object
func GetOwnedPods(c client.Client, owner metav1.Object, selector labels.Set) (corev1.PodList, error) {
	// prepare list options
	options := &client.ListOptions{Namespace: owner.GetNamespace()}
	if selector != nil {
		options.LabelSelector = selector.AsSelector()
	}

	// get pods
	pods := corev1.PodList{}
	ownedPods := corev1.PodList{}
	err := c.List(context.Background(), &pods, options)
	if err != nil {
		return ownedPods, err
	}

	// check owner reference
	for _, pod := range pods.Items {
		if metav1.IsControlledBy(&pod, owner) {
			ownedPods.Items = append(ownedPods.Items, pod)
		}
	}

	return ownedPods, nil
}

// GetContainerdID gets the ID of the first container ID found in a Pod
// It expects container ids to follow the format "containerd://<ID>"
func GetContainerdID(pod *corev1.Pod) (string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return "", fmt.Errorf("Missing container ids for pod '%s'", pod.Name)
	}

	containerID := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "containerd://")
	if len(containerID) != 2 {
		return "", fmt.Errorf("Unrecognized container ID format '%s', expecting 'containerd://<ID>'", pod.Status.ContainerStatuses[0].ContainerID)
	}

	return containerID[1], nil
}

// ResolveHost tries to resolve the given host
// it tries to resolve it as a CIDR, as a single IP, or as a hostname
// it returns a list of IP or an error if it fails to resolve the hostname
func ResolveHost(hosts []string) ([]*net.IPNet, error) {
	var ips []*net.IPNet

	for _, host := range hosts {
		// try to parse the given host as a CIDR
		_, ipnet, err := net.ParseCIDR(host)
		if err != nil {
			// if it fails, try to parse the given host as a single IP
			ip := net.ParseIP(host)
			if ip == nil {
				// if no IP has been parsed, fallback on a hostname
				// and try to resolve it by using the container resolv.conf file
				dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
				if err != nil {
					return nil, fmt.Errorf("can't read resolve.conf file: %w", err)
				}
				dnsClient := dns.Client{}
				dnsMessage := dns.Msg{}
				dnsMessage.SetQuestion(host+".", dns.TypeA)
				response, _, err := dnsClient.Exchange(&dnsMessage, dnsConfig.Servers[0]+":53")
				if err != nil {
					return nil, fmt.Errorf("can't resolve the given hostname %s: %w", host, err)
				}

				// append all the records returned by the resolver for this hostname
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
					return nil, fmt.Errorf("the given IP (%s) seems to be an IPv6, aborting", host)
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
			return nil, fmt.Errorf("failed to resolve the given host: %s", host)
		}
	}

	return ips, nil
}

// ContainsString returns true if the given slice contains the given string,
// or returns false otherwise
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString removes the given string from the given slice,
// returning a new slice
func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
