// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cgroup

// This entire file mostly copied from the datadog-agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/env"
	"go.uber.org/zap"
)

var (
	containerRe = regexp.MustCompile("[0-9a-f]{64}|[0-9a-f]{8}(-[0-9a-f]{4}){4}")
)

// ContainerCgroup is a structure that stores paths and mounts for a cgroup.
// It provides several methods for collecting stats about the cgroup using the
// paths and mounts metadata.
type ContainerCgroup struct {
	ContainerID string
	Pids        []int32
	Paths       map[string]string
}

// hostProc returns the location of a host's procfs. This can and will be
// overridden when running inside a container.
func hostProc(procRoot string, combineWith ...string) string {
	parts := append([]string{procRoot}, combineWith...)
	return filepath.Join(parts...)
}

// readCgroupsForPath reads the cgroups from a /proc/$pid/cgroup path.
func readCgroupsForPath(pidCgroupPath string, log *zap.SugaredLogger) (string, map[string]string, error) {
	f, err := os.Open(pidCgroupPath) //nolint:gosec

	if os.IsNotExist(err) {
		log.Debugf("cgroup path '%s' could not be read: %s", pidCgroupPath, err)
		return "", nil, nil
	} else if err != nil {
		log.Debugf("cgroup path '%s' could not be read: %s", pidCgroupPath, err)
		return "", nil, err
	}
	defer f.Close() //nolint:gosec

	return parseCgroupPaths(f, log)
}

func containerIDFromCgroup(cgroup string) (string, bool) {
	sp := strings.SplitN(cgroup, ":", 3)

	if len(sp) < 3 {
		return "", false
	}

	matches := containerRe.FindAllString(sp[2], -1)
	if matches == nil {
		return "", false
	}

	return matches[len(matches)-1], true
}

// parseCgroupPaths parses out the cgroup paths from a /proc/$pid/cgroup file.
// The file format will be something like:
//
// 11:net_cls:/kubepods/besteffort/pod2baa3444-4d37-11e7-bd2f-080027d2bf10/47fc31db38b4fa0f4db44b99d0cad10e3cd4d5f142135a7721c1c95c1aadfb2e
// 10:freezer:/kubepods/besteffort/pod2baa3444-4d37-11e7-bd2f-080027d2bf10/47fc31db38b4fa0f4db44b99d0cad10e3cd4d5f142135a7721c1c95c1aadfb2e
// 9:cpu,cpuacct:/kubepods/besteffort/pod2baa3444-4d37-11e7-bd2f-080027d2bf10/47fc31db38b4fa0f4db44b99d0cad10e3cd4d5f142135a7721c1c95c1aadfb2e
// 8:memory:/kubepods/besteffort/pod2baa3444-4d37-11e7-bd2f-080027d2bf10/47fc31db38b4fa0f4db44b99d0cad10e3cd4d5f142135a7721c1c95c1aadfb2e
// 7:blkio:/kubepods/besteffort/pod2baa3444-4d37-11e7-bd2f-080027d2bf10/47fc31db38b4fa0f4db44b99d0cad10e3cd4d5f142135a7721c1c95c1aadfb2e
//
// Returns the common containerID and a mapping of target => path
// If any line doesn't have a valid container ID we will return an empty string and an empty slice of paths
func parseCgroupPaths(r io.Reader, log *zap.SugaredLogger) (string, map[string]string, error) {
	var containerID string

	paths := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		l := scanner.Text()
		cID, ok := containerIDFromCgroup(l)

		if !ok {
			log.Debugf("could not parse container id from path '%s'", l)
		}

		if containerID == "" {
			// Take the first valid containerID
			containerID = cID
		}

		sp := strings.SplitN(l, ":", 3)

		if len(sp) < 3 {
			continue
		}
		// Target can be comma-separate values like cpu,cpuacct
		tsp := strings.Split(sp[1], ",")

		for _, target := range tsp {
			if len(sp[2]) > 1 && sp[2] != "/docker" { // if the path is only one character it's the root cgroup
				paths[target] = sp[2]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", nil, err
	}

	// if we haven't picked up a container id from any cgroup, then we don't care about the paths either
	if containerID == "" {
		paths = make(map[string]string)
	}

	// In Ubuntu Xenial, we've encountered containers with no `cpu`
	_, cpuok := paths["cpu"]
	cpuacct, cpuacctok := paths["cpuacct"]

	if !cpuok && cpuacctok {
		paths["cpu"] = cpuacct
	}

	return containerID, paths, nil
}

// ScrapeAllCgroups returns ContainerCgroup for every container that's in a Cgroup.
// This version iterates on /{host/}proc to retrieve processes out of the namespace.
// We return as a map[containerID]Cgroup for easy look-up.
func ScrapeAllCgroups(log *zap.SugaredLogger) (map[string]*ContainerCgroup, error) {
	cgs := make(map[string]*ContainerCgroup)

	// Opening proc dir
	procMount, ok := os.LookupEnv(env.InjectorMountProc)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountProc)
	}

	procDir, err := os.Open(hostProc(procMount))

	if err != nil {
		return cgs, err
	}

	defer procDir.Close() //nolint:gosec
	dirNames, err := procDir.Readdirnames(-1)

	if err != nil {
		return cgs, err
	}

	for _, dirName := range dirNames {
		pid, err := strconv.ParseInt(dirName, 10, 32)

		if err != nil {
			continue
		}

		cgPath := hostProc(procMount, dirName, "cgroup")
		containerID, paths, err := readCgroupsForPath(cgPath, log)

		// Checking if it's a container cgroup. With CRIO and systemd cgroup manager
		// we can encounter hierarchies like:
		// /usr/libexec/crio/conmon -b /var/run/containers/storage/overlay-containers/abbfba09988 -> dedicated cgroup containing containerID
		//   \_ /pause -> real container cgroup
		// However a real container should always have a 'freezer' cgroup
		if containerID == "" {
			continue
		}

		mP, mFound := paths["memory"]
		fP, fFound := paths["freezer"]

		if !fFound || !mFound || mP != fP {
			log.Debugf("skipping cgroup from pid: %d - does not appear to be a container: memory path: %s, freezer path: %s", pid, mP, fP)
			continue
		}

		if err != nil {
			log.Infof("error reading cgroup paths %s: %s", cgPath, err)
			continue
		}

		if cg, ok := cgs[containerID]; ok {
			cg.Pids = append(cg.Pids, int32(pid))
		} else {
			cgs[containerID] = &ContainerCgroup{
				ContainerID: containerID,
				Pids:        []int32{int32(pid)},
				Paths:       paths,
			}
		}
	}

	return cgs, nil
}
