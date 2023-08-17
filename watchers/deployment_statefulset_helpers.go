// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

func ContainersChanged(oldPodSpec, newPodSpec *corev1.PodSpec, log *zap.SugaredLogger) (bool, map[string]string, map[string]string, error) {
	oldInitContainersHash, oldContainersHash, err := HashPodSpec(oldPodSpec)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to hash old pod spec: %w", err)
	}

	newInitContainersHash, newContainersHash, err := HashPodSpec(newPodSpec)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to hash new pod spec: %w", err)
	}

	if HashesChanged(oldInitContainersHash, newInitContainersHash, log) ||
		HashesChanged(oldContainersHash, newContainersHash, log) {
		return true, newInitContainersHash, newContainersHash, nil
	}

	return false, oldInitContainersHash, oldContainersHash, nil
}

func Hash(container *corev1.Container) (string, error) {
	containerJSON, err := json.Marshal(*container)
	if err != nil {
		return "", fmt.Errorf("error serializing instance spec: %w", err)
	}

	return hex.EncodeToString(md5.New().Sum(containerJSON)), nil
}

func HashPodSpec(podSpec *corev1.PodSpec) (map[string]string, map[string]string, error) {
	initContainersHash, err := HashContainerList(&podSpec.InitContainers)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to hash initContainers: %w", err)
	}

	containersHash, err := HashContainerList(&podSpec.Containers)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to hash containers: %w", err)
	}

	return initContainersHash, containersHash, nil
}

func HashContainerList(containers *[]corev1.Container) (map[string]string, error) {
	containerHashes := make(map[string]string, len(*containers))

	for _, container := range *containers {
		hash, err := Hash(&container)
		if err != nil {
			return nil, fmt.Errorf("error hashing container %s: %v", container.Name, err)
		}

		containerHashes[container.Name] = hash
	}

	return containerHashes, nil
}

func HashesChanged(oldHashes, newHashes map[string]string, log *zap.SugaredLogger) bool {
	// Check if any hashes in oldHashes don't match the ones in newHashes
	for k, v := range oldHashes {
		newVal, ok := newHashes[k]
		if !ok {
			log.Infof("container %s is missing in new hashes", k)
			return true
		}

		if newVal != v {
			log.Infof("hash for container %s has changed", k)
			return true
		}
	}

	// Check if any hashes in newHashes don't match the ones in oldHashes
	for k, v := range newHashes {
		oldVal, ok := oldHashes[k]
		if !ok {
			log.Infof("container %s is missing in old hashes", k)
			return true
		}

		if oldVal != v {
			log.Infof("hash for container %s has changed", k)
			return true
		}
	}

	return false
}
