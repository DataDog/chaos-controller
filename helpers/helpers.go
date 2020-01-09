package helpers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/DataDog/chaos-fi-controller/types"
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
func GeneratePod(name string, pod *corev1.Pod, args []string, mode types.PodMode) *corev1.Pod {
	image, ok := os.LookupEnv(ChaosFailureInjectionImageVariableName)
	if !ok {
		image = "chaos-fi"
	}

	privileged := true
	hostPathDirectory := corev1.HostPathDirectory
	hostPathFile := corev1.HostPathFile
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pod.Namespace,
			Labels: map[string]string{
				types.PodModeLabel: string(mode),
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

// GetMatchingPods returns a PodList containing all pods matching the NetworkFailureInjection's label selector and namespace.
func GetMatchingPods(c client.Client, namespace string, selector labels.Set) (*corev1.PodList, error) {
	// We want to ensure we never run into the possibility of using an empty label selector
	if len(selector) < 1 || selector == nil {
		return nil, errors.New("selector can't be an empty set")
	}

	// Filter pods based on the nfi's label selector, and only consider those within the same namespace as the nfi
	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     namespace,
	}

	// Fetch pods from label selector
	pods := &corev1.PodList{}
	err := c.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// PickRandomPods returns a shuffled sub-slice with a size of n of the given slice
func PickRandomPods(n uint, pods []corev1.Pod) []corev1.Pod {
	// Copy slice to don't modify the given one
	list := append([]corev1.Pod(nil), pods...)

	// Shuffle the slice
	rand.Seed(time.Now().Unix())
	for i := len(list) - 1; i > 0; i-- {
		j := rand.Intn(i)
		list[i], list[j] = list[j], list[i]
	}

	// Return the whole shuffled slice if the requested size is greater than the size of the slice
	if int(n) > len(list) {
		return list
	}

	return list[:n]
}

// GetOwnedPods returns a list of pods owned by the given object
func GetOwnedPods(c client.Client, owner metav1.Object) (corev1.PodList, error) {
	// Get pods
	pods := corev1.PodList{}
	ownedPods := corev1.PodList{}
	err := c.List(context.Background(), &pods, &client.ListOptions{Namespace: owner.GetNamespace()})
	if err != nil {
		return ownedPods, err
	}

	// Check owner reference
	for _, pod := range pods.Items {
		if metav1.IsControlledBy(&pod, owner) {
			ownedPods.Items = append(ownedPods.Items, pod)
		}
	}

	return ownedPods, nil
}

// GetContainerdID gets the ID of the first container ID found in a Pod.
// It expects container ids to follow the format "containerd://<ID>".
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
