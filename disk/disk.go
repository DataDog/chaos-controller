// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package disk

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Informer represents a disk informer giving information about
// the device
type Informer interface {
	Major() int
	Minor() int
	Source() string
}

type disk struct {
	major  int
	minor  int
	source string
}

// FromPath returns a disk informer from the given path
func FromPath(path string) (Informer, error) {
	return fromPath(path, "/sys/dev/block")
}

func fromPath(path, sysfs string) (Informer, error) {
	// ensure the file exists before going further
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// get host path device
	df := exec.Command("df", "--output=source", path)

	out, err := df.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing df command on %s: %w", path, err)
	}

	// parse df output
	lines := []string{}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading df output: %w", err)
	}

	// we expect 2 lines here, the first one being the header
	if len(lines) != 2 {
		return nil, fmt.Errorf("unexpected df output: %v", lines)
	}

	// get the second line (and get rid of the header)
	device := lines[1]

	// get device major and minor identifiers
	ls := exec.Command("ls", "-lH", device)

	out, err = ls.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing ls command: %w\noutput: %s", err, out)
	}

	// parse ls output, format is like:
	//	<permissions> <size> <user> <group> <major>, <minor> <month> <day> <time> <path>
	// example:
	//	# ls -lH /dev/sda1
	//	brw-rw---- 1 root disk 8, 1 Jul  7 09:51 /dev/sda1
	lsParsed := strings.Fields(string(out))
	if len(lsParsed) < 10 || !strings.HasPrefix(lsParsed[0], "b") || !strings.HasSuffix(lsParsed[4], ",") {
		return nil, fmt.Errorf("unexpected ls output: %s", out)
	}

	major, minor, err := parseDeviceID(strings.TrimSuffix(lsParsed[4], ",") + ":" + lsParsed[5])
	if err != nil {
		return nil, err
	}

	major, minor, err = wholeDevice(sysfs, major, minor)
	if err != nil {
		return nil, err
	}

	return disk{
		major:  major,
		minor:  minor,
		source: device,
	}, nil
}

// wholeDevice resolves partitions to their parent disk, as blkcg requires a whole device.
func wholeDevice(sysfs string, major, minor int) (int, int, error) {
	device, err := filepath.EvalSymlinks(filepath.Join(sysfs, fmt.Sprintf("%d:%d", major, minor)))
	if err != nil {
		return 0, 0, fmt.Errorf("error resolving block device %d:%d: %w", major, minor, err)
	}

	if _, err := os.Stat(filepath.Join(device, "partition")); os.IsNotExist(err) {
		return major, minor, nil
	} else if err != nil {
		return 0, 0, fmt.Errorf("error checking block device partition: %w", err)
	}

	parent := filepath.Join(filepath.Dir(device), "dev")

	out, err := os.ReadFile(parent)
	if err != nil {
		return 0, 0, fmt.Errorf("error reading parent block device %s: %w", parent, err)
	}

	major, minor, err = parseDeviceID(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, 0, fmt.Errorf("error reading parent block device %s: %w", parent, err)
	}

	return major, minor, nil
}

func parseDeviceID(id string) (int, int, error) {
	fields := strings.Split(id, ":")
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected block device identifier: %q", id)
	}

	major, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("unexpected device major identifier (%s): %w", fields[0], err)
	}

	minor, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("unexpected device minor identifier (%s): %w", fields[1], err)
	}

	return int(major), int(minor), nil
}

func (d disk) Major() int {
	return d.major
}

func (d disk) Minor() int {
	return d.minor
}

func (d disk) Source() string {
	return d.source
}
